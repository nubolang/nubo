package language

import (
	"context"
	"testing"
	"time"
)

func TestElementAttributeCloneCanAccessElement(t *testing.T) {
	for _, method := range []string{"setAttribute", "__set__"} {
		t.Run(method, func(t *testing.T) {
			ctx := context.Background()
			old := NewString("old", nil)
			keep := NewString("keep", nil)
			element := NewElement(&ElementData{
				TagName: "div",
				Args: []Attribute{
					{Name: "data-value", Kind: "TEXT", Value: old},
					{Name: "keep", Kind: "TEXT", Value: keep},
				},
			}, nil)
			proto := element.GetPrototype()
			getMethod := func(name string) *Function {
				t.Helper()
				obj, ok := proto.GetObject(ctx, name)
				if !ok {
					t.Fatalf("missing method %q", name)
				}
				fn, ok := obj.(*Function)
				if !ok {
					t.Fatalf("method %q is not a function", name)
				}
				return fn
			}
			set := getMethod(method)
			get := getMethod("getAttribute")
			nestedSet := getMethod("setAttribute")
			definition := NewStruct("AttributeValue", nil, nil)
			var cloned *StructInstance
			var observed Object
			var hookErr error
			calls := 0
			err := definition.GetPrototype().SetObject(ctx, "__clone__", NewTypedFunction([]FnArg{
				&BasicFnArg{NameVal: "self", TypeVal: definition.Type()},
			}, definition.Type(), func(ctx context.Context, _ []Object) (Object, error) {
				calls++
				observed, hookErr = get.Call(ctx, []Object{NewString("dataValue", nil)})
				if hookErr == nil {
					_, hookErr = nestedSet.Call(ctx, []Object{NewString("nested", nil), NewString("hook", nil)})
				}
				if hookErr == nil {
					// The outer setter must replace this entry, not append a duplicate.
					_, hookErr = nestedSet.Call(ctx, []Object{NewString("dataValue", nil), NewString("inner", nil)})
				}
				// Record errors for the test; Clone currently treats hook errors as fatal.
				return cloned, nil
			}, nil))
			if err != nil {
				t.Fatal(err)
			}
			source, err := definition.NewInstance()
			if err != nil {
				t.Fatal(err)
			}
			cloned, err = definition.NewInstance()
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan error, 1)
			go func() {
				_, err := set.Call(ctx, []Object{NewString("dataValue", nil), source})
				done <- err
			}()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("attribute clone did not finish; possible reentrant deadlock")
			}

			if hookErr != nil || calls != 1 {
				t.Fatalf("clone hook: error %v, calls %d", hookErr, calls)
			}
			if observed != old {
				t.Fatal("clone hook must see the attribute before the outer write")
			}
			attrs := element.Data.Args
			if len(attrs) != 3 {
				t.Fatalf("got %d attributes; want three without duplicates", len(attrs))
			}
			if attrs[0].Name != "data-value" || attrs[0].Kind != "DYNAMIC" || attrs[0].Value != cloned {
				t.Fatal("outer setter must store the clone result in the original attribute position")
			}
			if attrs[1].Name != "keep" || attrs[1].Value != keep {
				t.Fatal("unrelated attribute was changed")
			}
			if attrs[2].Name != "nested" || attrs[2].Kind != "TEXT" || attrs[2].Value.String() != "hook" {
				t.Fatal("nested string attribute write was not preserved")
			}
		})
	}
}
