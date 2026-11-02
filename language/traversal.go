package language

import (
	"context"
	"errors"
	"sync/atomic"
)

const maxValueDepth = 256

var (
	ErrCyclicValue = errors.New("cyclic value")
	ErrValueDepth  = errors.New("value nesting limit exceeded")
)

type traversalKey int

const (
	stringTraversal traversalKey = iota
	valueTraversal
)

type traversalFrame struct {
	object any
	parent *traversalFrame
	active atomic.Bool
}

func enterTraversal(ctx context.Context, obj Object, key traversalKey) (context.Context, func(), error) {
	var identity any

	switch v := obj.(type) {
	case *List, *StructInstance, *Ref:
		identity = obj
	case *Dict:
		identity = v.Data
	case *Element:
		identity = v.Data
	default:
		return ctx, func() {}, nil
	}

	head, _ := ctx.Value(key).(*traversalFrame)
	depth := 0

	for frame := head; frame != nil; frame = frame.parent {
		if !frame.active.Load() {
			continue
		}

		if frame.object == identity {
			return ctx, nil, ErrCyclicValue
		}

		depth++
	}

	if depth >= maxValueDepth {
		return ctx, nil, ErrValueDepth
	}

	// Captured contexts may outlive the traversal.
	frame := &traversalFrame{object: identity, parent: head}
	frame.active.Store(true)

	return context.WithValue(ctx, key, frame), func() { frame.active.Store(false) }, nil
}

func Stringify(ctx context.Context, obj Object) string {
	if obj == nil {
		return "<nil>"
	}

	ctx, leave, err := enterTraversal(ctx, obj, stringTraversal)
	if err == ErrCyclicValue {
		return "<cycle>"
	}

	if err != nil {
		return "<max-depth>"
	}
	defer leave()

	switch v := obj.(type) {
	case *List:
		return v.stringWithContext(ctx)
	case *Dict:
		return v.stringWithContext(ctx)
	case *StructInstance:
		return v.stringWithContext(ctx)
	case *Element:
		return v.stringWithContext(ctx)
	case *Ref:
		return Stringify(ctx, v.Data)
	default:
		return obj.String()
	}
}
