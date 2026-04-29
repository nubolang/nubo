package reflect

import (
	"context"
	"fmt"
	"sort"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var reflStruct *language.Struct

func StructReflect(dg *debug.Debug, ctx context.Context) language.Object {
	if reflStruct != nil {
		return reflStruct
	}

	reflStruct = language.NewStruct("Struct", []language.StructField{
		{Name: "object", Type: language.TypeAny, Private: true},
	}, dg)

	sp := reflStruct.GetPrototype().(*language.StructPrototype)
	sp.Unlock()
	defer func() {
		sp.Lock()
		sp.Implement()
	}()

	set := func(name string, value language.Object) {
		_ = sp.SetObject(ctx, name, value)
	}

	set("init", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
		n.Arg("object", n.TAny),
	).Returns(reflStruct.Type()), func(args *n.Args) (any, error) {
		self := args.Name("self")
		object := args.Name("object")

		setCtx := language.StructAllowPrivateCtx(context.Background())
		if err := self.GetPrototype().SetObject(setCtx, "object", object); err != nil {
			return nil, err
		}
		return self, nil
	}))

	set("kind", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TString), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return kindOf(obj), nil
	}))

	set("type", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TString), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return obj.Type().String(), nil
	}))

	set("typeOf", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TString), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return obj.Type().String(), nil
	}))

	set("getType", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TString), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return obj.Type().String(), nil
	}))

	set("id", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TString), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return obj.ID(), nil
	}))

	set("inspect", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TString), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return obj.Inspect(), nil
	}))

	set("string", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TString), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return obj.String(), nil
	}))

	set("value", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TAny), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return obj, nil
	}))

	set("isStruct", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TBool), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		base := obj.Type().Base()
		return base == language.ObjectTypeStructDefinition || base == language.ObjectTypeStructInstance, nil
	}))

	set("isFunction", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TBool), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		return obj.Type().Base() == language.ObjectTypeFunction, nil
	}))

	set("fields", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
		n.Arg("includePrivate", n.TBool, n.Bool(false)),
	).Returns(n.TTList(language.NewDictType(n.TString, n.TAny))), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		includePrivate := args.Name("includePrivate").Value().(bool)
		return reflectFields(obj, includePrivate, obj.Debug()), nil
	}))

	set("methods", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
		n.Arg("includePrivate", n.TBool, n.Bool(false)),
	).Returns(n.TTList(language.NewDictType(n.TString, n.TAny))), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		includePrivate := args.Name("includePrivate").Value().(bool)
		return reflectMethods(obj, includePrivate, obj.Debug()), nil
	}))

	set("members", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
		n.Arg("includePrivate", n.TBool, n.Bool(false)),
	).Returns(n.TTList(language.NewDictType(n.TString, n.TAny))), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		includePrivate := args.Name("includePrivate").Value().(bool)

		fields := reflectFields(obj, includePrivate, obj.Debug()).Data
		methods := reflectMethods(obj, includePrivate, obj.Debug()).Data

		members := make([]language.Object, 0, len(fields)+len(methods))
		members = append(members, fields...)
		members = append(members, methods...)

		return language.NewList(members, language.NewDictType(language.TypeString, language.TypeAny), obj.Debug()), nil
	}))

	set("signature", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
	).Returns(n.TAny), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		fn, ok := obj.(*language.Function)
		if !ok {
			return language.Nil, nil
		}
		return functionSignature(fn, obj.Debug()), nil
	}))

	set("has", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
		n.Arg("name", n.TString),
		n.Arg("includePrivate", n.TBool, n.Bool(false)),
	).Returns(n.TBool), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}
		proto := obj.GetPrototype()
		if proto == nil {
			return false, nil
		}

		name := args.Name("name").String()
		includePrivate := args.Name("includePrivate").Value().(bool)

		readCtx := context.Background()
		if includePrivate {
			readCtx = language.StructAllowPrivateCtx(readCtx)
		}

		_, ok := proto.GetObject(readCtx, name)
		if ok {
			return true, nil
		}

		if includePrivate {
			_, ok = proto.Objects()[name]
			return ok, nil
		}

		return false, nil
	}))

	set("get", n.Function(n.Describe(
		n.Arg("self", reflStruct.Type()),
		n.Arg("name", n.TString),
		n.Arg("includePrivate", n.TBool, n.Bool(false)),
	).Returns(n.TAny), func(args *n.Args) (any, error) {
		obj, err := reflectedObject(args.Name("self"))
		if err != nil {
			return nil, err
		}

		proto := obj.GetPrototype()
		if proto == nil {
			return language.Nil, nil
		}

		name := args.Name("name").String()
		includePrivate := args.Name("includePrivate").Value().(bool)

		readCtx := context.Background()
		if includePrivate {
			readCtx = language.StructAllowPrivateCtx(readCtx)
		}

		value, ok := proto.GetObject(readCtx, name)
		if ok {
			return value, nil
		}

		if includePrivate {
			if v, ok := proto.Objects()[name]; ok {
				return v, nil
			}
		}

		return language.Nil, nil
	}))

	return reflStruct
}

func NewStructReflection(obj language.Object, dg *debug.Debug) (language.Object, error) {
	StructReflect(dg, context.Background())

	inst, err := reflStruct.NewInstance()
	if err != nil {
		return nil, err
	}

	initObj, ok := inst.GetPrototype().GetObject(context.Background(), "init")
	if !ok {
		return nil, fmt.Errorf("reflect Struct missing init method")
	}

	initFn, ok := initObj.(*language.Function)
	if !ok {
		return nil, fmt.Errorf("reflect Struct init is not a function")
	}

	return initFn.Data(context.Background(), []language.Object{obj})
}

func reflectedObject(self language.Object) (language.Object, error) {
	if self == nil {
		return nil, fmt.Errorf("reflection object is nil")
	}

	proto := self.GetPrototype()
	if proto == nil {
		return nil, fmt.Errorf("reflection object has no prototype")
	}

	obj, ok := proto.GetObject(language.StructAllowPrivateCtx(context.Background()), "object")
	if !ok || obj == nil {
		return nil, fmt.Errorf("reflection object has no bound value")
	}

	return obj, nil
}

func kindOf(obj language.Object) string {
	if obj == nil {
		return language.ObjectTypeVoid.String()
	}

	switch obj.Type().Base() {
	case language.ObjectTypeStructDefinition:
		return "struct.definition"
	case language.ObjectTypeStructInstance:
		return "struct.instance"
	case language.ObjectTypeFunction:
		return "function"
	default:
		return obj.Type().Base().String()
	}
}

func reflectFields(obj language.Object, includePrivate bool, dg *debug.Debug) *language.List {
	var entries []language.Object

	if def, ok := obj.(*language.Struct); ok {
		for _, field := range def.Data {
			if !includePrivate && field.Private {
				continue
			}

			item := dictOf(dg, map[string]language.Object{
				"kind":    n.String("field", dg),
				"name":    n.String(field.Name, dg),
				"type":    n.String(field.Type.String(), dg),
				"private": n.Bool(field.Private, dg),
				"value":   language.Nil,
			})
			entries = append(entries, item)
		}

		return language.NewList(entries, language.NewDictType(language.TypeString, language.TypeAny), dg)
	}

	proto := obj.GetPrototype()
	if proto == nil {
		return language.NewList(nil, language.NewDictType(language.TypeString, language.TypeAny), dg)
	}

	objs := proto.Objects()
	keys := sortedKeys(objs)
	for _, name := range keys {
		value := objs[name]
		if value.Type().Base() == language.ObjectTypeFunction {
			continue
		}

		private := memberIsPrivate(proto, name)
		if private && !includePrivate {
			continue
		}

		publicValue := value
		if private && !includePrivate {
			publicValue = language.Nil
		}

		item := dictOf(dg, map[string]language.Object{
			"kind":    n.String("field", dg),
			"name":    n.String(name, dg),
			"type":    n.String(value.Type().String(), dg),
			"private": n.Bool(private, dg),
			"value":   publicValue,
		})
		entries = append(entries, item)
	}

	return language.NewList(entries, language.NewDictType(language.TypeString, language.TypeAny), dg)
}

func reflectMethods(obj language.Object, includePrivate bool, dg *debug.Debug) *language.List {
	if def, ok := obj.(*language.Struct); ok {
		return reflectStructDefinitionMethods(def, includePrivate, dg)
	}

	proto := obj.GetPrototype()
	if proto == nil {
		return language.NewList(nil, language.NewDictType(language.TypeString, language.TypeAny), dg)
	}

	var entries []language.Object
	objs := proto.Objects()
	keys := sortedKeys(objs)

	for _, name := range keys {
		value := objs[name]
		if value.Type().Base() != language.ObjectTypeFunction || name == "$convout" {
			continue
		}

		private := memberIsPrivate(proto, name)
		if private && !includePrivate {
			continue
		}

		var signature language.Object = language.Nil
		if fn, ok := value.(*language.Function); ok {
			signature = functionSignature(fn, dg)
		}

		item := dictOf(dg, map[string]language.Object{
			"kind":      n.String("method", dg),
			"name":      n.String(name, dg),
			"type":      n.String(value.Type().String(), dg),
			"private":   n.Bool(private, dg),
			"signature": signature,
			"value":     value,
		})
		entries = append(entries, item)
	}

	return language.NewList(entries, language.NewDictType(language.TypeString, language.TypeAny), dg)
}

func reflectStructDefinitionMethods(def *language.Struct, includePrivate bool, dg *debug.Debug) *language.List {
	inst, err := def.NewInstance()
	if err != nil {
		return language.NewList(nil, language.NewDictType(language.TypeString, language.TypeAny), dg)
	}

	fieldSet := make(map[string]struct{}, len(def.Data))
	for _, field := range def.Data {
		fieldSet[field.Name] = struct{}{}
	}

	proto := inst.GetPrototype()
	objs := proto.Objects()
	keys := sortedKeys(objs)

	var entries []language.Object
	for _, name := range keys {
		value := objs[name]
		if _, isField := fieldSet[name]; isField {
			continue
		}
		if value.Type().Base() != language.ObjectTypeFunction || name == "$convout" {
			continue
		}

		private := memberIsPrivate(proto, name)
		if private && !includePrivate {
			continue
		}

		var signature language.Object = language.Nil
		if fn, ok := value.(*language.Function); ok {
			signature = functionSignature(fn, dg)
		}

		item := dictOf(dg, map[string]language.Object{
			"kind":      n.String("method", dg),
			"name":      n.String(name, dg),
			"type":      n.String(value.Type().String(), dg),
			"private":   n.Bool(private, dg),
			"signature": signature,
			"value":     value,
		})
		entries = append(entries, item)
	}

	return language.NewList(entries, language.NewDictType(language.TypeString, language.TypeAny), dg)
}

func functionSignature(fn *language.Function, dg *debug.Debug) language.Object {
	args := make([]language.Object, 0, len(fn.ArgTypes))
	for _, arg := range fn.ArgTypes {
		var defaultValue language.Object = language.Nil
		hasDefault := n.Bool(false, dg)
		if arg.Default() != nil {
			defaultValue = arg.Default().Clone()
			hasDefault = n.Bool(true, dg)
		}

		args = append(args, dictOf(dg, map[string]language.Object{
			"name":       n.String(arg.Name(), dg),
			"type":       n.String(arg.Type().String(), dg),
			"hasDefault": hasDefault,
			"default":    defaultValue,
		}))
	}

	argsList := language.NewList(args, language.NewDictType(language.TypeString, language.TypeAny), dg)

	returnType := language.TypeVoid
	if fn.ReturnType != nil {
		returnType = fn.ReturnType
	}

	return dictOf(dg, map[string]language.Object{
		"args":      argsList,
		"returns":   n.String(returnType.String(), dg),
		"signature": n.String(fn.Type().String(), dg),
	})
}

func memberIsPrivate(proto language.Prototype, name string) bool {
	_, ok := proto.GetObject(context.Background(), name)
	return !ok
}

func sortedKeys(data map[string]language.Object) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func dictOf(dg *debug.Debug, data map[string]language.Object) language.Object {
	keys := make([]language.Object, 0, len(data))
	values := make([]language.Object, 0, len(data))

	sorted := make([]string, 0, len(data))
	for key := range data {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)

	for _, key := range sorted {
		keys = append(keys, n.String(key, dg))
		values = append(values, data[key])
	}

	dict, err := language.NewDict(keys, values, language.TypeString, language.TypeAny, dg)
	if err != nil {
		return language.Nil
	}
	return dict
}
