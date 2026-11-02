package reflect

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func installOperations(set func(string, language.Object), mirrorType *language.Type) {
	set("set", n.Function(n.Describe(
		n.Arg("self", mirrorType), n.Arg("name", n.TString), n.Arg("value", n.TAny),
		n.Arg("includePrivate", n.TBool, n.Bool(false)),
	).Returns(mirrorType), func(a *n.Args) (any, error) {
		obj, err := reflectedObject(a.Name("self"))
		if err != nil {
			return nil, err
		}
		instance, ok := obj.(*language.StructInstance)
		if !ok {
			return nil, fmt.Errorf("set requires a struct instance")
		}
		name := a.Name("name").String()
		fieldFound := false
		for _, field := range instance.Definition().Data {
			if field.Name == name {
				fieldFound = true
				break
			}
		}
		if !fieldFound {
			return nil, fmt.Errorf("field %q not found", name)
		}
		ctx := reflectionAccess(a)
		value, err := bindingCopy(ctx, a.Name("value"))
		if err != nil {
			return nil, err
		}

		if err := instance.GetPrototype().SetObject(ctx, name, value); err != nil {
			return nil, err
		}
		return a.Name("self"), nil
	}))
	set("call", n.Function(n.Describe(
		n.Arg("self", mirrorType), n.Arg("name", n.TString),
		n.Arg("args", n.TList, language.NewList(nil, n.TAny, nil)),
		n.Arg("includePrivate", n.TBool, n.Bool(false)),
	).Returns(n.TAny), func(a *n.Args) (any, error) {
		obj, err := reflectedObject(a.Name("self"))
		if err != nil {
			return nil, err
		}
		if _, ok := obj.(*language.Struct); ok {
			return nil, fmt.Errorf("call requires an instance; use new to construct one")
		}
		proto := obj.GetPrototype()
		if proto == nil {
			return nil, fmt.Errorf("value has no methods")
		}
		name := a.Name("name").String()
		method, ok := proto.GetObject(reflectionAccess(a), name)
		if !ok {
			return nil, fmt.Errorf("method %q not found or private", name)
		}
		fn, ok := method.(*language.Function)
		if !ok {
			return nil, fmt.Errorf("member %q is not callable", name)
		}
		values, err := reflectionArgs(a)
		if err != nil {
			return nil, err
		}
		return invoke(a.Context(), fn, values)
	}))
	set("new", n.Function(n.Describe(
		n.Arg("self", mirrorType), n.Arg("args", n.TList, language.NewList(nil, n.TAny, nil)),
	).Returns(n.TAny), func(a *n.Args) (any, error) {
		obj, err := reflectedObject(a.Name("self"))
		if err != nil {
			return nil, err
		}
		def, ok := obj.(*language.Struct)
		if !ok {
			return nil, fmt.Errorf("new requires a struct definition")
		}
		instance, err := def.NewInstance()
		if err != nil {
			return nil, err
		}
		values, err := reflectionArgs(a)
		if err != nil {
			return nil, err
		}
		init, ok := instance.GetPrototype().GetObject(language.StructAllowPrivateCtx(a.Context()), "init")
		if !ok {
			if len(values) != 0 {
				return nil, fmt.Errorf("struct %s has no initializer; expected 0 arguments", def.Name)
			}
			return instance, nil
		}
		fn, ok := init.(*language.Function)
		if !ok {
			return nil, fmt.Errorf("initializer is not callable")
		}
		if fn.ReturnType == nil || (!language.TypeCheck(instance.Type(), fn.ReturnType) && fn.ReturnType.Base() != language.ObjectTypeVoid) {
			return nil, fmt.Errorf("initializer must return %s or void", instance.Type())
		}
		result, err := invoke(language.StructAllowPrivateCtx(a.Context()), fn, values)
		if err != nil {
			return nil, err
		}
		if fn.ReturnType.Base() == language.ObjectTypeVoid {
			return instance, nil
		}
		return result, nil
	}))
	set("implements", n.Function(n.Describe(
		n.Arg("self", mirrorType), n.Arg("contract", n.TAny),
	).Returns(n.TBool), func(a *n.Args) (any, error) {
		obj, err := reflectedObject(a.Name("self"))
		if err != nil {
			return nil, err
		}
		return implementsContract(obj, a.Name("contract"), a.Context())
	}))
}

func reflectionAccess(a *n.Args) context.Context {
	ctx := a.Context()
	if a.Name("includePrivate").Value().(bool) {
		return language.StructAllowPrivateCtx(ctx)
	}
	// Reflections created inside methods must not implicitly bypass privacy.
	return context.WithValue(ctx, "unlock_struct_private", false)
}

func bindingCopy(ctx context.Context, obj language.Object) (language.Object, error) {
	if obj.Type().Base() == language.ObjectTypeStructInstance {
		return obj, nil
	}

	return language.Clone(ctx, obj)
}

func invoke(ctx context.Context, fn *language.Function, values []language.Object) (language.Object, error) {
	if fn.ReturnType != nil && len(values) > len(fn.ArgTypes) {
		return nil, fmt.Errorf("expected at most %d arguments, got %d", len(fn.ArgTypes), len(values))
	}
	args := make([]language.Object, len(values))
	for i, value := range values {
		bound, err := bindingCopy(ctx, value)
		if err != nil {
			return nil, err
		}

		args[i] = bound
	}
	result, err := fn.Data(ctx, args)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Type().Base() == language.ObjectTypeSignal {
		return language.Nil, nil
	}
	return result, nil
}

func contractSubject(obj language.Object) (language.Object, error) {
	if def, ok := obj.(*language.Struct); ok {
		return def.NewInstance()
	}
	return obj, nil
}

func implementsContract(obj, contract language.Object, ctx context.Context) (bool, error) {
	candidate, err := contractSubject(obj)
	if err != nil {
		return false, err
	}
	// Always check public members, independent of the caller's private context.
	ctx = context.WithValue(ctx, "unlock_struct_private", false)
	proto := candidate.GetPrototype()
	matches := func(name string, expected *language.Type) bool {
		if proto == nil {
			return false
		}
		member, ok := proto.GetObject(ctx, name)
		return ok && language.TypeCheck(expected, member.Type())
	}
	if alias, ok := contract.(*language.TypeObject); ok && alias.Data != nil && alias.Data.Base() == language.ObjectTypeIface && alias.Data.Iface != nil {
		for _, method := range alias.Data.Iface.Functions {
			if !matches(method.Name, language.NewFunctionType(method.Returns, method.Args...)) {
				return false, nil
			}
		}
		return true, nil
	}
	if shape, ok := contract.(*language.Dict); ok {
		matched := true
		var descriptorErr error
		shape.Data.Iterate(func(key, value language.Object) bool {
			if key.Type().Base() != language.ObjectTypeString {
				descriptorErr = fmt.Errorf("contract member names must be strings")
				return false
			}
			var expected *language.Type
			switch descriptor := value.(type) {
			case *language.TypeObject:
				expected = descriptor.Data
			case *language.Struct:
				expected = descriptor.Type()
			case *language.Function:
				expected = descriptor.ReturnType
			}
			if expected == nil {
				descriptorErr = fmt.Errorf("contract member %q must specify a type alias, struct, or typed function", key.String())
				return false
			}
			if !matches(key.String(), expected) {
				matched = false
			}
			return true
		})
		return matched && descriptorErr == nil, descriptorErr
	}
	return false, fmt.Errorf("contract must be an interface type alias or a dictionary of member types")
}

func reflectionArgs(a *n.Args) ([]language.Object, error) {
	values, ok := a.Name("args").(*language.List)
	if !ok {
		return nil, fmt.Errorf("args must be a list")
	}
	return values.Data, nil
}
