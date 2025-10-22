package language

import (
	"context"
	"sync"
)

type IntPrototype struct {
	base *Int
	data map[string]Object
	mu   sync.RWMutex
}

func NewIntPrototype(base *Int) *IntPrototype {
	ip := &IntPrototype{
		base: base,
		data: make(map[string]Object),
	}

	ctx := context.Background()

	ip.SetObject(ctx, "increment", NewFunction(func(ctx context.Context, o []Object) (Object, error) {
		ip.mu.Lock()
		defer ip.mu.Unlock()

		ip.base.Data++
		return nil, nil
	}, base.Debug()))

	ip.SetObject(ctx, "decrement", NewFunction(func(ctx context.Context, o []Object) (Object, error) {
		ip.mu.Lock()
		defer ip.mu.Unlock()

		ip.base.Data--
		return nil, nil
	}, base.Debug()))

	return ip
}

func (i *IntPrototype) GetObject(ctx context.Context, name string) (Object, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	obj, ok := i.data[name]
	return obj, ok
}

func (i *IntPrototype) SetObject(ctx context.Context, name string, value Object) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.data[name] = value
	return nil
}

func (i *IntPrototype) Objects() map[string]Object {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.data
}
