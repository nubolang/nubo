// Package execution owns native resources for an interpreter execution.
package execution

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type key struct{}

type Resources struct {
	wake      atomic.Bool
	closeOnce sync.Once
	closeErr  error
	closed    bool
	mu        sync.Mutex
	closers   map[uint64]func() error
	next      uint64
	stopped   error
	pending   []func() error
}

func New(ctx context.Context) (context.Context, *Resources) {
	r := &Resources{closers: make(map[uint64]func() error)}
	return context.WithValue(ctx, key{}, r), r
}

func From(ctx context.Context) *Resources {
	r, _ := ctx.Value(key{}).(*Resources)
	return r
}

func (r *Resources) Add(close func() error) (func(), error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, errors.New("execution has ended")
	}
	r.next++
	id := r.next
	r.closers[id] = close
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		delete(r.closers, id)
		r.mu.Unlock()
	}, nil
}

func (r *Resources) Close() error {
	r.closeOnce.Do(func() {
		r.mu.Lock()
		r.closed = true
		closers := r.closers
		r.closers = nil
		r.pending = nil
		r.mu.Unlock()
		for _, close := range closers {
			r.closeErr = errors.Join(r.closeErr, close())
		}
	})
	return r.closeErr
}

func (r *Resources) Stop(err error) {
	r.mu.Lock()
	r.stopped = err
	r.wake.Store(true)
	r.mu.Unlock()
	_ = r.Close()
}

func (r *Resources) Enqueue(fn func() error) {
	r.mu.Lock()
	if !r.closed && len(r.pending) < 64 {
		r.pending = append(r.pending, fn)
		r.wake.Store(true)
	}
	r.mu.Unlock()
}
func (r *Resources) Check() error {
	if !r.wake.Load() {
		return nil
	}
	r.mu.Lock()
	err := r.stopped
	pending := r.pending
	r.pending = nil
	r.wake.Store(err != nil)
	r.mu.Unlock()

	if err != nil {
		return err
	}

	for _, fn := range pending {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

type cleanupKey struct{}

func Cleanup(ctx context.Context) context.Context { return context.WithValue(ctx, cleanupKey{}, true) }
func Check(ctx context.Context) error {
	if ctx.Value(cleanupKey{}) == true {
		return nil
	}
	if r := From(ctx); r != nil {
		return r.Check()
	}
	return nil
}
