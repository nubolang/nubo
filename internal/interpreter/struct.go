package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleStruct(node *astnode.Node) error {
	name := node.Content
	body := make([]language.StructField, len(node.Body))

	for inx, field := range node.Body {
		typ, err := i.parseTypeNode(field.ValueType)
		if err != nil {
			return newErr(ErrTypeMismatch, err.Error(), node.Debug)
		}

		body[inx] = language.StructField{
			Name: field.Content,
			Type: typ,
		}
	}

	definition := language.NewStruct(name, body, node.Debug)
	i.Declare(name, definition, definition.Type(), false)

	return nil
}

func (i *Interpreter) handleStructCreation(obj language.Object, node *astnode.Node) (language.Object, error) {
	definition, ok := obj.(*language.Struct)
	if !ok {
		return nil, newErr(ErrTypeMismatch, fmt.Sprintf("expected struct, got %s", obj.Type()), node.Debug)
	}

	var args = make([]language.Object, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.evaluateExpression(arg)
		if err != nil {
			return nil, err
		}
		args[j] = value.Clone()
	}

	instance, err := definition.NewInstance()
	if err != nil {
		return nil, err
	}

	return instance, nil
}
