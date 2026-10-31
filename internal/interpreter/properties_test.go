package interpreter

import (
	"context"
	"strings"
	"testing"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
)

func TestBuiltinPrototypeWritesAreRejected(t *testing.T) {
	ctx := context.Background()
	fn := language.NewFunction(func(context.Context, []language.Object) (language.Object, error) {
		return nil, nil
	}, nil)
	list := language.NewList(nil, language.TypeAny, nil)
	dict, err := language.NewDict(nil, nil, language.TypeInt, language.TypeAny, nil)
	if err != nil {
		t.Fatal(err)
	}

	native := language.NewStruct("Native", nil, nil)
	instance, err := native.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	instance.GetPrototype().(*language.StructPrototype).Unlock()

	for _, tc := range []struct {
		name string
		obj  language.Object
		key  string
	}{
		{"int addition", language.NewInt(1, nil), "probe"},
		{"int replacement", language.NewInt(1, nil), "increment"},
		{"reference", language.NewRef(language.NewInt(1, nil)), "probe"},
		{"string", language.NewString("text", nil), "length"},
		{"untyped function", fn, "probe"},
		{"list", list, "push"},
		{"dictionary", dict, "__set__"},
		{"HTML", language.NewElement(&language.ElementData{TagName: "div"}, nil), "setAttribute"},
		{"native struct", native, "probe"},
		{"native instance", instance, "probe"},
		{"native reference", language.NewRef(instance), "probe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, mutable := range []bool{false, true} {
				ir := &Interpreter{ctx: ctx, objects: make(map[uint32]*entry)}
				if err := ir.Declare("value", tc.obj, tc.obj.Type(), mutable); err != nil {
					t.Fatal(err)
				}

				err := ir.Assign("value."+tc.key, fn)
				data, ok := exception.Unwrap(err)
				if !ok || !strings.Contains(data.Message, "cannot modify built-in prototype") {
					t.Fatalf("expected a frozen-prototype error, got %v", err)
				}
			}
		})
	}

	ir := &Interpreter{ctx: ctx, objects: make(map[uint32]*entry)}
	if err := ir.Declare("mapping", dict, dict.Type(), true); err != nil {
		t.Fatal(err)
	}

	if err := ir.Assign("mapping.probe", fn); err == nil {
		t.Fatal("failed dictionary setter allowed a prototype write")
	}

	if _, ok := dict.GetPrototype().GetObject(ctx, "probe"); ok {
		t.Fatal("failed setter changed the dictionary prototype")
	}
}

func TestCustomStructPropertiesRemainWritable(t *testing.T) {
	ctx := context.Background()
	definition := language.NewUserStruct("Custom", nil)
	definition.DefineFieldset([]language.StructField{{Name: "value", Type: language.TypeInt}})
	instance, err := definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	ir := &Interpreter{ctx: ctx, objects: make(map[uint32]*entry)}
	ref := language.NewRef(instance)
	if err := ir.Declare("custom", ref, ref.Type(), false); err != nil {
		t.Fatal(err)
	}

	value := language.NewInt(7, nil)
	if err := ir.Assign("custom.value", value); err != nil {
		t.Fatal(err)
	}

	if got, _ := instance.GetPrototype().GetObject(ctx, "value"); got != value {
		t.Fatal("reference did not update the declared field")
	}

	if err := ir.Assign("custom.value.probe", value); err == nil {
		t.Fatal("nested access bypassed the prototype restriction")
	}

	if err := ir.Declare("Custom", definition, definition.Type(), false); err != nil {
		t.Fatal(err)
	}

	method := language.NewTypedFunction(nil, language.TypeVoid, func(context.Context, []language.Object) (language.Object, error) {
		return nil, nil
	}, nil)

	if err := ir.Assign("Custom.probe", method); err != nil {
		t.Fatalf("custom definition rejected a prototype method: %v", err)
	}

	created, err := definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	if got, ok := created.GetPrototype().GetObject(ctx, "probe"); !ok || got != method {
		t.Fatal("custom prototype method was not installed")
	}
}

func TestNativeStructRejectsImpl(t *testing.T) {
	definition := language.NewStruct("Native", nil, nil)
	ir := &Interpreter{
		ctx:     context.Background(),
		objects: make(map[uint32]*entry),
		runtime: hookTestRuntime{},
	}

	if err := ir.Declare("Native", definition, definition.Type(), false); err != nil {
		t.Fatal(err)
	}

	if err := ir.handleImpl(&astnode.Node{Content: "Native"}); err == nil {
		t.Fatal("impl accepted a native struct")
	}
}
