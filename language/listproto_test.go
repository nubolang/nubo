package language

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
)

// Bound calls so a regression reports a failure instead of hanging the suite.
func callListTestFunction(t *testing.T, ctx context.Context, fn *Function, args ...Object) (Object, error) {
	t.Helper()
	type result struct {
		value Object
		err   error
	}
	done := make(chan result, 1)
	go func() {
		value, err := fn.Call(ctx, args)
		done <- result{value: value, err: err}
	}()
	select {
	case r := <-done:
		return r.value, r.err
	case <-time.After(5 * time.Second):
		t.Fatal("list callback did not finish; possible reentrant deadlock")
		return nil, nil
	}
}

func listTestMethod(t *testing.T, proto Prototype, name string) *Function {
	t.Helper()
	obj, ok := proto.GetObject(context.Background(), name)
	if !ok {
		t.Fatalf("missing method %q", name)
	}
	fn, ok := obj.(*Function)
	if !ok {
		t.Fatalf("method %q is not a function", name)
	}
	return fn
}

func TestListCallbacksCanMutateSource(t *testing.T) {
	methods := []struct {
		name    string
		returns *Type
	}{
		{"each", TypeVoid},
		{"map", TypeInt},
		{"filter", TypeBool},
	}
	for _, method := range methods {
		for _, mutation := range []string{"__set__", "push", "clear", "reverse"} {
			t.Run(method.name+"/"+mutation, func(t *testing.T) {
				original := []Object{NewInt(1, nil), NewInt(2, nil)}
				list := NewList(slices.Clone(original), TypeInt, nil)
				proto := list.GetPrototype()
				mutate := listTestMethod(t, proto, mutation)
				fn := listTestMethod(t, proto, method.name)
				var seen []Object
				type contextKey struct{}
				ctx := context.WithValue(context.Background(), contextKey{}, "caller")
				callback := NewTypedFunction([]FnArg{
					&BasicFnArg{NameVal: "item", TypeVal: TypeInt},
				}, method.returns, func(ctx context.Context, args []Object) (Object, error) {
					if ctx.Value(contextKey{}) != "caller" {
						return nil, fmt.Errorf("callback lost caller context")
					}
					seen = append(seen, args[0])
					if len(seen) == 1 {
						// Prototype writes also need the list's exclusive lock.
						if err := proto.SetObject(ctx, "visited", NewBool(true, nil)); err != nil {
							return nil, err
						}
						var mutationArgs []Object
						switch mutation {
						case "__set__":
							mutationArgs = []Object{NewInt(1, nil), NewInt(99, nil)}
						case "push":
							mutationArgs = []Object{NewInt(99, nil)}
						}
						if _, err := mutate.Call(ctx, mutationArgs); err != nil {
							return nil, err
						}
					}
					switch method.name {
					case "map":
						return args[0], nil
					case "filter":
						return NewBool(true, nil), nil
					default:
						return nil, nil
					}
				}, nil)

				result, err := callListTestFunction(t, ctx, fn, callback)
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(seen, original) {
					t.Fatal("callback traversal must preserve original entry order and identity")
				}
				if method.name == "each" {
					if result != nil {
						t.Fatal("each must return void")
					}
				} else if output, ok := result.(*List); !ok || !slices.Equal(output.Data, original) {
					t.Fatal("map/filter must return the callback's original objects")
				}
			})
		}
	}
}

func TestListCallbacksPropagateErrors(t *testing.T) {
	for _, method := range []string{"each", "map", "filter"} {
		t.Run(method, func(t *testing.T) {
			list := NewList([]Object{NewInt(1, nil), NewInt(2, nil)}, TypeInt, nil)
			proto := list.GetPrototype()
			fn := listTestMethod(t, proto, method)
			returns := TypeVoid
			if method == "map" {
				returns = TypeInt
			} else if method == "filter" {
				returns = TypeBool
			}
			wantErr := errors.New("callback failed")
			calls := 0
			callback := NewTypedFunction([]FnArg{
				&BasicFnArg{NameVal: "item", TypeVal: TypeInt},
			}, returns, func(ctx context.Context, _ []Object) (Object, error) {
				calls++
				if err := proto.SetObject(ctx, "visited", NewBool(true, nil)); err != nil {
					return nil, err
				}
				return nil, wantErr
			}, nil)

			_, err := callListTestFunction(t, context.Background(), fn, callback)
			if !errors.Is(err, wantErr) || calls != 1 {
				t.Fatalf("got error %v after %d calls; want callback error after one call", err, calls)
			}
		})
	}
}

func TestListJoinCanMutateSourceFromStringHook(t *testing.T) {
	ctx := context.Background()
	list := NewList(nil, TypeAny, nil)
	proto := list.GetPrototype()
	set := listTestMethod(t, proto, "__set__")
	join := listTestMethod(t, proto, "join")
	definition := NewStruct("JoinItem", nil, nil)
	var hookErr error
	calls := 0
	err := definition.GetPrototype().SetObject(ctx, "__string__", NewTypedFunction([]FnArg{
		&BasicFnArg{NameVal: "self", TypeVal: definition.Type()},
	}, TypeString, func(ctx context.Context, _ []Object) (Object, error) {
		calls++
		_, hookErr = set.Call(ctx, []Object{NewInt(1, nil), NewString("changed", nil)})
		return NewString("first", nil), hookErr
	}, nil))
	if err != nil {
		t.Fatal(err)
	}
	item, err := definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	list.Data = []Object{item, NewString("second", nil)}

	result, err := callListTestFunction(t, ctx, join, NewString(",", nil))
	if err != nil {
		t.Fatal(err)
	}
	if hookErr != nil || calls != 1 {
		t.Fatalf("string hook: error %v, calls %d", hookErr, calls)
	}
	if result.String() != "first,second" {
		t.Fatalf("join returned %q; want original entries", result.String())
	}
	if list.Data[1].String() != "changed" {
		t.Fatal("string hook's mutation was not applied to the source list")
	}
}
