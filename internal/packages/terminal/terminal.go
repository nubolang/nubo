// Package terminal provides an exclusively owned interactive stdin/stdout session.
package terminal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/muesli/cancelreader"
	"github.com/nubolang/nubo/internal/execution"
	"github.com/nubolang/nubo/language"
	"golang.org/x/term"
)

var ownership sync.Mutex
var active bool

type session struct {
	keyEvents                    bool
	keyFlags                     int
	keyPushed                    bool
	down                         map[string]bool
	queued                       []any
	closeOnce                    sync.Once
	closeErr                     error
	stopOnce                     sync.Once
	mu                           sync.Mutex
	input                        cancelreader.CancelReader
	original                     *term.State
	restoreOutput                func() error
	resources                    *execution.Resources
	unregister                   func()
	ctx                          context.Context
	ctrlC                        *language.Function
	closed                       bool
	raw, alternate, hidden       bool
	bytes                        chan byte
	done, readerDone, signalDone chan struct{}
	signals                      chan os.Signal
	buffer                       []byte
	partial                      time.Time
	width, height                int
	readErr                      error
}

func open(ctx context.Context) (*session, error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil, errors.New("terminal sessions require Linux or macOS; a safely cancellable Windows console reader is not yet supported")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil, fmt.Errorf("terminal.TerminalSession requires terminal stdin and stdout")
	}
	resources := execution.From(ctx)
	if resources == nil {
		return nil, fmt.Errorf("terminal.TerminalSession requires an active runtime execution")
	}
	ownership.Lock()
	defer ownership.Unlock()
	if active {
		return nil, fmt.Errorf("a terminal session is already open")
	}
	original, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	restoreOutput, err := prepareOutput()
	if err != nil {
		return nil, err
	}
	reader, err := cancelreader.NewReader(os.Stdin)
	if err != nil {
		_ = restoreOutput()
		return nil, err
	}
	s := &session{input: reader, original: original, restoreOutput: restoreOutput, resources: resources, ctx: ctx, bytes: make(chan byte, 1024), done: make(chan struct{}), readerDone: make(chan struct{}), signalDone: make(chan struct{}), signals: make(chan os.Signal, 4), buffer: make([]byte, 0, 128)}
	s.width, s.height, _ = term.GetSize(int(os.Stdout.Fd()))
	s.unregister, err = resources.Add(s.close)
	if err != nil {
		reader.Close()
		restoreOutput()
		return nil, err
	}
	active = true
	signal.Notify(s.signals, os.Interrupt, syscall.SIGTERM)
	go s.read()
	go func() {
		defer close(s.signalDone)
		for {
			select {
			case sig := <-s.signals:
				if sig == os.Interrupt {
					s.interrupt()
				} else {
					go resources.Stop(fmt.Errorf("terminal terminated: %s", sig))
					return
				}
			case <-s.done:
				return
			}
		}
	}()
	return s, nil
}

func (s *session) interrupt() {
	s.mu.Lock()
	fn := s.ctrlC
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return
	}
	if fn == nil {
		s.stopOnce.Do(func() { go s.resources.Stop(errors.New("terminal interrupted")) })
		return
	}
	s.resources.Enqueue(func() error { _, err := fn.Data(s.ctx, nil); return err })
}

func (s *session) read() {
	defer close(s.readerDone)
	defer close(s.bytes)
	var buffer [256]byte
	for {
		count, err := s.input.Read(buffer[:])
		for _, b := range buffer[:count] {
			if b == 3 {
				s.interrupt()
				continue
			}
			select {
			case s.bytes <- b:
			case <-s.done:
				return
			}
		}
		if err != nil {
			s.mu.Lock()
			s.readErr = err
			s.mu.Unlock()
			return
		}
	}
}

func (s *session) close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		close(s.done)
		signal.Stop(s.signals)
		s.mu.Unlock()
		s.input.Cancel()
		<-s.readerDone
		<-s.signalDone
		err := s.input.Close()
		s.mu.Lock()
		if s.keyPushed {
			_, e := os.Stdout.WriteString("\x1b[<u\x1b[?1004l")
			err = errors.Join(err, e)
			s.keyEvents = false
			clear(s.down)
		}
		if s.hidden {
			_, e := os.Stdout.WriteString("\x1b[?25h")
			err = errors.Join(err, e)
		}
		if s.alternate {
			_, e := os.Stdout.WriteString("\x1b[?1049l")
			err = errors.Join(err, e)
		}
		err = errors.Join(err, term.Restore(int(os.Stdin.Fd()), s.original), s.restoreOutput())
		s.mu.Unlock()
		s.unregister()
		ownership.Lock()
		active = false
		ownership.Unlock()
		s.closeErr = err
	})
	return s.closeErr
}

func (s *session) inputError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readErr != nil {
		return fmt.Errorf("terminal input ended: %w", s.readErr)
	}
	return errors.New("terminal input ended")
}

func (s *session) poll(ms int64) (any, error) {
	if len(s.queued) > 0 {
		event := s.queued[0]
		s.queued = s.queued[1:]
		return event, nil
	}
	return s.pollInput(ms)
}

func (s *session) pollInput(ms int64) (any, error) {
	if ms < 0 || ms > int64((1<<63-1)/time.Millisecond) {
		return nil, fmt.Errorf("timeout must be nonnegative milliseconds")
	}
	deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
	for {
		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()
		if closed {
			return nil, fmt.Errorf("terminal session is closed")
		}
		if len(s.buffer) >= 128 {
			s.buffer = s.buffer[:0]
			return keyEvent{Key: "unknown"}.value(), nil
		}
		event, count := decode(s.buffer, !s.partial.IsZero() && time.Since(s.partial) >= 30*time.Millisecond)
		if count > 0 {
			copy(s.buffer, s.buffer[count:])
			s.buffer = s.buffer[:len(s.buffer)-count]
			s.partial = time.Time{}
			if event.Key == "protocol" {
				s.mu.Lock()
				s.keyFlags = event.flags
				s.mu.Unlock()
				return language.Nil, nil
			}
			if event.Key == "focus" {
				if event.Type == "out" {
					s.mu.Lock()
					clear(s.down)
					s.mu.Unlock()
				}
				continue
			}
			s.mu.Lock()
			if (s.keyEvents || s.keyPushed) && event.protocol && event.identity != "" {
				if event.Type == "release" {
					delete(s.down, event.identity)
				} else {
					s.down[event.identity] = true
				}
			}
			s.mu.Unlock()
			if event.Ctrl && (event.Text == "c" || event.Key == "c") {
				if event.Type != "release" {
					s.interrupt()
				}
				continue
			}
			return event.value(), nil
		}
		if len(s.buffer) > 0 && s.partial.IsZero() {
			s.partial = time.Now()
		}
		select {
		case b, ok := <-s.bytes:
			if !ok {
				return nil, s.inputError()
			}
			s.buffer = append(s.buffer, b)
			continue
		default:
		}
		if ms == 0 || !time.Now().Before(deadline) {
			return language.Nil, nil
		}
		wait := time.Until(deadline)
		if len(s.buffer) > 0 {
			remaining := 30*time.Millisecond - time.Since(s.partial)
			if remaining > 0 && remaining < wait {
				wait = remaining
			}
		}
		timer := time.NewTimer(wait)
		select {
		case b, ok := <-s.bytes:
			timer.Stop()
			if !ok {
				return nil, s.inputError()
			}
			s.buffer = append(s.buffer, b)
		case <-s.done:
			timer.Stop()
			return nil, errors.New("terminal session is closed")
		case <-timer.C:
		}
	}
}

func (s *session) onCtrlC(handler language.Object) error {
	var fn *language.Function
	if handler != language.Nil {
		var ok bool
		fn, ok = handler.(*language.Function)
		if !ok || len(fn.ArgTypes) != 0 {
			return errors.New("Ctrl+C handler must be a zero-argument function or nil")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctrlC = fn
	return nil
}

func (s *session) size() (map[string]any, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return nil, err
	}
	return map[string]any{"width": width, "height": height}, nil
}

func (s *session) pollResize() (any, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return nil, err
	}
	if width == s.width && height == s.height {
		return language.Nil, nil
	}
	s.width, s.height = width, height
	return map[string]any{"width": width, "height": height}, nil
}

func (s *session) setRaw(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("terminal session is closed")
	}
	if s.keyPushed {
		return errors.New("disable key events before changing raw mode")
	}
	if enabled == s.raw {
		return nil
	}
	var err error
	if enabled {
		_, err = term.MakeRaw(int(os.Stdin.Fd()))
	} else {
		err = term.Restore(int(os.Stdin.Fd()), s.original)
	}
	if err == nil {
		s.raw = enabled
	}
	return err
}

func (s *session) setAlternateScreen(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("terminal session is closed")
	}
	if s.keyPushed {
		return errors.New("disable key events before changing screen mode")
	}
	sequence := "\x1b[?1049l"
	if enabled {
		sequence = "\x1b[?1049h"
	}
	s.alternate = enabled
	_, err := os.Stdout.WriteString(sequence)
	return err
}

func (s *session) setCursorVisible(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("terminal session is closed")
	}
	sequence := "\x1b[?25l"
	if enabled {
		sequence = "\x1b[?25h"
	}
	s.hidden = !enabled
	_, err := os.Stdout.WriteString(sequence)
	return err
}

func (s *session) write(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("terminal session is closed")
	}
	_, err := os.Stdout.WriteString(text)
	return err
}

func (s *session) isKeyDown(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false, errors.New("terminal session is closed")
	}
	return s.keyEvents && s.down[key], nil
}
