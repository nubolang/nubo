package interpreter

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleFunctionDecl(node *astnode.Node, ret ...bool) (language.Object, error) {
	var args = make([]language.FnArg, len(node.Args))
	var returnType *language.Type

	if node.ValueType != nil {
		rt, err := i.parseTypeNode(node.ValueType)
		if err != nil {
			return nil, err
		}
		returnType = rt
	}

	if returnType == nil {
		returnType = language.NewUnionType(language.TypeAny, language.TypeVoid)
	}

	for j, arg := range node.Args {
		typ, err := i.parseTypeNode(arg.ValueType)
		if err != nil {
			return nil, err
		}

		if arg.FallbackValue != nil {
			val, err := i.eval(arg.FallbackValue)
			if err != nil {
				return nil, err
			}

			if arg.ValueType == nil {
				typ = val.Type()
			} else {
				if !typ.Compare(val.Type()) {
					return nil, typeError("expected %s but got %s", typ, val.Type()).WithDebug(arg.Debug)
				}
			}

			args[j] = &language.BasicFnArg{
				NameVal:    arg.Content,
				TypeVal:    typ,
				DefaultVal: val,
			}
		} else {
			args[j] = &language.BasicFnArg{
				NameVal: arg.Content,
				TypeVal: typ,
			}
		}
	}

	fn := language.NewTypedFunction(args, returnType, func(o []language.Object) (language.Object, error) {
		ir := NewWithParent(i, ScopeFunction)

		for j, arg := range args {
			providedArg := o[j]
			if !language.TypeCheck(arg.Type(), providedArg.Type()) {
				return nil, typeError("expected %s but got %s", arg.Type(), providedArg.Type()).WithDebug(providedArg.Debug())
			}

			if err := ir.Declare(arg.Name(), providedArg, arg.Type(), true); err != nil {
				return nil, err
			}
		}

		return ir.Run(node.Body)
	}, node.Debug)

	if len(ret) > 0 && ret[0] {
		return fn, nil
	}

	return nil, i.Declare(node.Content, fn, fn.Type(), false)
}

func (i *Interpreter) handleFunctionCall(node *astnode.Node) (language.Object, error) {
	if node.Content == "xdbg" {
		return xdbg(node)
	}

	fn, ok := i.GetObject(node.Content)
	if !ok {
		return nil, newErr(ErrUndefinedFunction, node.Content+"(...)", node.Debug)
	}

	if fn.Type().Base() != language.ObjectTypeFunction {
		if fn.Type().Base() == language.ObjectTypeStructDefinition {
			return i.handleStructCreation(fn, node)
		}

		return nil, newErr(ErrExpectedFunction, fmt.Sprintf("got %s", node.Type), node.Debug)
	}

	var args = make([]language.Object, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.eval(arg)
		if err != nil {
			return nil, err
		}

		if value == nil {
			return nil, newErr(ErrVoidAsValue, fmt.Sprintf("argument %d is void, expected to be a value", j+1), node.Debug)
		}

		args[j] = value.Clone()
	}

	okFn, ok := fn.(*language.Function)
	if !ok {
		return nil, newErr(ErrExpectedFunction, fmt.Sprintf("got %s", node.Type), node.Debug)
	}

	value, err := okFn.Data(args)
	if err != nil {
		e, msg, _ := debug.Unwrap(err)
		if msg == "" {
			msg = e.Error()
		}
		return nil, newErr(fmt.Errorf("Error calling %s(...)", node.Content), msg, node.Debug)
	}

	if len(node.Children) == 1 {
		return i.getValueFromObjByNode(value, node.Children[0])
	}

	return value, nil
}

func (i *Interpreter) getValueFromObjByNode(value language.Object, node *astnode.Node) (language.Object, error) {
	objID := node.Content
	if strings.Contains(objID, ".") {
		parts := strings.Split(objID, ".")
		last := parts[len(parts)-1]
		parts = parts[:len(parts)-1]

		nextValue, err := i.getFieldFromObjByNode(value, parts)
		if err != nil {
			return nil, err
		}
		value = nextValue
		objID = last
	}

	proto := value.GetPrototype()
	if proto == nil {
		return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot get prototype for type %s", value.Type()), value.Debug())
	}

	switch node.Type {
	case astnode.NodeTypeValue:
		if node.Kind == "IDENTIFIER" {
			obj, ok := proto.GetObject(objID)
			if !ok {
				return nil, newErr(ErrUnknownNode, fmt.Sprintf("Cannot find property %s on %s", node.Content, value.Type()), node.Debug)
			}
			if len(node.Children) == 1 {
				return i.getValueFromObjByNode(obj, node.Children[0])
			}
			return obj, nil
		}
	case astnode.NodeTypeFunctionCall:
		fn, ok := proto.GetObject(objID)
		if !ok {
			return nil, newErr(ErrUnknownNode, fmt.Sprintf("Cannot find function %s", node.Content), node.Debug)
		}

		if fn.Type().Base() != language.ObjectTypeFunction {
			return nil, newErr(ErrExpectedFunction, fmt.Sprintf("got %s", node.Type), node.Debug)
		}

		var args = make([]language.Object, len(node.Args))
		for j, arg := range node.Args {
			value, err := i.eval(arg)
			if err != nil {
				return nil, err
			}
			args[j] = value
		}

		okFn, ok := fn.(*language.Function)
		if !ok {
			return nil, newErr(ErrExpectedFunction, fmt.Sprintf("got %s", node.Type), node.Debug)
		}

		value, err := okFn.Data(args)
		if err != nil {
			return nil, err
		}

		if len(node.Children) == 1 {
			return i.getValueFromObjByNode(value, node.Children[0])
		}

		return value, nil
	}

	return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot get prototype for type %s with node %s", value.Type(), node.Type), value.Debug())
}

func (i *Interpreter) getFieldFromObjByNode(value language.Object, fields []string) (language.Object, error) {
	for _, field := range fields {
		proto := value.GetPrototype()
		if proto == nil {
			return nil, newErr(ErrUndefinedFunction, fmt.Sprintf("prototype not found"), value.Debug())
		}

		fieldValue, ok := proto.GetObject(field)
		if !ok {
			return nil, newErr(ErrPrototype, fmt.Sprintf("field %s not found", field), value.Debug())
		}
		value = fieldValue
	}

	return value, nil
}

func (i *Interpreter) createInlineFunction(node *astnode.Node) (language.Object, error) {
	var args = make([]language.FnArg, len(node.Args))
	var returnType *language.Type

	if node.ValueType != nil {
		rt, err := i.parseTypeNode(node.ValueType)
		if err != nil {
			return nil, err
		}
		returnType = rt
	}

	if returnType == nil {
		returnType = language.TypeVoid
	}

	for j, arg := range node.Args {
		typ, err := i.parseTypeNode(arg.ValueType)
		if err != nil {
			return nil, err
		}
		args[j] = &language.BasicFnArg{
			NameVal: arg.Content,
			TypeVal: typ,
		}
	}

	fn := language.NewTypedFunction(args, returnType, func(o []language.Object) (language.Object, error) {
		ir := NewWithParent(i, ScopeFunction)

		for j, arg := range args {
			providedArg := o[j]
			if !language.TypeCheck(arg.Type(), providedArg.Type()) {
				return nil, newErr(ErrTypeMismatch, fmt.Sprintf("Expected %s but got %s", arg.Type(), providedArg.Type()), providedArg.Debug())
			}

			if err := ir.Declare(arg.Name(), providedArg, arg.Type(), true); err != nil {
				return nil, err
			}
		}

		return ir.Run(node.Body)
	}, node.Debug)

	if node.Flags.Contains("SELFCALL") {
		var args = make([]language.Object, len(node.Children))
		for j, arg := range node.Children {
			value, err := i.eval(arg)
			if err != nil {
				return nil, err
			}

			if value == nil {
				return nil, newErr(ErrVoidAsValue, fmt.Sprintf("argument %d is void, expected to be a value", j+1), node.Debug)
			}

			args[j] = value.Clone()
		}

		return fn.Data(args)
	}

	return fn, nil
}
