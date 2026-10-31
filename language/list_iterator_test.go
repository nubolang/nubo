package language

import (
	"context"
	"slices"
	"testing"
)

func TestListIteratorAllowsMutation(t *testing.T) {
	for _, mutation := range []string{"clear", "pop", "shift", "del", "push", "unshift", "insert", "reverse", "__set__"} {
		t.Run(mutation, func(t *testing.T) {
			original := []Object{NewInt(1, nil), NewInt(2, nil), NewInt(3, nil)}
			list := NewList(slices.Clone(original), TypeInt, nil)
			mutate := listTestMethod(t, list.GetPrototype(), mutation)
			next := list.Iterator()

			for index, want := range original {
				key, value, ok := next()
				if !ok || key.Value() != int64(index) || value != want {
					t.Fatalf("entry %d lost its original index or object", index)
				}

				if index == 0 {
					var args []Object
					switch mutation {
					case "del":
						args = []Object{NewInt(0, nil)}
					case "push", "unshift":
						args = []Object{NewInt(9, nil)}
					case "insert", "__set__":
						args = []Object{NewInt(1, nil), NewInt(9, nil)}
					}

					if _, err := callListTestFunction(t, context.Background(), mutate, args...); err != nil {
						t.Fatal(err)
					}
				}
			}

			for range 2 {
				if key, value, ok := next(); ok || key != nil || value != nil {
					t.Fatal("completed iterator should stay exhausted")
				}
			}
		})
	}
}

func TestEmptyListIteratorIgnoresLaterEntries(t *testing.T) {
	list := NewList(nil, TypeInt, nil)
	next := list.Iterator()
	push := listTestMethod(t, list.GetPrototype(), "push")

	if _, err := callListTestFunction(t, context.Background(), push, NewInt(1, nil)); err != nil {
		t.Fatal(err)
	}

	if _, _, ok := next(); ok {
		t.Fatal("empty iterator visited a later addition")
	}

	if _, _, ok := list.Iterator()(); !ok {
		t.Fatal("new iterator did not see the addition")
	}
}

func TestListIteratorKeepsObjectsShared(t *testing.T) {
	item := NewInt(1, nil)
	list := NewList([]Object{item}, TypeInt, nil)
	next := list.Iterator()
	increment := listTestMethod(t, item.GetPrototype(), "increment")

	if _, err := increment.Call(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	_, value, ok := next()
	if !ok || value != item || value.Value() != int64(2) {
		t.Fatal("iterator copied the contained object")
	}
}
