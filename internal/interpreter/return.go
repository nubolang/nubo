package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleReturn(node *astnode.Node) (language.Object, error) {
	value, err := i.evaluateExpression(node)
	if err != nil {
		return nil, err
	}

	if value != nil {
		return value, nil
	}

	return nil, nil
}
