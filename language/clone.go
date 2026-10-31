package language

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/nubolang/nubo/internal/exception"
	"go.uber.org/zap"
)

type cloneStackKey struct{}

type cloneFrame struct {
	typeID string
	parent *cloneFrame
	active atomic.Bool
}

func NewCloneContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, cloneStackKey{}, (*cloneFrame)(nil))
}

func enterStructClone(ctx context.Context, typeID string) (context.Context, func(), bool) {
	head, _ := ctx.Value(cloneStackKey{}).(*cloneFrame)

	for frame := head; frame != nil; frame = frame.parent {
		if frame.active.Load() && frame.typeID == typeID {
			return ctx, nil, false
		}
	}

	// Captured contexts may outlive the clone.
	frame := &cloneFrame{typeID: typeID, parent: head}
	frame.active.Store(true)

	return context.WithValue(ctx, cloneStackKey{}, frame), func() { frame.active.Store(false) }, true
}

func Clone(ctx context.Context, value Object) (result Object, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if cause, ok := recovered.(error); ok {
				err = fmt.Errorf("clone panicked: %w", cause)
			} else {
				err = fmt.Errorf("clone panicked: %v", recovered)
			}
		}

		if err != nil {
			result = nil

			if value != nil {
				err = exception.From(err, value.Debug()).WithLevel(exception.LevelRuntime)
			}
		}
	}()

	return cloneObject(ctx, value, make(map[*Dict]*Dict))
}

func cloneObject(ctx context.Context, value Object, active map[*Dict]*Dict) (Object, error) {
	switch obj := value.(type) {
	case nil:
		return nil, fmt.Errorf("cannot clone void")
	case *StructInstance:
		return obj.clone(ctx)
	case *Dict:
		return obj.clone(ctx, active)
	case *Any:
		cloned, err := cloneObject(ctx, obj.Data, active)
		if err != nil {
			return nil, err
		}

		return NewAny(cloned, obj.debug), nil
	default:
		return value.Clone(), nil
	}
}

// The legacy Object API cannot return errors.
func cloneOrOriginal(value Object) Object {
	cloned, err := Clone(context.Background(), value)
	if err != nil {
		zap.L().Error("clone failed", zap.Error(err))
		return value
	}

	return cloned
}
