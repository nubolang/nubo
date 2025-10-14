package language

import (
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

	// get(index int) -> ItemType
	lp.SetObject("__get__", NewTypedFunction([]FnArg{&BasicFnArg{TypeVal: TypeInt, NameVal: "index"}}, lp.base.ItemType,
		func(o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			index := int(o[0].Value().(int64))

			if index < 0 || index >= len(lp.base.Data) {
				return nil, ErrIndexOutOfBounds
			}

			return lp.base.Data[index], nil
		}, base.Debug()))

	// set(index int, value any)
	lp.SetObject("__set__", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: TypeInt, NameVal: "index"},
		&BasicFnArg{TypeVal: lp.base.ItemType, NameVal: "value"},
	}, TypeVoid, func(o []Object) (Object, error) {
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
	lp.SetObject("join", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: TypeString, NameVal: "sep"},
	}, TypeString, func(o []Object) (Object, error) {
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
	lp.SetObject("length", NewTypedFunction(nil, TypeInt, func(o []Object) (Object, error) {
		lp.mu.RLock()
		defer lp.mu.RUnlock()

		return NewInt(int64(len(lp.base.Data)), base.Debug()), nil
	}, base.Debug()))

	// map(fn: function(item any) any) -> List
	lp.SetObject("map", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: NewFunctionType(TypeAny, base.ItemType), NameVal: "fn"},
	}, TypeList, func(o []Object) (Object, error) {
		fn := o[0].(*Function)

		lp.mu.RLock()
		defer lp.mu.RUnlock()

		newItems := make([]Object, 0, len(lp.base.Data))
		for _, item := range lp.base.Data {
			result, err := fn.Data([]Object{item})
			if err != nil {
				return nil, err
			}
			newItems = append(newItems, result)
		}

		return NewList(newItems, fn.ReturnType, fn.Debug()), nil
	}, base.Debug()))

	lp.SetObject("includes", NewTypedFunction([]FnArg{&BasicFnArg{TypeVal: TypeAny, NameVal: "search"}}, TypeBool,
		func(o []Object) (Object, error) {
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

	lp.SetObject("push", NewTypedFunction([]FnArg{&BasicFnArg{TypeVal: lp.base.ItemType, NameVal: "value"}}, TypeVoid,
		func(o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			base.Data = append(base.Data, o[0])
			return nil, nil
		}, base.Debug()))

	lp.SetObject("insert", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeInt, NameVal: "index"},
			&BasicFnArg{TypeVal: lp.base.ItemType, NameVal: "value"},
		}, TypeVoid,
		func(o []Object) (Object, error) {
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

	lp.SetObject("del", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeInt, NameVal: "index"},
		}, TypeVoid,
		func(o []Object) (Object, error) {
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

	lp.SetObject("pop", NewTypedFunction(nil, lp.base.ItemType,
		func(o []Object) (Object, error) {
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

	lp.SetObject("shift", NewTypedFunction(nil, lp.base.ItemType,
		func(o []Object) (Object, error) {
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

	lp.SetObject("unshift", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: lp.base.ItemType, NameVal: "value"}},
		TypeVoid,
		func(o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			base.Data = append([]Object{o[0]}, base.Data...)
			return nil, nil
		}, base.Debug(),
	))

	lp.SetObject("slice", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeInt, NameVal: "start"},
			&BasicFnArg{TypeVal: TypeInt, NameVal: "end"},
		}, lp.base.Type(),
		func(o []Object) (Object, error) {
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

	lp.SetObject("clear", NewTypedFunction(nil, TypeVoid,
		func(o []Object) (Object, error) {
			lp.mu.RLock()
			defer lp.mu.RUnlock()

			base.Data = []Object{}
			return nil, nil
		}, base.Debug(),
	))

	lp.SetObject("reverse", NewTypedFunction(nil, TypeVoid,
		func(o []Object) (Object, error) {
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

func (lp *ListPrototype) GetObject(name string) (Object, bool) {
	lp.mu.RLock()
	defer lp.mu.RUnlock()
	obj, ok := lp.data[name]
	return obj, ok
}

func (lp *ListPrototype) SetObject(name string, value Object) error {
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
