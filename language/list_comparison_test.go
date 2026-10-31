package language

import (
	"context"
	"math"
	"strings"
	"testing"
)

type includesTestValue struct {
	Object
	raw any
}

func (v *includesTestValue) Value() any {
	return v.raw
}

func TestListIncludesPreservesComparisons(t *testing.T) {
	item := NewInt(7, nil)
	dict, err := NewDict(nil, nil, TypeString, TypeAny, nil)
	if err != nil {
		t.Fatal(err)
	}

	definition := NewStruct("Item", nil, nil)
	instance, err := definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	element := NewElement(&ElementData{TagName: "div"}, nil)
	fn := NewFunction(func(context.Context, []Object) (Object, error) { return nil, nil }, nil)
	list := NewList(nil, TypeAny, nil)

	for _, tc := range []struct {
		name   string
		items  []Object
		search Object
		want   bool
	}{
		{"int", []Object{item}, NewInt(7, nil), true},
		{"string", []Object{NewString("text", nil)}, NewString("text", nil), true},
		{"bool", []Object{NewBool(true, nil)}, NewBool(true, nil), true},
		{"float", []Object{NewFloat(1.5, nil)}, NewFloat(1.5, nil), true},
		{"NaN", []Object{NewFloat(math.NaN(), nil)}, NewFloat(math.NaN(), nil), false},
		{"numeric kinds", []Object{item}, NewFloat(7, nil), false},
		{"nil", []Object{Nil}, Nil, true},
		{"missing", []Object{item}, NewInt(8, nil), false},
		{"reference", []Object{item}, NewRef(item), true},
		{"any", []Object{NewAny(item, nil)}, item, true},
		{"dictionary identity", []Object{dict}, NewRef(dict), true},
		{"struct identity", []Object{instance}, NewRef(instance), true},
		{"HTML identity", []Object{element}, NewRef(element), true},
		{"mixed", []Object{list, fn, Nil, item}, NewInt(7, nil), true},
		{"different kind", []Object{list}, fn, false},
		{"empty", nil, list, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := NewList(tc.items, TypeAny, nil)
			includes := listTestMethod(t, values.GetPrototype(), "includes")
			result, err := includes.Call(context.Background(), []Object{tc.search})
			if err != nil || result == nil || result.Value() != tc.want {
				t.Fatalf("includes returned %v, %v; want %t", result, err, tc.want)
			}
		})
	}
}

func TestListIncludesRejectsUncomparableValues(t *testing.T) {
	list := NewList(nil, TypeAny, nil)
	list.Data = []Object{list}
	fn := NewFunction(func(context.Context, []Object) (Object, error) { return nil, nil }, nil)
	comparable := &includesTestValue{Object: NewInt(0, nil), raw: struct{ Value any }{1}}
	uncomparable := &includesTestValue{Object: NewInt(0, nil), raw: struct{ Value any }{[]int{1}}}

	for _, tc := range []struct {
		name   string
		item   Object
		search Object
	}{
		{"cyclic list", list, list},
		{"list reference", NewRef(list), NewRef(list)},
		{"list any", NewAny(list, nil), list},
		{"function", fn, fn},
		{"definition", NewStruct("A", nil, nil), NewStruct("B", nil, nil)},
		{"nested interface", uncomparable, comparable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := NewList([]Object{tc.item}, TypeAny, nil)
			includes := listTestMethod(t, values.GetPrototype(), "includes")

			result, err := includes.Call(context.Background(), []Object{tc.search})
			if result != nil || err == nil || !strings.Contains(err.Error(), "includes cannot compare") {
				t.Fatalf("expected a comparison error, got %v, %v", result, err)
			}

			push := listTestMethod(t, values.GetPrototype(), "push")
			if _, err := callListTestFunction(t, context.Background(), push, NewInt(7, nil)); err != nil {
				t.Fatal(err)
			}

			result, err = includes.Call(context.Background(), []Object{NewInt(7, nil)})
			if err != nil || result == nil || result.Value() != true {
				t.Fatal("failed comparison left the list unusable")
			}
		})
	}
}
