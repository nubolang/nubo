package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleBlock(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.handler.block", zap.Uint("id", i.ID))
	return NewWithParent(i, ScopeBlock, "block").Run(node.Children)
}
