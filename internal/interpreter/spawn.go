package interpreter

import (
	"log"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleSpawn(node *astnode.Node) {
	ir := NewWithParent(i, ScopeFunction, "nubo_concurrent")
	ir.Declare("__concurrent__", language.NewBool(true, node.Debug), language.TypeBool, false)

	zap.L().Debug("interpreter.spawn.start", zap.Uint("id", i.ID))
	for _, node := range node.Children {
		if _, err := ir.eval(node); err != nil {
			zap.L().Error("interpreter.spawn.error", zap.Uint("id", i.ID), zap.Error(err))
			i.Detach()
			log.Fatal(err)
		}
	}
}
