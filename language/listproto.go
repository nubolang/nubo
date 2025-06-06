package language

import (
	"errors"
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
