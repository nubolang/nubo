package terminal

import (
	"context"
	_ "embed"
	"errors"
	"os"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
	"golang.org/x/term"
)

//go:embed package.nubo
var Source string

func Functions() map[string]any {
	return map[string]any{
		"isInputTerminal":  func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		"isOutputTerminal": func() bool { return term.IsTerminal(int(os.Stdout.Fd())) },
		"open": func(ctx context.Context) (*native.GoHandle, error) {
			s, err := open(ctx)
			if err != nil {
				return nil, err
			}
			return native.NewGoHandle(s), nil
		},
		"close": bindVoid((*session).close),
		"size":  bindSession((*session).size),
		"pollResize": bindSession(func(s *session) (any, error) {
			value, err := s.pollResize()
			return bridgeNil(value), err
		}),
		"pollKey": bindSessionArg(func(s *session, timeoutMs int64) (any, error) {
			value, err := s.poll(timeoutMs)
			return bridgeNil(value), err
		}),
		"setKeyEvents":       bindSessionArg((*session).setKeyEvents),
		"isKeyDown":          bindSessionArg((*session).isKeyDown),
		"setRaw":             bindVoidArg((*session).setRaw),
		"setAlternateScreen": bindVoidArg((*session).setAlternateScreen),
		"setCursorVisible":   bindVoidArg((*session).setCursorVisible),
		"write":              bindVoidArg((*session).write),
		// Keep the original Nubo closure, including its captured variables.
		"onCtrlC": n.Function(n.Describe(n.Arg("handle", language.TypeGoHandle), n.Arg("handler", n.TAny)),
			func(a *n.Args) (any, error) {
				s, err := sessionFromHandle(a.Get(0).(*native.GoHandle))
				if err != nil {
					return nil, err
				}
				return nil, s.onCtrlC(a.Get(1))
			}),
	}
}

func sessionFromHandle(handle *native.GoHandle) (*session, error) {
	if handle != nil {
		if s, ok := handle.RealData.(*session); ok && s != nil {
			return s, nil
		}
	}
	return nil, errors.New("invalid terminal session handle")
}

func bridgeNil(value any) any {
	if value == language.Nil {
		return nil
	}
	return value
}

func bindSession[R any](fn func(*session) (R, error)) func(*native.GoHandle) (R, error) {
	return func(handle *native.GoHandle) (R, error) {
		s, err := sessionFromHandle(handle)
		if err != nil {
			var zero R
			return zero, err
		}
		return fn(s)
	}
}

func bindSessionArg[A, R any](fn func(*session, A) (R, error)) func(*native.GoHandle, A) (R, error) {
	return func(handle *native.GoHandle, arg A) (R, error) {
		s, err := sessionFromHandle(handle)
		if err != nil {
			var zero R
			return zero, err
		}
		return fn(s, arg)
	}
}

func bindVoid(fn func(*session) error) func(*native.GoHandle) error {
	return func(handle *native.GoHandle) error {
		s, err := sessionFromHandle(handle)
		if err != nil {
			return err
		}
		return fn(s)
	}
}

func bindVoidArg[A any](fn func(*session, A) error) func(*native.GoHandle, A) error {
	return func(handle *native.GoHandle, arg A) error {
		s, err := sessionFromHandle(handle)
		if err != nil {
			return err
		}
		return fn(s, arg)
	}
}
