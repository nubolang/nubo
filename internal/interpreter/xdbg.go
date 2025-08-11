package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func xdbg(node *astnode.Node) (language.Object, error) {
	if len(node.Args) > 0 {
		return nil, n.Err("xdbg does not accept arguments", node.Debug)
	}

	return n.Dict(map[any]any{
		"line":   node.Debug.Line,
		"column": node.Debug.Column,
		"file":   node.Debug.File,
	}, node.Debug)
}
