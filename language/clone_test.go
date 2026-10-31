package language

import (
	"context"
	"errors"
	"testing"
)

func TestCloneFailuresReturnErrors(t *testing.T) {
	for _, kind := range []string{"error", "panic", "not callable", "void"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			definition := NewStruct("BrokenClone", nil, nil)
			failure := errors.New("clone failed")
			calls := 0
			var hook Object = NewFunction(func(context.Context, []Object) (Object, error) {
				calls++
				switch kind {
				case "panic":
					panic("clone failed")
				case "void":
					return nil, nil
				default:
					return nil, failure
				}
			}, nil)

			if kind == "not callable" {
				hook = NewInt(1, nil)
			}

			if err := definition.GetPrototype().SetObject(ctx, "__clone__", hook); err != nil {
				t.Fatal(err)
			}

			instance, err := definition.NewInstance()
			if err != nil {
				t.Fatal(err)
			}

			for range 2 {
				result, err := Clone(ctx, instance)
				if err == nil || result != nil {
					t.Fatal("failed clone did not return an error")
				}

				if kind == "error" && !errors.Is(err, failure) {
					t.Fatal("clone lost the original error")
				}
			}

			if kind != "not callable" && calls != 2 {
				t.Fatal("failed clone left the recursion guard active")
			}

			if got := instance.Clone(); got != instance {
				t.Fatal("legacy clone did not preserve its fallback")
			}
		})
	}
}

func TestDictionaryClonePropagatesHookError(t *testing.T) {
	ctx := context.Background()
	definition := NewStruct("NestedClone", nil, nil)
	failure := errors.New("nested clone failed")
	err := definition.GetPrototype().SetObject(ctx, "__clone__", NewFunction(func(context.Context, []Object) (Object, error) {
		return nil, failure
	}, nil))
	if err != nil {
		t.Fatal(err)
	}

	instance, err := definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	dict, err := NewDict([]Object{NewString("item", nil)}, []Object{instance}, TypeString, TypeAny, nil)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Clone(ctx, dict)
	if result != nil {
		t.Fatalf("failed clone returned a value of type %T", result)
	}

	if !errors.Is(err, failure) {
		t.Fatalf("nested clone error was lost: %v", err)
	}
}
