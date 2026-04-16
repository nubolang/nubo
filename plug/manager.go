package plug

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

// Plugin represents a running plugin subprocess.
type Plugin struct {
	// Path is the absolute directory from which the plugin was loaded.
	Path string

	// DisplayName is the base directory name, used for logging and error messages.
	DisplayName string

	cfg    PlugConfig
	cmd    *exec.Cmd
	fw     *frameWriter
	reader io.Reader
	conn   net.Conn // non-nil for tcp transport

	mu      sync.Mutex
	pending map[uint32]chan *frame
	seq     uint32
}

// Call invokes method on the plugin with msgpack-serialised params and returns
// the raw msgpack result bytes. Use Unmarshal to decode.
func (p *Plugin) Call(ctx context.Context, method string, params any) ([]byte, error) {
	data, err := Marshal(params)
	if err != nil {
		return nil, err
	}

	id := atomic.AddUint32(&p.seq, 1)
	ch := make(chan *frame, 1)

	p.mu.Lock()
	p.pending[id] = ch
	p.mu.Unlock()

	if err := p.fw.write(&frame{ID: id, Method: method, Params: data}); err != nil {
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.Err != "" {
			return nil, fmt.Errorf("plug[%s]: %s", p.DisplayName, resp.Err)
		}
		return resp.Result, nil
	}
}

// CallInto is a convenience wrapper that decodes the result into v.
func (p *Plugin) CallInto(ctx context.Context, method string, params, v any) error {
	raw, err := p.Call(ctx, method, params)
	if err != nil {
		return err
	}
	return Unmarshal(raw, v)
}

// Stop kills the plugin subprocess and closes any open TCP connection.
func (p *Plugin) Stop() error {
	if p.conn != nil {
		_ = p.conn.Close()
	}
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

// readLoop runs in a goroutine and fans incoming frames out to pending callers.
func (p *Plugin) readLoop() {
	for {
		f, err := readFrame(p.reader)
		if err != nil {
			if err != io.EOF {
				log.Printf("plug[%s]: read: %v", p.DisplayName, err)
			}
			p.mu.Lock()
			for id, ch := range p.pending {
				ch <- &frame{ID: id, Err: "plugin disconnected"}
				delete(p.pending, id)
			}
			p.mu.Unlock()
			return
		}

		p.mu.Lock()
		ch, ok := p.pending[f.ID]
		if ok {
			delete(p.pending, f.ID)
		}
		p.mu.Unlock()

		if ok {
			ch <- f
		}
	}
}

// Manager owns a set of plugins keyed by absolute directory path.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]*Plugin
}

var manager *Manager

// GetManager returns an the Manager.
func GetManager() *Manager {
	if manager == nil {
		manager = &Manager{plugins: make(map[string]*Plugin)}
	}

	return manager
}

// PluginOption configures how the Manager connects to a plugin at load time.
type PluginOption func(*pluginOptions)

type pluginOptions struct {
	// authToken is sent to the plugin during the TCP handshake when non-empty.
	// Leave empty if the plugin does not require authentication.
	authToken string
}

// WithToken supplies the shared secret sent to a TCP plugin during the
// authentication handshake.
func WithToken(token string) PluginOption {
	return func(o *pluginOptions) {
		o.authToken = token
	}
}

// Load reads _plug.yaml from path, validates the current OS is supported,
// builds the binary, and starts the plugin using the configured transport.
func (m *Manager) Load(path string, opts ...PluginOption) (*Plugin, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Fast path: already loaded.
	m.mu.RLock()
	if existing, ok := m.plugins[absPath]; ok {
		m.mu.RUnlock()
		return existing, nil
	}
	m.mu.RUnlock()

	cfgPath := filepath.Join(absPath, "_plug.yaml")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("plug: read config at %s: %w", cfgPath, err)
	}

	var cfg PlugConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("plug: parse config: %w", err)
	}

	// Reject early if this OS is not in the supported architecture list.
	if err := checkArchSupport(cfg); err != nil {
		return nil, err
	}

	if err := buildPlugin(absPath, cfg); err != nil {
		return nil, fmt.Errorf("plug: build at %q: %w", absPath, err)
	}

	po := &pluginOptions{}
	for _, o := range opts {
		o(po)
	}

	displayName := filepath.Base(absPath)
	binPath := resolveBinary(absPath, cfg)

	mode := strings.ToLower(strings.TrimSpace(cfg.Plugin.Transport.Mode))
	if mode == "" {
		mode = "stdio"
	}

	var p *Plugin
	switch mode {
	case "stdio":
		p, err = startStdio(absPath, binPath, cfg, displayName)
	case "tcp":
		p, err = startTCP(absPath, binPath, cfg, displayName, po)
	default:
		return nil, fmt.Errorf("plug: unknown transport mode %q (want stdio or tcp)", mode)
	}
	if err != nil {
		return nil, err
	}

	// Lock and double-check (TOCTOU).
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, dup := m.plugins[absPath]; dup {
		_ = p.Stop()
		return existing, nil
	}
	m.plugins[absPath] = p
	return p, nil
}

// startStdio launches the plugin and wires stdin/stdout as the transport.
func startStdio(base, binPath string, cfg PlugConfig, displayName string) (*Plugin, error) {
	cmd := exec.Command(binPath)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("plug: start %q: %w", displayName, err)
	}

	p := &Plugin{
		Path:        base,
		DisplayName: displayName,
		cfg:         cfg,
		cmd:         cmd,
		fw:          &frameWriter{w: stdin},
		reader:      stdout,
		pending:     make(map[uint32]chan *frame),
	}
	go p.readLoop()
	return p, nil
}

// startTCP launches the plugin binary and discovers its TCP address from the
// first line it writes to stdout ("PLUG_TCP_ADDR=<addr>"). This means the
// plugin can bind on :0 to let the OS choose a free port, and the address
// never needs to appear in _plug.yaml.
//
// After the address is discovered the manager dials it, performs the optional
// auth handshake (token supplied via WithToken on the host side, validated by
// WithAuth / WithAuthValidator on the plugin side), and starts the read loop.
func startTCP(base, binPath string, cfg PlugConfig, displayName string, po *pluginOptions) (*Plugin, error) {
	cmd := exec.Command(binPath)
	cmd.Stderr = os.Stderr

	// Pipe stdout so we can read the address announcement.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("plug: start %q: %w", displayName, err)
	}

	// Read the first line: "PLUG_TCP_ADDR=<host:port>"
	addr, err := readTCPAddrAnnouncement(stdout, displayName)
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}

	// Dial with retries – the listener is up (it wrote the announcement before
	// we read it), but the OS may need a moment under load.
	var conn net.Conn
	var dialErr error
	for _ = range 10 {
		conn, dialErr = net.DialTimeout("tcp", addr, time.Second)
		if dialErr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if dialErr != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("plug: connect to %q at %s: %w", displayName, addr, dialErr)
	}

	// Optional auth handshake.
	if po.authToken != "" {
		if err := sendAuthToken(conn, po.authToken); err != nil {
			_ = conn.Close()
			_ = cmd.Process.Kill()
			return nil, fmt.Errorf("plug: auth for %q: %w", displayName, err)
		}
	}

	p := &Plugin{
		Path:        base,
		DisplayName: displayName,
		cfg:         cfg,
		cmd:         cmd,
		fw:          &frameWriter{w: conn},
		reader:      conn,
		conn:        conn,
		pending:     make(map[uint32]chan *frame),
	}
	go p.readLoop()
	return p, nil
}

// readTCPAddrAnnouncement reads the single "PLUG_TCP_ADDR=<addr>" line that
// the plugin writes to its stdout immediately after binding its TCP listener.
func readTCPAddrAnnouncement(r io.Reader, displayName string) (string, error) {
	type result struct {
		addr string
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		sc := bufio.NewScanner(r)
		if sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			const prefix = "PLUG_TCP_ADDR="
			if strings.HasPrefix(line, prefix) {
				ch <- result{addr: strings.TrimPrefix(line, prefix)}
				return
			}
			ch <- result{err: fmt.Errorf("plug: unexpected announcement from %q: %q", displayName, line)}
			return
		}
		if err := sc.Err(); err != nil {
			ch <- result{err: fmt.Errorf("plug: reading announcement from %q: %w", displayName, err)}
			return
		}
		ch <- result{err: fmt.Errorf("plug: plugin %q closed stdout before writing PLUG_TCP_ADDR", displayName)}
	}()

	select {
	case res := <-ch:
		return res.addr, res.err
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("plug: timed out waiting for PLUG_TCP_ADDR from %q", displayName)
	}
}

// sendAuthToken writes a length-prefixed token over conn and waits for a
// 2-byte acknowledgement ("OK" or "NO").
func sendAuthToken(conn net.Conn, token string) error {
	tb := []byte(token)
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, uint32(len(tb)))
	if _, err := conn.Write(hdr); err != nil {
		return err
	}
	if _, err := conn.Write(tb); err != nil {
		return err
	}
	ack := make([]byte, 2)
	if _, err := io.ReadFull(conn, ack); err != nil {
		return err
	}
	if string(ack) != "OK" {
		return fmt.Errorf("authentication rejected by plugin")
	}
	return nil
}

// checkArchSupport returns an error if the current OS is not listed.
func checkArchSupport(cfg PlugConfig) error {
	if len(cfg.Plugin.Architecture) == 0 {
		return nil
	}
	current := runtime.GOOS
	for _, arch := range cfg.Plugin.Architecture {
		if strings.EqualFold(strings.TrimSpace(arch), current) {
			return nil
		}
	}
	return fmt.Errorf(
		"plug: current OS %q is not in the plugin's supported architecture list %v",
		current, cfg.Plugin.Architecture,
	)
}

// Get returns a loaded plugin by its directory path.
func (m *Manager) Get(path string) (*Plugin, bool) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[absPath]
	return p, ok
}

// Paths returns the absolute directory paths of all currently loaded plugins.
func (m *Manager) Paths() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	paths := make([]string, 0, len(m.plugins))
	for p := range m.plugins {
		paths = append(paths, p)
	}
	return paths
}

// StopAll kills every running plugin subprocess.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.plugins {
		_ = p.Stop()
	}
}

// buildPlugin resolves template variables and runs the build command.
func buildPlugin(base string, cfg PlugConfig) error {
	src := filepath.Join(base, cfg.Plugin.Source)
	dest := resolveBinary(base, cfg)

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	rawCmd := cfg.Plugin.Cmd
	rawCmd = strings.ReplaceAll(rawCmd, "{src}", src)
	rawCmd = strings.ReplaceAll(rawCmd, "{dest}", dest)

	parts := strings.Fields(rawCmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty build command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = base
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := os.Environ()
	cgoVal := "1"
	if !cfg.Plugin.CGO {
		cgoVal = "0"
	}
	cmd.Env = append(env, "CGO_ENABLED="+cgoVal)

	return cmd.Run()
}

// resolveBinary returns the absolute path to the compiled plugin binary.
func resolveBinary(base string, cfg PlugConfig) string {
	b := cfg.Plugin.Binary
	if filepath.IsAbs(b) {
		return b
	}
	return filepath.Join(base, b)
}
