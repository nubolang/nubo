package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleTypeDecl(node *astnode.Node) error {
	typ, err := i.parseTypeNode(node.ValueType)
	name := node.Content

	if err != nil {
		return wrapRunExc(err, node.ValueType.Debug)
	}

	value := language.NewTypeObject(typ, node.Debug)
	typ.ID = name

	if err := i.Declare(name, value, value.Type(), false); err != nil {
		return wrapRunExc(err, node.Debug)
	}

	return nil
}
