package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleImpl(node *astnode.Node) error {
	zap.L().Debug("interpreter.impl.start", zap.Uint("id", i.ID), zap.String("name", node.Content))

	name := node.Content
	definition, ok := i.GetObject(name)
	if !ok || definition.Type().Base() != language.ObjectTypeStructDefinition {
		err := runExc("cannot implement object %q", name).WithDebug(node.Debug)
		zap.L().Error("interpreter.impl.undefined", zap.Uint("id", i.ID), zap.String("name", name), zap.Error(err))
		return err
	}

	proto, ok := definition.GetPrototype().(*language.StructPrototype)
	if !ok {
		err := runExc("cannot implement object %q, no prototype found", name).WithDebug(node.Debug)
		zap.L().Error("interpreter.impl.noPrototype", zap.Uint("id", i.ID), zap.String("name", name), zap.Error(err))
		return err
	}

	if proto.Implemented() {
		err := runExc("cannot re-implement %q, already implemented", name).WithDebug(node.Debug)
		zap.L().Error("interpreter.impl.already", zap.Uint("id", i.ID), zap.String("name", name), zap.Error(err))
		return err
	}

	proto.Unlock()

	for _, child := range node.Body {
		name := child.Content
		fn, err := i.handleFunctionDecl(child, true)
		if err != nil {
			zap.L().Error("interpreter.impl.fnError", zap.Uint("id", i.ID), zap.String("method", name), zap.Error(err))
			return wrapRunExc(err, child.Debug)
		}

		ctx := i.ctx
		if child.Flags.Contains("PRIVATE") {
			if name == "init" {
				err := runExc("struct (%s) \"init\" hook method cannot be private", node.Content).WithDebug(child.Debug)
				zap.L().Error("interpreter.impl.privateInit", zap.Uint("id", i.ID), zap.String("name", node.Content), zap.Error(err))
				return err
			}
			ctx = language.StructSetPrivate(ctx)
		}

		if err := proto.SetObject(ctx, name, fn); err != nil {
			zap.L().Error("interpreter.impl.setObject", zap.Uint("id", i.ID), zap.String("method", name), zap.Error(err))
			return wrapRunExc(err, node.Debug)
		}
	}

	proto.Lock()
	proto.Implement()

	zap.L().Debug("interpreter.impl.success", zap.Uint("id", i.ID), zap.String("name", node.Content))
	return nil
}
