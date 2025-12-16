package interpreter

import (
	"context"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleStruct(node *astnode.Node) error {
	zap.L().Debug("interpreter.struct.declare.start", zap.Uint("id", i.ID), zap.String("name", node.Content))

	name := node.Content
	body := make([]language.StructField, len(node.Body))

	definition := language.NewStructBetter(name, node.Debug)

	if err := i.Declare(name, definition, definition.Type(), false); err != nil {
		zap.L().Error("interpreter.struct.declare.store", zap.Uint("id", i.ID), zap.String("name", name), zap.Error(err))
		return wrapRunExc(err, node.Debug)
	}

	for inx, field := range node.Body {
		typ, err := i.parseTypeNode(field.ValueType)
		if err != nil {
			zap.L().Error("interpreter.struct.declare.typeError", zap.Uint("id", i.ID), zap.String("name", name), zap.String("field", field.Content), zap.Error(err))
			return wrapRunExc(err, node.Debug)
		}
		priv := field.Flags.Contains("PRIVATE")

		body[inx] = language.StructField{
			Name:    field.Content,
			Type:    typ,
			Private: priv,
		}
	}

	definition.DefineFieldset(body)

	zap.L().Debug("interpreter.struct.declare.success", zap.Uint("id", i.ID), zap.String("name", name))
	return nil
}

func (i *Interpreter) handleStructCreation(obj language.Object, node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.struct.create.start", zap.Uint("id", i.ID), zap.String("struct", logObjectType(obj)))

	definition, ok := obj.(*language.Struct)
	if !ok {
		err := typeError("expected (struct), got %s", obj.Type()).WithDebug(node.Debug)
		zap.L().Error("interpreter.struct.create.invalidType", zap.Uint("id", i.ID), zap.Error(err))
		return nil, err
	}

	var args = make([]language.Object, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.eval(arg)
		if err != nil {
			zap.L().Error("interpreter.struct.create.argEval", zap.Uint("id", i.ID), zap.Int("index", j), zap.Error(err))
			return nil, wrapRunExc(err, arg.Debug)
		}
		args[j] = value.Clone()
	}

	instance, err := definition.NewInstance()
	if err != nil {
		zap.L().Error("interpreter.struct.create.instanceError", zap.Uint("id", i.ID), zap.Error(err))
		return nil, wrapRunExc(err, node.Debug)
	}

	if newer, ok := instance.GetPrototype().GetObject(context.Background(), "init"); ok {
		fn, ok := newer.(*language.Function)
		if !ok {
			err := typeError("expected function, got %s", newer.Type()).WithDebug(node.Debug)
			zap.L().Error("interpreter.struct.create.initInvalid", zap.Uint("id", i.ID), zap.Error(err))
			return nil, err
		}

		var args = make([]language.Object, len(node.Args))
		for j, arg := range node.Args {
			value, err := i.eval(arg)
			if err != nil {
				zap.L().Error("interpreter.struct.create.initArgEval", zap.Uint("id", i.ID), zap.Int("index", j), zap.Error(err))
				return nil, wrapRunExc(err, arg.Debug)
			}
			args[j] = value.Clone()
		}

		var inst language.Object
		if language.TypeCheck(instance.Type(), fn.ReturnType) {
			inst, err = fn.Data(language.StructAllowPrivateCtx(i.ctx), args)
			if err != nil {
				zap.L().Error("interpreter.struct.create.initExec", zap.Uint("id", i.ID), zap.Error(err))
				return nil, wrapRunExc(err, node.Debug)
			}
		} else if language.TypeCheck(language.TypeVoid, fn.ReturnType) {
			_, err = fn.Data(language.StructAllowPrivateCtx(i.ctx), args)
			if err != nil {
				zap.L().Error("interpreter.struct.create.initExec", zap.Uint("id", i.ID), zap.Error(err))
				return nil, wrapRunExc(err, node.Debug)
			}
			inst = instance
		} else {
			err := typeError("function return type %s does not match struct type %s|void", fn.ReturnType.String(), instance.Type().String()).WithDebug(node.Debug)
			zap.L().Error("interpreter.struct.create.initReturnType", zap.Uint("id", i.ID), zap.Error(err))
			return nil, err
		}

		if len(node.Children) == 1 {
			ob, err := i.getValueFromObjByNode(instance, node.Children[0])
			if err != nil {
				zap.L().Error("interpreter.struct.create.childError", zap.Uint("id", i.ID), zap.Error(err))
				return nil, wrapRunExc(err, node.Children[0].Debug)
			}
			zap.L().Debug("interpreter.struct.create.childReturn", zap.Uint("id", i.ID), zap.String("returnType", logObjectType(ob)))
			return ob, nil
		}

		zap.L().Debug("interpreter.struct.create.initSuccess", zap.Uint("id", i.ID), zap.String("returnType", logObjectType(inst)))
		return inst, nil
	}

	if len(node.Children) == 1 {
		ob, err := i.getValueFromObjByNode(instance, node.Children[0])
		if err != nil {
			zap.L().Error("interpreter.struct.create.childError", zap.Uint("id", i.ID), zap.Error(err))
			return nil, wrapRunExc(err, node.Children[0].Debug)
		}
		zap.L().Debug("interpreter.struct.create.childReturn", zap.Uint("id", i.ID), zap.String("returnType", logObjectType(ob)))
		return ob, nil
	}

	zap.L().Debug("interpreter.struct.create.success", zap.Uint("id", i.ID), zap.String("returnType", logObjectType(instance)))
	return instance, nil
}
