package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleReturn(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.return.start", zap.Uint("id", i.ID), zap.String("file", i.currentFile))

	if node.Flags.Contains("NODEVALUE") {
		zap.L().Debug("interpreter.return.nodeValue", zap.Uint("id", i.ID))
		node = node.Value.(*astnode.Node)
	}

	if node.Flags.Contains("VOID") {
		zap.L().Debug("interpreter.return.void", zap.Uint("id", i.ID))
		return language.NewSignal("return", node.Debug), nil
	}

	value, err := i.eval(node)
	if err != nil {
		zap.L().Error("interpreter.return.evalError", zap.Uint("id", i.ID), zap.Error(err))
		return nil, wrapRunExc(err, node.Debug)
	}

	if value != nil {
		zap.L().Debug("interpreter.return.value", zap.Uint("id", i.ID), zap.String("returnType", logObjectType(value)))
		return value, nil
	}

	zap.L().Debug("interpreter.return.empty", zap.Uint("id", i.ID))
	return nil, nil
}
