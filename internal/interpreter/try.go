package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func (i *Interpreter) handleTry(node *astnode.Node) (language.Object, error) {
	ret, err := i.Run(node.Body)
	if err != nil {
		_, msg, dg := debug.Unwrap(err)

		dictErr, err := n.Dict(map[any]any{
			"message": msg,
			"file":    dg.File,
			"line":    dg.Line,
			"column":  dg.Column,
		}, node.Debug)

		if err != nil {
			return nil, err
		}

		return nil, i.Declare(node.Content, dictErr, language.TypeAny, true)
	}

	return ret, i.Declare(node.Content, language.Nil, language.TypeAny, true)
}
