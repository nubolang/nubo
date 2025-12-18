package language

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var ErrIndexOutOfBounds = errors.New("index out of bounds")

type ListPrototype struct {
	base *List
	data map[string]Object
	mu   sync.RWMutex
}

func NewListPrototype(base *List) *ListPrototype {
	lp := &ListPrototype{
		base: base,
		data: make(map[string]Object),
	}

	ctx := context.Background()

	// get(index int) -> ItemType
	lp.SetObject(ctx, "__get__", NewTypedFunction([]FnArg{&BasicFnArg{TypeVal: TypeInt, NameVal: "index"}}, lp.base.ItemType,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			index := int(o[0].Value().(int64))

			if index < 0 || index >= len(lp.base.Data) {
				return nil, ErrIndexOutOfBounds
			}

			return lp.base.Data[index], nil
		}, base.Debug()))

	// set(index int, value any)
	lp.SetObject(ctx, "__set__", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: TypeInt, NameVal: "index"},
		&BasicFnArg{TypeVal: lp.base.ItemType, NameVal: "value"},
	}, TypeVoid, func(ctx context.Context, o []Object) (Object, error) {
		lp.mu.Lock()
		defer lp.mu.Unlock()

		index := int(o[0].Value().(int64))
		if index < 0 || index >= len(lp.base.Data) {
			return nil, ErrIndexOutOfBounds
		}

		lp.base.Data[index] = o[1]
		return nil, nil
	}, base.Debug()))

	// join(sep string) -> string
	lp.SetObject(ctx, "join", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: TypeString, NameVal: "sep"},
	}, TypeString, func(ctx context.Context, o []Object) (Object, error) {
		lp.mu.RLock()
		defer lp.mu.RUnlock()

		sep := o[0].Value().(string)
		parts := make([]string, len(lp.base.Data))
		for i, v := range lp.base.Data {
			parts[i] = v.String()
		}

		return NewString(strings.Join(parts, sep), o[0].Debug()), nil
	}, base.Debug()))

	// length() -> int
	lp.SetObject(ctx, "length", NewTypedFunction(nil, TypeInt, func(ctx context.Context, o []Object) (Object, error) {
		lp.mu.RLock()
		defer lp.mu.RUnlock()

		return NewInt(int64(len(lp.base.Data)), base.Debug()), nil
	}, base.Debug()))

	// map(fn: function(item any) any) -> List
	lp.SetObject(ctx, "map", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: NewFunctionType(TypeAny, base.ItemType), NameVal: "fn"},
	}, TypeList, func(ctx context.Context, o []Object) (Object, error) {
		fn := o[0].(*Function)

		lp.mu.RLock()
		defer lp.mu.RUnlock()

		newItems := make([]Object, 0, len(lp.base.Data))
		for _, item := range lp.base.Data {
			result, err := fn.Data(ctx, []Object{item})
			if err != nil {
				return nil, err
			}
			newItems = append(newItems, result)
		}

		return NewList(newItems, fn.ReturnType, fn.Debug()), nil
	}, base.Debug()))

	lp.SetObject(ctx, "filter", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: NewFunctionType(TypeBool, base.ItemType), NameVal: "filterFunc"},
	}, TypeList, func(ctx context.Context, o []Object) (Object, error) {
		fn := o[0].(*Function)

		lp.mu.RLock()
		defer lp.mu.RUnlock()

		newItems := make([]Object, 0, len(lp.base.Data))
		for _, item := range lp.base.Data {
			// Call the predicate function
			result, err := fn.Data(ctx, []Object{item})
			if err != nil {
				return nil, err
			}

			// Keep only if result is true
			keep := false
			if b, ok := result.(*Bool); ok {
				keep = b.Data
			} else {
				return nil, fmt.Errorf("filter function must return a boolean, got %T", result)
			}

			if keep {
				newItems = append(newItems, item)
			}
		}

		return NewList(newItems, base.ItemType, nil), nil
	}, base.Debug()))

	lp.SetObject(ctx, "includes", NewTypedFunction([]FnArg{&BasicFnArg{TypeVal: TypeAny, NameVal: "search"}}, TypeBool,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			search := o[0].Value()
			for _, value := range lp.base.Data {
				if search == value.Value() {
					return NewBool(true, lp.base.Debug()), nil
				}
			}

			return NewBool(false, lp.base.Debug()), nil
		}, base.Debug()))

	lp.SetObject(ctx, "push", NewTypedFunction([]FnArg{&BasicFnArg{TypeVal: lp.base.ItemType, NameVal: "value"}}, TypeVoid,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			base.Data = append(base.Data, o[0])
			return nil, nil
		}, base.Debug()))

	lp.SetObject(ctx, "insert", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeInt, NameVal: "index"},
			&BasicFnArg{TypeVal: lp.base.ItemType, NameVal: "value"},
		}, TypeVoid,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			idx := int(o[0].Value().(int64))
			val := o[1]
			if idx < 0 || idx > len(base.Data) {
				return nil, ErrIndexOutOfBounds
			}
			base.Data = append(base.Data[:idx], append([]Object{val}, base.Data[idx:]...)...)
			return nil, nil
		}, base.Debug(),
	))

	lp.SetObject(ctx, "del", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeInt, NameVal: "index"},
		}, TypeVoid,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			idx := int(o[0].Value().(int64))
			if idx < 0 || idx >= len(base.Data) {
				return nil, ErrIndexOutOfBounds
			}
			base.Data = append(base.Data[:idx], base.Data[idx+1:]...)
			return nil, nil
		}, base.Debug(),
	))

	lp.SetObject(ctx, "pop", NewTypedFunction(nil, lp.base.ItemType,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			if len(base.Data) == 0 {
				return nil, fmt.Errorf("list is empty")
			}
			val := base.Data[len(base.Data)-1]
			base.Data = base.Data[:len(base.Data)-1]
			return val, nil
		}, base.Debug(),
	))

	lp.SetObject(ctx, "shift", NewTypedFunction(nil, lp.base.ItemType,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			if len(base.Data) == 0 {
				return nil, fmt.Errorf("list is empty")
			}
			val := base.Data[0]
			base.Data = base.Data[1:]
			return val, nil
		}, base.Debug(),
	))

	lp.SetObject(ctx, "unshift", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: lp.base.ItemType, NameVal: "value"}},
		TypeVoid,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			base.Data = append([]Object{o[0]}, base.Data...)
			return nil, nil
		}, base.Debug(),
	))

	lp.SetObject(ctx, "slice", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeInt, NameVal: "start"},
			&BasicFnArg{TypeVal: TypeInt, NameVal: "end"},
		}, lp.base.Type(),
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			start := int(o[0].Value().(int64))
			end := int(o[1].Value().(int64))
			if start < 0 || end > len(base.Data) || start > end {
				return nil, fmt.Errorf("invalid slice range")
			}
			sublist := base.Data[start:end]
			return NewList(sublist, lp.base.ItemType, lp.base.Debug()), nil
		}, base.Debug(),
	))

	lp.SetObject(ctx, "clear", NewTypedFunction(nil, TypeVoid,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			base.Data = []Object{}
			return nil, nil
		}, base.Debug(),
	))

	lp.SetObject(ctx, "reverse", NewTypedFunction(nil, TypeVoid,
		func(ctx context.Context, o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			for i, j := 0, len(base.Data)-1; i < j; i, j = i+1, j-1 {
				base.Data[i], base.Data[j] = base.Data[j], base.Data[i]
			}
			return nil, nil
		}, base.Debug(),
	))

	return lp
}

func (lp *ListPrototype) GetObject(ctx context.Context, name string) (Object, bool) {
	lp.mu.RLock()
	defer lp.mu.RUnlock()
	obj, ok := lp.data[name]
	return obj, ok
}

func (lp *ListPrototype) SetObject(ctx context.Context, name string, value Object) error {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	lp.data[name] = value
	return nil
}

func (lp *ListPrototype) Objects() map[string]Object {
	lp.mu.RLock()
	defer lp.mu.RUnlock()
	return lp.data
}
