package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleReturn(node *astnode.Node) (language.Object, error) {
	if node.Flags.Contains("NODEVALUE") {
		node = node.Value.(*astnode.Node)
	}

	if node.Flags.Contains("VOID") {
		return language.NewSignal("return", node.Debug), nil
	}

	value, err := i.evaluateExpression(node)
	if err != nil {
		return nil, err
	}

	if value != nil {
		return value, nil
	}

	return nil, nil
}
