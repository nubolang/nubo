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

	dp.SetObject("get", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: base.KeyType, NameVal: "key"},
	}, base.ValueType, func(o []Object) (Object, error) {
		dp.mu.RLock()
		defer dp.mu.RUnlock()

		getterKey := o[0]
		var found Object
		base.Data.Iterate(func(key Object, value Object) bool {
			if key.Value() == getterKey.Value() {
				found = value
				return false // stop iteration
			}
			return true
		})

		if found == nil {
			return nil, fmt.Errorf("Key %s not found", getterKey)
		}

		return found, nil
	}, base.Debug()))

	dp.SetObject("set", NewTypedFunction([]FnArg{
		&BasicFnArg{TypeVal: dp.base.KeyType, NameVal: "key"},
		&BasicFnArg{TypeVal: dp.base.ValueType, NameVal: "value"},
	}, TypeVoid, func(o []Object) (Object, error) {
		dp.mu.Lock()
		defer dp.mu.Unlock()

		finalKey := o[0]
		base.Data.Iterate(func(key Object, _ Object) bool {
			if key.Value() == o[0].Value() {
				finalKey = key
				return false
			}
			return true
		})

		base.Data.Set(finalKey, o[1])
		return nil, nil
	}, base.Debug()))

	dp.SetObject("keys", NewTypedFunction(nil, NewListType(base.KeyType),
		func(o []Object) (Object, error) {
			dp.mu.RLock()
			defer dp.mu.RUnlock()

			var keys = make([]Object, 0, base.Data.Len())
			base.Data.Iterate(func(key Object, _ Object) bool {
				keys = append(keys, key)
				return true
			})

			return NewList(keys, base.KeyType, base.Debug()), nil
		}, base.Debug()))

	dp.SetObject("values", NewTypedFunction(nil, NewListType(base.ValueType),
		func(o []Object) (Object, error) {
			dp.mu.RLock()
			defer dp.mu.RUnlock()

			var values = make([]Object, 0, base.Data.Len())
			base.Data.Iterate(func(_ Object, value Object) bool {
				values = append(values, value)
				return true
			})

			return NewList(values, base.ValueType, base.Debug()), nil
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
