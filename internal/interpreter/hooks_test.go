package interpreter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
)

func TestHookDepthAndUnwind(t *testing.T) {
	var fn *language.Function
	var captured context.Context
	remaining := 0

	fn = language.NewFunction(func(ctx context.Context, _ []language.Object) (language.Object, error) {
		captured = ctx
		remaining--

		if remaining == 0 {
			return language.Nil, nil
		}

		return callHook(ctx, fn, nil)
	}, &debug.Debug{File: "hooks.nubo", Line: 1})

	remaining = maxHookDepth
	if _, err := callHook(context.Background(), fn, nil); err != nil {
		t.Fatalf("terminating recursion failed: %v", err)
	}

	remaining = maxHookDepth + 1
	_, err := callHook(context.Background(), fn, nil)
	if !errors.Is(err, errHookDepth) {
		t.Fatalf("expected depth error, got %v", err)
	}

	wrapped := wrapRunExc(err, &debug.Debug{File: "hooks.nubo", Line: 2})
	data, ok := exception.Unwrap(wrapped)
	if !ok || data.Debug == nil || len(data.StackTrace) == 0 || !errors.Is(wrapped, errHookDepth) {
		t.Fatal("wrapping lost the depth error or its trace")
	}

	remaining = maxHookDepth
	if _, err := callHook(captured, fn, nil); err != nil {
		t.Fatalf("completed hooks stayed active: %v", err)
	}
}

func TestConcurrentHooksAreIndependent(t *testing.T) {
	count := maxHookDepth + 1
	entered := make(chan struct{}, count)
	results := make(chan error, count)
	release := make(chan struct{})
	defer close(release)

	fn := language.NewFunction(func(ctx context.Context, _ []language.Object) (language.Object, error) {
		entered <- struct{}{}
		<-release

		return language.Nil, nil
	}, nil)

	for range count {
		go func() {
			_, err := callHook(context.Background(), fn, nil)
			results <- err
		}()
	}

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for range count {
		select {
		case <-entered:
		case err := <-results:
			t.Fatalf("independent hook was rejected: %v", err)
		case <-timer.C:
			t.Fatal("independent hooks did not all enter")
		}
	}
}

type hookTestRuntime struct {
	Runtime
}

func (hookTestRuntime) GetBuiltin(string) (language.Object, bool) {
	return nil, false
}

func TestPropertyHookErrors(t *testing.T) {
	for _, kind := range []string{"getter", "setter", "mixed"} {
		t.Run(kind, func(t *testing.T) {
			ir := &Interpreter{
				ctx:     context.Background(),
				runtime: hookTestRuntime{},
				objects: make(map[uint32]*entry),
			}

			definition := language.NewUserStruct("Hooked", nil)
			definition.DefineFieldset([]language.StructField{{Name: "stored", Type: language.TypeAny}})
			instance, err := definition.NewInstance()
			if err != nil {
				t.Fatal(err)
			}

			proto := instance.GetPrototype().(*language.StructPrototype)
			proto.Unlock()
			defer proto.Lock()

			value := language.NewInt(7, nil)

			if err := ir.Declare("items", instance, instance.Type(), true); err != nil {
				t.Fatal(err)
			}

			getter := language.NewFunction(func(ctx context.Context, _ []language.Object) (language.Object, error) {
				child := NewWithParent(ir, ScopeFunction)
				child.ctx = ctx

				if kind == "mixed" {
					return nil, child.Assign("items.missing", value)
				}

				obj, _, err := child.lookupObject("items.missing")
				return obj, err
			}, nil)

			setter := language.NewFunction(func(ctx context.Context, _ []language.Object) (language.Object, error) {
				child := NewWithParent(ir, ScopeFunction)
				child.ctx = ctx

				if kind == "mixed" {
					obj, _, err := child.lookupObject("items.missing")
					return obj, err
				}

				return nil, child.Assign("items.missing", value)
			}, nil)

			if err := proto.SetObject(ir.ctx, "__get__", getter); err != nil {
				t.Fatal(err)
			}

			if err := proto.SetObject(ir.ctx, "__set__", setter); err != nil {
				t.Fatal(err)
			}

			err = nil
			if kind == "setter" {
				err = ir.Assign("items.missing", value)
			} else {
				_, _, err = ir.lookupObject("items.missing")
			}

			if !errors.Is(err, errHookDepth) {
				t.Fatalf("depth error was swallowed: %v", err)
			}

			if _, ok := proto.GetObject(ir.ctx, "missing"); ok {
				t.Fatal("failed hook fell back to a direct write")
			}

			failure := language.NewFunction(func(context.Context, []language.Object) (language.Object, error) {
				return nil, errors.New("ordinary hook error")
			}, nil)

			for _, name := range []string{"__get__", "__set__"} {
				if err := proto.SetObject(ir.ctx, name, failure); err != nil {
					t.Fatal(err)
				}
			}

			if _, ok, err := ir.lookupObject("items.missing"); ok || err != nil {
				t.Fatal("ordinary getter fallback changed")
			}

			if err := ir.Assign("items.stored", value); err != nil {
				t.Fatalf("ordinary setter fallback changed: %v", err)
			}

			if got, ok := proto.GetObject(ir.ctx, "stored"); !ok || got != value {
				t.Fatal("ordinary setter fallback did not store the value")
			}
		})
	}
}
