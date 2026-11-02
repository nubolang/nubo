package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func _callsite(node *astnode.Node) (language.Object, error) {
	if len(node.Args) > 0 {
		return nil, n.Err("_callsite does not accept arguments", node.Debug)
	}

	return n.Dict(map[any]any{
		"line":      node.Debug.Line,
		"column":    node.Debug.Column,
		"columnEnd": node.Debug.ColumnEnd,
		"file":      node.Debug.File,
	}, node.Debug)
}

func ref(i *Interpreter, node *astnode.Node) (language.Object, error) {
	if len(node.Args) != 1 {
		return nil, n.Err("ref only accepts one argument", node.Debug)
	}

	value, err := i.evalValue(node.Args[0])
	if err != nil {
		return nil, err
	}

	return language.NewRef(value), nil
}

func _id(i *Interpreter, node *astnode.Node) (language.Object, error) {
	if len(node.Args) != 1 {
		return nil, n.Err("_id only accepts one argument", node.Debug)
	}

	value, err := i.evalValue(node.Args[0])
	if err != nil {
		return nil, err
	}

	return language.NewString(value.ID(), node.Debug), nil
}
