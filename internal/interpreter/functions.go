package interpreter

import (
	"context"
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleFunctionDecl(node *astnode.Node, ret ...bool) (language.Object, error) {
	zap.L().Debug("interpreter.function.declare.start", zap.Uint("id", i.ID), zap.String("name", node.Content))

	var args = make([]language.FnArg, len(node.Args))
	var returnType *language.Type

	if node.ValueType != nil {
		rt, err := i.parseTypeNode(node.ValueType)
		if err != nil {
			zap.L().Error("interpreter.function.declare.returnType", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
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
			zap.L().Error("interpreter.function.declare.argType", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.String("arg", arg.Content), zap.Error(err))
			return nil, exception.From(err, arg.Debug, "failed to parse type: @err")
		}

		if arg.FallbackValue != nil {
			val, err := i.eval(arg.FallbackValue)
			if err != nil {
				zap.L().Error("interpreter.function.declare.argFallback", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.String("arg", arg.Content), zap.Error(err))
				return nil, exception.From(err, arg.Debug, "failed to evaluate fallback value: @err")
			}

			if arg.ValueType == nil {
				typ = val.Type()
			} else {
				if !typ.Compare(val.Type()) {
					err := typeError("expected %s but got %s", typ, val.Type()).WithDebug(arg.Debug)
					zap.L().Error("interpreter.function.declare.argMismatch", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.String("arg", arg.Content), zap.Error(err))
					return nil, err
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

		zap.L().Debug("interpreter.function.declare.arg", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.String("arg", arg.Content), zap.Int("index", j))
	}

	fn := language.NewTypedFunction(args, returnType, func(ctx context.Context, o []language.Object) (language.Object, error) {
		ir := NewWithParent(i, ScopeFunction)
		ir.ctx = ctx

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
		zap.L().Debug("interpreter.function.declare.inlineReturn", zap.Uint("id", i.ID), zap.String("name", node.Content))
		return fn, nil
	}

	if err := i.Declare(node.Content, fn, fn.Type(), false); err != nil {
		zap.L().Error("interpreter.function.declare.store", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
		return nil, exception.From(err, node.Debug, "failed to declare function: @err")
	}
	zap.L().Debug("interpreter.function.declare.success", zap.Uint("id", i.ID), zap.String("name", node.Content))
	return nil, nil
}

func (i *Interpreter) handleFunctionCall(node *astnode.Node) (language.Object, error) {
	if node.Content == "xdbg" {
		zap.L().Debug("interpreter.function.call.xdbg", zap.Uint("id", i.ID))
		return xdbg(node)
	}

	zap.L().Debug("interpreter.function.call.start", zap.Uint("id", i.ID), zap.String("name", node.Content))

	fn, ok := i.GetObject(node.Content)
	if !ok {
		err := exception.Create("undefined function: %s(...)", node.Content).WithDebug(node.Debug).WithLevel(exception.LevelRuntime)
		zap.L().Error("interpreter.function.call.undefined", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
		return nil, err
	}

	if fn.Type().Base() != language.ObjectTypeFunction {
		if fn.Type().Base() == language.ObjectTypeStructDefinition {
			ob, err := i.handleStructCreation(fn, node)
			if err != nil {
				zap.L().Error("interpreter.function.call.structError", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
				return nil, exception.From(err, node.Debug, "failed to create struct: @err")
			}
			zap.L().Debug("interpreter.function.call.structReturn", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.String("returnType", logObjectType(ob)))
			return ob, nil
		}

		err := exception.Create("expected function, got %s", fn.Type().Base()).WithDebug(node.Debug).WithLevel(exception.LevelRuntime)
		zap.L().Error("interpreter.function.call.invalidType", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
		return nil, err
	}

	var (
		args      = make([]language.Object, 0, len(node.Args))
		namedArgs = make(map[string]language.Object)
	)
	for j, arg := range node.Args {
		value, err := i.eval(arg)
		if err != nil {
			zap.L().Error("interpreter.function.call.argEval", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Int("index", j), zap.Error(err))
			return nil, exception.From(err, arg.Debug, "failed to evaluate argument: @err")
		}

		if value == nil {
			err := exception.Create("argument %d is expected to be a value, got void", j+1).WithDebug(node.Debug)
			zap.L().Error("interpreter.function.call.argVoid", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Int("index", j), zap.Error(err))
			return nil, err
		}

		if arg.Kind == "NAMED_ARG" {
			namedArgs[arg.ArgName] = value.Clone()
		} else {
			args = append(args, value.Clone())
		}
	}

	okFn, ok := fn.(*language.Function)
	if !ok {
		err := exception.Create("expected function, got %s", node.Type).WithDebug(node.Debug)
		zap.L().Error("interpreter.function.call.nonFunction", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
		return nil, err
	}

	value, err := okFn.Call(i.ctx, args, namedArgs)
	if err != nil {
		zap.L().Error("interpreter.function.call.execError", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
		return nil, exception.From(err, node.Debug, fmt.Sprintf("error calling function %s: @err", node.Content))
	}

	if len(node.Children) == 1 {
		ob, err := i.getValueFromObjByNode(value, node.Children[0])
		if err != nil {
			zap.L().Error("interpreter.function.call.childAccess", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
			return nil, exception.From(err, node.Children[0].Debug, "failed to get value by node")
		}
		zap.L().Debug("interpreter.function.call.childReturn", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.String("returnType", logObjectType(ob)))
		return ob, nil
	}

	zap.L().Debug("interpreter.function.call.success", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.String("returnType", logObjectType(value)))
	return value, nil
}

func (i *Interpreter) getValueFromObjByNode(value language.Object, node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.function.access.start", zap.Uint("id", i.ID), zap.String("valueType", logObjectType(value)), zap.String("nodeType", string(node.Type)), zap.String("content", node.Content))

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
		err := protoErr("%s does not have a prototype", value.Type()).WithDebug(value.Debug())
		zap.L().Error("interpreter.function.access.noPrototype", zap.Uint("id", i.ID), zap.String("valueType", logObjectType(value)), zap.Error(err))
		return nil, err
	}

	switch node.Type {
	case astnode.NodeTypeValue:
		if node.Kind == "IDENTIFIER" {
			obj, ok := proto.GetObject(i.ctx, objID)
			if !ok {
				err := runExc("cannot find property %q on %s", node.Content, value.Type()).WithDebug(node.Debug)
				zap.L().Error("interpreter.function.access.missingProperty", zap.Uint("id", i.ID), zap.String("property", node.Content), zap.String("valueType", logObjectType(value)), zap.Error(err))
				return nil, err
			}
			if len(node.Children) == 1 {
				ob, err := i.getValueFromObjByNode(obj, node.Children[0])
				if err != nil {
					zap.L().Error("interpreter.function.access.childError", zap.Uint("id", i.ID), zap.Error(err))
					return nil, exception.From(err, node.Children[0].Debug)
				}
				zap.L().Debug("interpreter.function.access.childReturn", zap.Uint("id", i.ID), zap.String("property", node.Content), zap.String("returnType", logObjectType(ob)))
				return ob, nil
			}
			zap.L().Debug("interpreter.function.access.return", zap.Uint("id", i.ID), zap.String("property", node.Content), zap.String("returnType", logObjectType(obj)))
			return obj, nil
		}
	case astnode.NodeTypeFunctionCall:
		fn, ok := proto.GetObject(i.ctx, objID)
		if !ok {
			err := valueExc("function %q does not exist", node.Content).WithDebug(node.Debug)
			zap.L().Error("interpreter.function.access.fnMissing", zap.Uint("id", i.ID), zap.String("function", node.Content), zap.Error(err))
			return nil, err
		}

		if fn.Type().Base() != language.ObjectTypeFunction {
			err := typeError("expected function, got %s", node.Type).WithDebug(node.Debug)
			zap.L().Error("interpreter.function.access.fnInvalidType", zap.Uint("id", i.ID), zap.String("function", node.Content), zap.Error(err))
			return nil, err
		}

		var args = make([]language.Object, len(node.Args))
		for j, arg := range node.Args {
			value, err := i.eval(arg)
			if err != nil {
				zap.L().Error("interpreter.function.access.fnArgEval", zap.Uint("id", i.ID), zap.String("function", node.Content), zap.Int("index", j), zap.Error(err))
				return nil, exception.From(err, arg.Debug)
			}
			args[j] = value
		}

		okFn, ok := fn.(*language.Function)
		if !ok {
			err := typeError("expected function, got %s", node.Type).WithDebug(node.Debug)
			zap.L().Error("interpreter.function.access.fnCast", zap.Uint("id", i.ID), zap.String("function", node.Content), zap.Error(err))
			return nil, err
		}

		value, err := okFn.Data(i.ctx, args)
		if err != nil {
			zap.L().Error("interpreter.function.access.fnExec", zap.Uint("id", i.ID), zap.String("function", node.Content), zap.Error(err))
			return nil, exception.From(err, node.Debug)
		}

		if len(node.Children) == 1 {
			ob, err := i.getValueFromObjByNode(value, node.Children[0])
			if err != nil {
				zap.L().Error("interpreter.function.access.fnChildError", zap.Uint("id", i.ID), zap.String("function", node.Content), zap.Error(err))
				return nil, exception.From(err, node.Children[0].Debug)
			}
			zap.L().Debug("interpreter.function.access.fnChildReturn", zap.Uint("id", i.ID), zap.String("function", node.Content), zap.String("returnType", logObjectType(ob)))
			return ob, nil
		}

		zap.L().Debug("interpreter.function.access.fnReturn", zap.Uint("id", i.ID), zap.String("function", node.Content), zap.String("returnType", logObjectType(value)))
		return value, nil
	}

	err := runExc("cannot get prototype for type %s with node %s", value.Type(), node.Type).WithDebug(value.Debug())
	zap.L().Error("interpreter.function.access.unsupportedNode", zap.Uint("id", i.ID), zap.String("nodeType", string(node.Type)), zap.Error(err))
	return nil, err
}

func (i *Interpreter) getFieldFromObjByNode(value language.Object, fields []string) (language.Object, error) {
	zap.L().Debug("interpreter.function.field.start", zap.Uint("id", i.ID), zap.String("valueType", logObjectType(value)), zap.String("path", strings.Join(fields, ".")))

	for _, field := range fields {
		proto := value.GetPrototype()
		if proto == nil {
			err := runExc("value does not have a prototype").WithDebug(value.Debug())
			zap.L().Error("interpreter.function.field.noPrototype", zap.Uint("id", i.ID), zap.String("path", strings.Join(fields, ".")), zap.Error(err))
			return nil, err
		}

		fieldValue, ok := proto.GetObject(i.ctx, field)
		if !ok {
			err := runExc("prototype field %q not found", field).WithDebug(value.Debug())
			zap.L().Error("interpreter.function.field.missing", zap.Uint("id", i.ID), zap.String("field", field), zap.Error(err))
			return nil, err
		}
		value = fieldValue
	}

	zap.L().Debug("interpreter.function.field.success", zap.Uint("id", i.ID), zap.String("returnType", logObjectType(value)))
	return value, nil
}

func (i *Interpreter) createInlineFunction(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.function.inline.start", zap.Uint("id", i.ID), zap.Bool("selfCall", node.Flags.Contains("SELFCALL")))

	var args = make([]language.FnArg, len(node.Args))
	var returnType *language.Type

	if node.ValueType != nil {
		rt, err := i.parseTypeNode(node.ValueType)
		if err != nil {
			zap.L().Error("interpreter.function.inline.returnType", zap.Uint("id", i.ID), zap.Error(err))
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
			zap.L().Error("interpreter.function.inline.argType", zap.Uint("id", i.ID), zap.Int("index", j), zap.Error(err))
			return nil, exception.From(err, node.Debug, "invalid type node: @err")
		}
		args[j] = &language.BasicFnArg{
			NameVal: arg.Content,
			TypeVal: typ,
		}
	}

	fn := language.NewTypedFunction(args, returnType, func(ctx context.Context, o []language.Object) (language.Object, error) {
		ir := NewWithParent(i, ScopeFunction)
		ir.ctx = ctx

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
				zap.L().Error("interpreter.function.inline.selfCallEval", zap.Uint("id", i.ID), zap.Int("index", j), zap.Error(err))
				return nil, exception.From(err, arg.Debug, "cannot evaluate argument")
			}

			if value == nil {
				err := typeError("argument %d is void, expected to be a value", j+1).WithDebug(node.Debug)
				zap.L().Error("interpreter.function.inline.selfCallVoid", zap.Uint("id", i.ID), zap.Int("index", j), zap.Error(err))
				return nil, err
			}

			args[j] = value.Clone()
		}

		ob, err := fn.Data(i.ctx, args)
		if err != nil {
			zap.L().Error("interpreter.function.inline.selfCallExec", zap.Uint("id", i.ID), zap.Error(err))
			return nil, exception.From(err, node.Debug, "cannot run function body")
		}
		zap.L().Debug("interpreter.function.inline.selfCallReturn", zap.Uint("id", i.ID), zap.String("returnType", logObjectType(ob)))
		return ob, nil
	}

	zap.L().Debug("interpreter.function.inline.success", zap.Uint("id", i.ID))
	return fn, nil
}
