package language

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

func TestTypeBindingDoesNotChangeTemplates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		template *Type
		first    Object
		second   Object
	}{
		{"int", TypeInt, NewInt(1, nil), NewInt(2, nil)},
		{"float", TypeFloat, NewFloat(1, nil), NewFloat(2, nil)},
		{"bool", TypeBool, NewBool(true, nil), NewBool(false, nil)},
		{"string", TypeString, NewString("a", nil), NewString("b", nil)},
		{"char", TypeChar, NewChar('a', nil), NewChar('b', nil)},
		{"byte", TypeByte, NewByte(1, nil), NewByte(2, nil)},
		{"html", TypeHtml, NewElement(nil, nil), NewElement(nil, nil)},
		{"type", TypeTypeObj, NewTypeObject(TypeInt, nil), NewTypeObject(TypeString, nil)},
		{"nil", TypeNil, &NilObj{}, &NilObj{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := tc.first.Type()
			second := tc.second.Type()

			if first.Object != tc.first || second.Object != tc.second {
				t.Fatal("another Type call replaced the bound object")
			}

			if tc.template.Object != nil {
				t.Fatal("Type changed the shared template")
			}

			if first.Base() != tc.template.Base() || !first.Compare(second) {
				t.Fatal("type compatibility changed")
			}

			first.Content = "local"
			if tc.first.Type().Content != tc.template.Content {
				t.Fatal("changing a returned descriptor changed the template")
			}
		})
	}
}

func TestFunctionAndWrapperTypeBindings(t *testing.T) {
	fn := NewTypedFunction(nil, TypeInt, func(context.Context, []Object) (Object, error) {
		return NewInt(1, nil), nil
	}, nil)
	cloned := fn.Clone()
	originalType := fn.Type()
	cloneType := cloned.Type()

	if originalType.Object != fn || cloneType.Object != cloned || fn.typ.Object != nil {
		t.Fatal("function clones shared an object binding")
	}

	value := NewInt(1, nil)
	ref := NewRef(value)
	wrapped := NewAny(value, nil)
	valueType := value.Type()
	refType := ref.Type()
	anyType := wrapped.Type()

	if valueType.Object != value || refType.Object != ref || anyType.Object != wrapped {
		t.Fatal("a wrapper replaced the underlying object's type binding")
	}

	if ref.Data != value || ref.GetPrototype() != value.GetPrototype() {
		t.Fatal("reference identity changed")
	}
}

func TestConcurrentTypeBindings(t *testing.T) {
	ctx := context.Background()
	withMethod := NewInt(1, nil)
	withoutMethod := NewInt(2, nil)

	method := NewTypedFunction(nil, TypeString, func(context.Context, []Object) (Object, error) {
		return NewString("ok", nil), nil
	}, nil)

	if err := withMethod.GetPrototype().SetObject(ctx, "probe", method); err != nil {
		t.Fatal(err)
	}

	// Keep lazy prototype initialization outside this race regression.
	withoutMethod.GetPrototype()

	contract := &Type{
		BaseType: ObjectTypeIface,
		Iface: &IfaceType{Functions: []IfaceTypeFn{
			{Name: "probe", Returns: TypeString},
		}},
	}

	retained := withMethod.Type()
	withoutMethod.Type()

	if !retained.SatisfiesIface(contract) {
		t.Fatal("a retained type inspected another object's prototype")
	}

	results := make(chan error, 8)
	for worker := range 8 {
		go func() {
			var obj Object = withMethod
			want := true
			if worker%2 != 0 {
				obj = withoutMethod
				want = false
			}

			for range 1000 {
				typ := obj.Type()
				NewRef(obj).Type()

				if typ.Object != obj || typ.SatisfiesIface(contract) != want {
					results <- fmt.Errorf("worker %d used another object's type binding", worker)
					return
				}
			}

			results <- nil
		}()
	}

	for range 8 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}

	if TypeInt.Object != nil || method.typ.Object != nil {
		t.Fatal("concurrent checks changed shared metadata")
	}
}

func TestConversionWithBoundTypes(t *testing.T) {
	for _, tc := range []struct {
		value Object
		want  any
	}{
		{NewInt(7, nil), []int64{7}},
		{NewFloat(1.5, nil), []float64{1.5}},
		{NewBool(true, nil), []bool{true}},
		{NewString("text", nil), []string{"text"}},
	} {
		list := NewList([]Object{tc.value}, tc.value.Type(), nil)
		got, err := ToValue(list)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("bound item type changed conversion: %#v, %v", got, err)
		}
	}

	key := NewString("key", nil)
	dict, err := NewDict([]Object{key}, []Object{NewInt(7, nil)}, key.Type(), TypeInt, nil)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ToValue(dict)
	if err != nil || !reflect.DeepEqual(got, map[string]any{"key": int64(7)}) {
		t.Fatalf("bound key type changed conversion: %#v, %v", got, err)
	}

	union := NewUnionType(TypeInt, TypeString)
	list := NewList([]Object{NewInt(7, nil), NewString("text", nil)}, union, nil)
	got, err = ToValue(list)
	if err != nil || !reflect.DeepEqual(got, []any{int64(7), "text"}) {
		t.Fatalf("union list lost its mixed values: %#v, %v", got, err)
	}
}
