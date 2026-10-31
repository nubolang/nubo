package language

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/nubolang/nubo/internal/ast/astnode"
)

func TestCyclicValueTraversal(t *testing.T) {
	list := NewList(nil, TypeAny, nil)
	list.Data = []Object{list}

	dict := &Dict{Data: NewOrderedMap(), KeyType: TypeString, ValueType: TypeAny}
	dict.Data.Set(NewString("self", nil), dict)

	definition := NewStruct("Node", []StructField{{Name: "next", Type: TypeAny}}, nil)
	instance, err := definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	if err := instance.GetPrototype().SetObject(context.Background(), "next", instance); err != nil {
		t.Fatal(err)
	}

	for _, obj := range []Object{list, dict, instance} {
		t.Run(obj.TypeString(), func(t *testing.T) {
			if text := obj.String(); !strings.Contains(text, "<cycle>") {
				t.Fatalf("missing cycle marker: %s", text)
			}

			for _, jsonMode := range []bool{false, true} {
				if _, err := ToValue(obj, jsonMode); !errors.Is(err, ErrCyclicValue) {
					t.Fatalf("expected a cycle error, got %v", err)
				}
			}
		})
	}

	ref := NewRef(list)
	if text := ref.String(); !strings.Contains(text, "<cycle>") {
		t.Fatalf("reference hid the cycle: %s", text)
	}

	cloned := dict.Clone().(*Dict)
	if cloned == dict || cloned.Data == dict.Data {
		t.Fatal("clone reused the original dictionary")
	}

	cloned.Data.Iterate(func(_ Object, value Object) bool {
		if value != cloned {
			t.Fatal("clone did not preserve its circular link")
		}

		return true
	})
}

func TestSharedValuesAreNotCycles(t *testing.T) {
	child := NewList([]Object{NewInt(1, nil), NewInt(2, nil)}, TypeInt, nil)
	root := NewList([]Object{child, child}, TypeList, nil)

	if text := root.String(); text != "[[1, 2], [1, 2]]" {
		t.Fatalf("shared child was treated as a cycle: %s", text)
	}

	value, err := ToValue(root)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(value, []any{[]int64{1, 2}, []int64{1, 2}}) {
		t.Fatalf("typed slice conversion changed: %#v", value)
	}

	jsonValue, err := ToValue(root, true)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(jsonValue, []any{[]any{int64(1), int64(2)}, []any{int64(1), int64(2)}}) {
		t.Fatalf("JSON array conversion changed: %#v", jsonValue)
	}

	done := make(chan bool, 8)
	for range 8 {
		go func() {
			_, err := ToValue(root)
			done <- err == nil && root.String() == "[[1, 2], [1, 2]]"
		}()
	}

	for range 8 {
		if !<-done {
			t.Fatal("independent traversals interfered")
		}
	}
}

func TestCyclicHTMLTraversal(t *testing.T) {
	element := NewElement(&ElementData{TagName: "div"}, nil)
	alias := NewElement(element.Data, nil)
	element.Data.Children = []ElementChild{{Type: astnode.NodeTypeElement, Value: alias}}

	if text := element.String(); text != "<div><cycle></div>" {
		t.Fatalf("HTML alias hid the cycle: %s", text)
	}

	element.Data.Children = nil
	element.Data.Args = []Attribute{{Name: "dataValue", Kind: "DYNAMIC", Value: element}}

	if text := element.String(); text != `<div data-value="&lt;cycle&gt;"></div>` {
		t.Fatalf("cyclic attribute was not escaped: %s", text)
	}
}

func TestStringAndConversionHookReentry(t *testing.T) {
	definition := NewStruct("Hooked", nil, nil)
	var instance *StructInstance
	var captured context.Context
	calls := 0

	hook := NewTypedFunction(nil, TypeString, func(ctx context.Context, _ []Object) (Object, error) {
		captured = ctx
		calls++

		return NewString(Stringify(ctx, instance), nil), nil
	}, nil)

	if err := definition.GetPrototype().SetObject(context.Background(), "__string__", hook); err != nil {
		t.Fatal(err)
	}

	var err error
	instance, err = definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	if text := instance.String(); text != "<cycle>" || calls != 1 {
		t.Fatalf("string hook recursion was not stopped: %s, calls %d", text, calls)
	}

	if text := Stringify(captured, instance); text != "<cycle>" || calls != 2 {
		t.Fatal("completed traversal stayed active in a captured context")
	}

	proto := instance.GetPrototype().(*StructPrototype)
	proto.Unlock()

	err = proto.SetObject(context.Background(), "$convout", NewFunction(func(ctx context.Context, _ []Object) (Object, error) {
		_, err := ToValueContext(ctx, instance)
		return nil, err
	}, nil))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ToValue(instance); !errors.Is(err, ErrCyclicValue) {
		t.Fatalf("conversion hook recursion was not stopped: %v", err)
	}
}

func TestValueDepthLimit(t *testing.T) {
	var obj Object = NewInt(1, nil)
	for range maxValueDepth + 1 {
		obj = NewList([]Object{obj}, TypeAny, nil)
	}

	if text := obj.String(); !strings.Contains(text, "<max-depth>") {
		t.Fatal("deep printing did not stop")
	}

	if _, err := ToValue(obj); !errors.Is(err, ErrValueDepth) {
		t.Fatalf("deep conversion did not stop: %v", err)
	}
}

func TestDictionaryTraversalIgnoresAddedEntries(t *testing.T) {
	dict := &Dict{Data: NewOrderedMap(), KeyType: TypeString, ValueType: TypeAny}
	definition := NewStruct("GrowingValue", nil, nil)
	var instance *StructInstance
	calls := 0

	hook := NewTypedFunction(nil, TypeString, func(ctx context.Context, _ []Object) (Object, error) {
		calls++
		dict.Data.Set(NewString("added", nil), instance)

		return NewString("value", nil), nil
	}, nil)

	if err := definition.GetPrototype().SetObject(context.Background(), "__string__", hook); err != nil {
		t.Fatal(err)
	}

	var err error
	instance, err = definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	dict.Data.Set(NewString("first", nil), instance)

	if text := dict.String(); calls != 1 || strings.Contains(text, "added") {
		t.Fatalf("printing followed entries added by a hook: %s", text)
	}

	if dict.Data.Len() != 2 {
		t.Fatal("printing discarded the hook's write")
	}
}
