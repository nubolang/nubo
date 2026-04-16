package plug

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

// HandlerFunc is implemented by plugin authors to handle a named RPC method.
type HandlerFunc func(ctx *Ctx) error

// AuthValidatorFunc is called on every incoming TCP connection with the token
// the host presented.
type AuthValidatorFunc func(token string) bool

// Map is a convenience type for RPC request/response data.
type Map map[string]any

// Ctx carries a single RPC invocation.
type Ctx struct {
	context.Context

	// Method is the name of the called RPC method.
	Method string

	raw []byte
	fw  *frameWriter
	id  uint32
}

// Bind decodes the msgpack request params into v.
func (c *Ctx) Bind(v any) error {
	return Unmarshal(c.raw, v)
}

// Send encodes v as msgpack and writes it as the response.
func (c *Ctx) Send(v any) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}
	return c.fw.write(&frame{ID: c.id, Result: data})
}

// Fail sends an error string as the response.
func (c *Ctx) Fail(err error) error {
	return c.fw.write(&frame{ID: c.id, Err: err.Error()})
}

// AppOption configures an App.
type AppOption func(*App)

// WithAuth enables token authentication on TCP connections using a static
// shared secret. The plugin rejects any connection whose token does not match.
func WithAuth(token string) AppOption {
	return WithAuthValidator(func(presented string) bool {
		return presented == token
	})
}

// WithAuthValidator enables token authentication on TCP connections with a
// custom validation function. The function receives the token string that the
// host presented and must return true to accept the connection.
func WithAuthValidator(fn AuthValidatorFunc) AppOption {
	return func(a *App) {
		a.authEnabled = true
		a.authValidator = fn
	}
}

// App is the plugin-side RPC server. Each plugin binary holds one App.
type App struct {
	mu            sync.RWMutex
	handlers      map[string]HandlerFunc
	fw            *frameWriter // stdio only
	in            io.Reader    // stdio only
	listener      net.Listener // tcp only
	authEnabled   bool
	authValidator AuthValidatorFunc
}

// Create returns a new App wired to os.Stdin and os.Stdout (stdio transport).
func Create() *App {
	return &App{
		handlers: make(map[string]HandlerFunc),
		fw:       &frameWriter{w: os.Stdout},
		in:       os.Stdin,
	}
}

// CreateTCP returns a new App that listens on addr (e.g. ":9000") and serves
// each accepted TCP connection independently.
func CreateTCP(addr string, opts ...AppOption) (*App, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("plug: listen on %s: %w", addr, err)
	}
	a := &App{
		handlers: make(map[string]HandlerFunc),
		listener: ln,
	}
	for _, o := range opts {
		o(a)
	}
	return a, nil
}

// Handler registers fn for the given method name.
func (a *App) Handler(method string, fn HandlerFunc) {
	a.mu.Lock()
	a.handlers[method] = fn
	a.mu.Unlock()
}

// Start blocks and serves incoming RPC frames until the transport is closed.
func (a *App) Start() {
	if a.listener != nil {
		// Announce the bound address to the host via a single stdout line.
		fmt.Fprintf(os.Stdout, "PLUG_TCP_ADDR=%s\n", a.listener.Addr().String())
		_ = os.Stdout.Sync()
		a.serveTCP()
		return
	}
	a.serveReader(a.in, a.fw)
}

// serveTCP accepts TCP connections and spawns a goroutine for each.
func (a *App) serveTCP() {
	defer a.listener.Close()
	for {
		conn, err := a.listener.Accept()
		if err != nil {
			log.Printf("plug: accept: %v", err)
			return
		}
		go a.handleConn(conn)
	}
}

// handleConn runs the optional auth handshake then serves a TCP connection.
func (a *App) handleConn(conn net.Conn) {
	defer conn.Close()
	if a.authEnabled {
		if !a.recvAuthHandshake(conn) {
			log.Printf("plug: auth rejected from %s", conn.RemoteAddr())
			return
		}
	}
	fw := &frameWriter{w: conn}
	a.serveReader(conn, fw)
}

// recvAuthHandshake reads the host token, runs it through the validator, and
// replies "OK" or "NO".
func (a *App) recvAuthHandshake(conn net.Conn) bool {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return false
	}
	n := binary.BigEndian.Uint32(hdr)
	tok := make([]byte, n)
	if _, err := io.ReadFull(conn, tok); err != nil {
		return false
	}
	if a.authValidator == nil || !a.authValidator(string(tok)) {
		_, _ = conn.Write([]byte("NO"))
		return false
	}
	_, _ = conn.Write([]byte("OK"))
	return true
}

// serveReader reads frames from r and dispatches them using fw for replies.
func (a *App) serveReader(r io.Reader, fw *frameWriter) {
	for {
		f, err := readFrame(r)
		if err != nil {
			if err != io.EOF {
				log.Printf("plug: read: %v", err)
			}
			return
		}
		go a.dispatch(f, fw)
	}
}

func (a *App) dispatch(f *frame, fw *frameWriter) {
	a.mu.RLock()
	fn, ok := a.handlers[f.Method]
	a.mu.RUnlock()

	if !ok {
		_ = fw.write(&frame{
			ID:  f.ID,
			Err: fmt.Sprintf("unknown method: %s", f.Method),
		})
		return
	}

	ctx := &Ctx{
		Context: context.Background(),
		Method:  f.Method,
		raw:     f.Params,
		fw:      fw,
		id:      f.ID,
	}

	if err := fn(ctx); err != nil {
		_ = fw.write(&frame{ID: f.ID, Err: err.Error()})
	}
}
