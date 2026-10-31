package interpreter

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"github.com/nubolang/nubo/language"
)

const maxHookDepth = 64

var errHookDepth = errors.New("getter/setter recursion limit exceeded")

type hookStackKey struct{}

type hookFrame struct {
	parent *hookFrame
	active atomic.Bool
}

func callHook(ctx context.Context, fn *language.Function, args []language.Object, namedArgs ...map[string]language.Object) (language.Object, error) {
	head, _ := ctx.Value(hookStackKey{}).(*hookFrame)
	depth := 0

	for frame := head; frame != nil; frame = frame.parent {
		if frame.active.Load() {
			depth++
		}
	}

	if depth >= maxHookDepth {
		return nil, runExc("getter/setter recursion limit exceeded (%d)", maxHookDepth).WithBase(errHookDepth).WithDebug(fn.Debug())
	}

	// Captured contexts may outlive the hook.
	frame := &hookFrame{parent: head}
	frame.active.Store(true)
	defer frame.active.Store(false)

	ctx = context.WithValue(ctx, hookStackKey{}, frame)
	if len(namedArgs) > 0 {
		return fn.Call(ctx, args, namedArgs...)
	}

	return fn.Data(ctx, args)
}

func callNamedFunction(ctx context.Context, fn *language.Function, name string, args []language.Object, namedArgs ...map[string]language.Object) (language.Object, error) {
	parts := strings.Split(name, ".")
	last := parts[len(parts)-1]

	if last == "__get__" || last == "__set__" {
		return callHook(ctx, fn, args, namedArgs...)
	}

	if len(namedArgs) > 0 {
		return fn.Call(ctx, args, namedArgs...)
	}

	return fn.Data(ctx, args)
}
