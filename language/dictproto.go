package language

import (
	"fmt"
	"sync"
)

type DictPrototype struct {
	base *Dict
	data map[string]Object
	mu   sync.RWMutex
}

func NewDictPrototype(base *Dict) *DictPrototype {
	dp := &DictPrototype{
		base: base,
		data: make(map[string]Object),
	}

	dp.SetObject("get", NewTypedFunction([]FnArg{&BasicFnArg{TypeVal: base.KeyType, NameVal: "key"}}, base.ValueType,
		func(o []Object) (Object, error) {
			dp.mu.RLock()
			defer dp.mu.RUnlock()

			getterKey := o[0]
			for key := range base.Data {
				if key.Value() == getterKey.Value() {
					return base.Data[key], nil
				}
			}

			return nil, fmt.Errorf("Key %s not found", getterKey)
		}, base.Debug()))

	dp.SetObject("keys", NewTypedFunction(nil, NewListType(base.KeyType),
		func(o []Object) (Object, error) {
			dp.mu.RLock()
			defer dp.mu.RUnlock()

			var keys = make([]Object, 0, len(base.Data))
			for key := range base.Data {
				keys = append(keys, key)
			}

			return NewList(keys, base.KeyType, base.Debug()), nil
		}, base.Debug()))

	dp.SetObject("values", NewTypedFunction(nil, NewListType(base.ValueType),
		func(o []Object) (Object, error) {
			dp.mu.RLock()
			defer dp.mu.RUnlock()

			var values = make([]Object, 0, len(base.Data))
			for _, value := range base.Data {
				values = append(values, value)
			}

			return NewList(values, base.KeyType, base.Debug()), nil
		}, base.Debug()))

	return dp
}

func (dp *DictPrototype) GetObject(name string) (Object, bool) {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	obj, ok := dp.data[name]
	return obj, ok
}

func (dp *DictPrototype) SetObject(name string, value Object) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.data[name] = value
	return nil
}

func (dp *DictPrototype) Objects() map[string]Object {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.data
}
