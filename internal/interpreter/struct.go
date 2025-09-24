package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleStruct(node *astnode.Node) error {
	name := node.Content
	body := make([]language.StructField, len(node.Body))

	for inx, field := range node.Body {
		typ, err := i.parseTypeNode(field.ValueType)
		if err != nil {
			return wrapRunExc(err, node.Debug)
		}

		body[inx] = language.StructField{
			Name: field.Content,
			Type: typ,
		}
	}

	definition := language.NewStruct(name, body, node.Debug)

	if err := i.Declare(name, definition, definition.Type(), false); err != nil {
		return wrapRunExc(err, node.Debug)
	}
	return nil
}

func (i *Interpreter) handleStructCreation(obj language.Object, node *astnode.Node) (language.Object, error) {
	definition, ok := obj.(*language.Struct)
	if !ok {
		return nil, typeError("expected (struct), got %s", obj.Type()).WithDebug(node.Debug)
	}

	var args = make([]language.Object, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.eval(arg)
		if err != nil {
			return nil, wrapRunExc(err, arg.Debug)
		}
		args[j] = value.Clone()
	}

	instance, err := definition.NewInstance()
	if err != nil {
		return nil, wrapRunExc(err, node.Debug)
	}

	if newer, ok := instance.GetPrototype().GetObject("init"); ok {
		fn, ok := newer.(*language.Function)
		if !ok {
			return nil, typeError("expected function, got %s", newer.Type()).WithDebug(node.Debug)
		}

		var args = make([]language.Object, len(node.Args))
		for j, arg := range node.Args {
			value, err := i.eval(arg)
			if err != nil {
				return nil, wrapRunExc(err, arg.Debug)
			}
			args[j] = value.Clone()
		}

		inst, err := fn.Data(args)
		if err != nil {
			return nil, wrapRunExc(err, node.Debug)
		}

		if len(node.Children) == 1 {
			ob, err := i.getValueFromObjByNode(instance, node.Children[0])
			if err != nil {
				return nil, wrapRunExc(err, node.Children[0].Debug)
			}
			return ob, nil
		}

		return inst, nil
	}

	if len(node.Children) == 1 {
		ob, err := i.getValueFromObjByNode(instance, node.Children[0])
		if err != nil {
			return nil, wrapRunExc(err, node.Children[0].Debug)
		}
		return ob, nil
	}

	return instance, nil
}
