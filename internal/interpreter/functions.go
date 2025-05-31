package interpreter

import (
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/language"
)

func (i *Interpreter) handleFunctionCall(node *astnode.Node) (language.Object, error) {
	fn, ok := i.GetObject(node.Content)
	if !ok {
		return nil, newErr(ErrUndefinedFunction, node.Content+"(...)", node.Debug)
	}

	if fn.Type() != language.TypeFunction {
		return nil, newErr(ErrExpectedFunction, fmt.Sprintf("got %s", node.Type), node.Debug)
	}

	var args = make([]language.Object, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.fromExpression(arg)
		if err != nil {
			return nil, err
		}
		args[j] = value
	}

	return fn.(*language.Function).Data(args)
}
