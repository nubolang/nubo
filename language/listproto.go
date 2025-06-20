package language

import (
	"errors"
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
	lp.SetObject("get", NewTypedFunction([]FnArg{&BasicFnArg{TypeVal: TypeInt, NameVal: "index"}}, lp.base.ItemType,
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
	lp.SetObject("set", NewTypedFunction([]FnArg{
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

		return NewList(newItems, TypeAny, fn.Debug()), nil
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
