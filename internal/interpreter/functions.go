package interpreter

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleFunctionDecl(node *astnode.Node, ret ...bool) (language.Object, error) {
	var args = make([]language.FnArg, len(node.Args))
	var returnType *language.Type

	if node.ValueType != nil {
		rt, err := i.parseTypeNode(node.ValueType)
		if err != nil {
			return nil, exception.From(err, node.Debug, "failed to parse type: @err")
		}
		returnType = rt
	}

	if returnType == nil {
		returnType = language.NewUnionType(language.TypeAny, language.TypeVoid)
	}

	for j, arg := range node.Args {
		typ, err := i.parseTypeNode(arg.ValueType)
		if err != nil {
			return nil, exception.From(err, arg.Debug, "failed to parse type: @err")
		}

		if arg.FallbackValue != nil {
			val, err := i.eval(arg.FallbackValue)
			if err != nil {
				return nil, exception.From(err, arg.Debug, "failed to evaluate fallback value: @err")
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
				return nil, exception.From(err, node.Debug, "failed to declare argument: @err")
			}
		}

		ob, err := ir.Run(node.Body)
		if err != nil {
			return nil, exception.From(err, node.Debug, "function execution failed: @err")
		}
		return ob, nil
	}, node.Debug)

	if len(ret) > 0 && ret[0] {
		return fn, nil
	}

	if err := i.Declare(node.Content, fn, fn.Type(), false); err != nil {
		return nil, exception.From(err, node.Debug, "failed to declare function: @err")
	}
	return nil, nil
}

func (i *Interpreter) handleFunctionCall(node *astnode.Node) (language.Object, error) {
	if node.Content == "xdbg" {
		return xdbg(node)
	}

	fn, ok := i.GetObject(node.Content)
	if !ok {
		return nil, exception.Create("undefined function: %s(...)", node.Content).WithDebug(node.Debug).WithLevel(exception.LevelRuntime)
	}

	if fn.Type().Base() != language.ObjectTypeFunction {
		if fn.Type().Base() == language.ObjectTypeStructDefinition {
			ob, err := i.handleStructCreation(fn, node)
			if err != nil {
				return nil, exception.From(err, node.Debug, "failed to create struct: @err")
			}
			return ob, nil
		}

		return nil, exception.Create("expected function, got %s", fn.Type().Base()).WithDebug(node.Debug).WithLevel(exception.LevelRuntime)
	}

	var args = make([]language.Object, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.eval(arg)
		if err != nil {
			return nil, exception.From(err, arg.Debug, "failed to evaluate argument: @err")
		}

		if value == nil {
			return nil, exception.Create("argument %d is expected to be a value, got void", j+1).WithDebug(node.Debug)
		}

		args[j] = value.Clone()
	}

	okFn, ok := fn.(*language.Function)
	if !ok {
		return nil, exception.Create("expected function, got %s", node.Type).WithDebug(node.Debug)
	}

	value, err := okFn.Data(args)
	if err != nil {
		return nil, exception.From(err, node.Debug, fmt.Sprintf("error calling function %s: @err", node.Content))
	}

	if len(node.Children) == 1 {
		ob, err := i.getValueFromObjByNode(value, node.Children[0])
		if err != nil {
			return nil, exception.From(err, node.Children[0].Debug, "failed to get value by node")
		}
		return ob, nil
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
			return nil, exception.From(err, node.Debug, fmt.Sprintf("failed to get field %q", objID))
		}
		value = nextValue
		objID = last
	}

	proto := value.GetPrototype()
	if proto == nil {
		return nil, protoErr("%s does not have a prototype", value.Type()).WithDebug(value.Debug())
	}

	switch node.Type {
	case astnode.NodeTypeValue:
		if node.Kind == "IDENTIFIER" {
			obj, ok := proto.GetObject(objID)
			if !ok {
				return nil, runExc("cannot find property %q on %s", node.Content, value.Type()).WithDebug(node.Debug)
			}
			if len(node.Children) == 1 {
				ob, err := i.getValueFromObjByNode(obj, node.Children[0])
				if err != nil {
					return nil, exception.From(err, node.Children[0].Debug)
				}
				return ob, nil
			}
			return obj, nil
		}
	case astnode.NodeTypeFunctionCall:
		fn, ok := proto.GetObject(objID)
		if !ok {
			return nil, valueExc("function %q does not exist", node.Content).WithDebug(node.Debug)
		}

		if fn.Type().Base() != language.ObjectTypeFunction {
			return nil, typeError("expected function, got %s", node.Type).WithDebug(node.Debug)
		}

		var args = make([]language.Object, len(node.Args))
		for j, arg := range node.Args {
			value, err := i.eval(arg)
			if err != nil {
				return nil, exception.From(err, arg.Debug)
			}
			args[j] = value
		}

		okFn, ok := fn.(*language.Function)
		if !ok {
			return nil, typeError("expected function, got %s", node.Type).WithDebug(node.Debug)
		}

		value, err := okFn.Data(args)
		if err != nil {
			return nil, exception.From(err, node.Debug)
		}

		if len(node.Children) == 1 {
			ob, err := i.getValueFromObjByNode(value, node.Children[0])
			if err != nil {
				return nil, exception.From(err, node.Children[0].Debug)
			}
			return ob, nil
		}

		return value, nil
	}

	return nil, runExc("cannot get prototype for type %s with node %s", value.Type(), node.Type).WithDebug(value.Debug())
}

func (i *Interpreter) getFieldFromObjByNode(value language.Object, fields []string) (language.Object, error) {
	for _, field := range fields {
		proto := value.GetPrototype()
		if proto == nil {
			return nil, runExc("value does not have a prototype").WithDebug(value.Debug())
		}

		fieldValue, ok := proto.GetObject(field)
		if !ok {
			return nil, runExc("prototype field %q not found", field).WithDebug(value.Debug())
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
			return nil, exception.From(err, node.Debug, "invalid type node: @err")
		}
		returnType = rt
	}

	if returnType == nil {
		returnType = language.TypeVoid
	}

	for j, arg := range node.Args {
		typ, err := i.parseTypeNode(arg.ValueType)
		if err != nil {
			return nil, exception.From(err, node.Debug, "invalid type node: @err")
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
				return nil, typeError("expected %s, got %s", arg.Type(), providedArg.Type()).WithDebug(providedArg.Debug())
			}

			if err := ir.Declare(arg.Name(), providedArg, arg.Type(), true); err != nil {
				return nil, exception.From(err, providedArg.Debug())
			}
		}

		ob, err := ir.Run(node.Body)
		if err != nil {
			return nil, exception.From(err, node.Debug, "cannot run function body")
		}
		return ob, nil
	}, node.Debug)

	if node.Flags.Contains("SELFCALL") {
		var args = make([]language.Object, len(node.Children))
		for j, arg := range node.Children {
			value, err := i.eval(arg)
			if err != nil {
				return nil, exception.From(err, arg.Debug, "cannot evaluate argument")
			}

			if value == nil {
				return nil, typeError("argument %d is void, expected to be a value", j+1).WithDebug(node.Debug)
			}

			args[j] = value.Clone()
		}

		ob, err := fn.Data(args)
		if err != nil {
			return nil, exception.From(err, node.Debug, "cannot run function body")
		}
		return ob, nil
	}

	return fn, nil
}
