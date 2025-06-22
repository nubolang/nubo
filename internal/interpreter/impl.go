package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleImpl(node *astnode.Node) error {
	name := node.Content
	definition, ok := i.GetObject(name)
	if !ok || definition.Type().Base() != language.ObjectTypeStructDefinition {
		return newErr(ErrInvalidImpl, fmt.Sprintf("Cannot implement %s", name), node.Debug)
	}

	proto := definition.GetPrototype()
	_, ok = proto.GetObject("#impl")
	if ok {
		return newErr(ErrInvalidImpl, fmt.Sprintf("Cannot re-implement %s", name), node.Debug)
	}

	proto.SetObject("#impl", language.NewBool(true, node.Debug))
	for _, child := range node.Body {
		name := child.Content
		fn, err := i.handleFunctionDecl(child, true)
		if err != nil {
			return err
		}
		if err := proto.SetObject(name, fn); err != nil {
			return err
		}
	}

	return nil
}
