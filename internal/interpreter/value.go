package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
)

func requireValue(value language.Object, source *debug.Debug) error {
	if value == nil || value.Type().Base() == language.ObjectTypeSignal || value.Type().Base() == language.ObjectTypeVoid {
		return valueExc("expected a value, got void").WithDebug(source)
	}

	return nil
}

func (i *Interpreter) evalValue(node *astnode.Node) (language.Object, error) {
	value, err := i.eval(node)
	if err != nil {
		return nil, err
	}

	if err := requireValue(value, node.Debug); err != nil {
		return nil, err
	}

	return value, nil
}
