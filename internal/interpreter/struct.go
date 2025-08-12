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

	return i.Declare(name, definition, definition.Type(), false)
}

func (i *Interpreter) handleStructCreation(obj language.Object, node *astnode.Node) (language.Object, error) {
	definition, ok := obj.(*language.Struct)
	if !ok {
		return nil, newErr(ErrTypeMismatch, fmt.Sprintf("expected struct, got %s", obj.Type()), node.Debug)
	}

	var args = make([]language.Object, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.eval(arg)
		if err != nil {
			return nil, err
		}
		args[j] = value.Clone()
	}

	instance, err := definition.NewInstance()
	if err != nil {
		return nil, newErr(ErrStructInstantiation, err.Error(), node.Debug)
	}

	if newer, ok := instance.GetPrototype().GetObject("init"); ok {
		fn, ok := newer.(*language.Function)
		if !ok {
			return nil, newErr(ErrTypeMismatch, fmt.Sprintf("expected function, got %s", newer.Type()), node.Debug)
		}

		var args = make([]language.Object, len(node.Args))
		for j, arg := range node.Args {
			value, err := i.eval(arg)
			if err != nil {
				return nil, err
			}
			args[j] = value.Clone()
		}

		inst, err := fn.Data(args)
		if err != nil {
			return nil, err
		}

		if len(node.Children) == 1 {
			return i.getValueFromObjByNode(instance, node.Children[0])
		}

		return inst, nil
	}

	if len(node.Children) == 1 {
		return i.getValueFromObjByNode(instance, node.Children[0])
	}

	return instance, nil
}
