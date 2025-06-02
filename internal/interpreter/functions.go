package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
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
		value, err := i.evaluateExpression(arg)
		if err != nil {
			return nil, err
		}
		args[j] = value
	}

	okFn, ok := fn.(*language.Function)
	if !ok {
		return nil, newErr(ErrExpectedFunction, fmt.Sprintf("got %s", node.Type), node.Debug)
	}

	value, err := okFn.Data(args)
	if err != nil {
		return nil, err
	}

	if len(node.Children) == 1 {
		return i.getValueFromObjByNode(value, node.Children[0])
	}

	return value, nil
}

func (i *Interpreter) getValueFromObjByNode(value language.Object, node *astnode.Node) (language.Object, error) {
	proto := value.GetPrototype()
	if proto == nil {
		return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot get prototype for type %s", value.Type()), value.Debug())
	}

	switch node.Type {
	case astnode.NodeTypeValue:
		if node.Kind == "IDENTIFIER" {
			obj, ok := proto.GetObject(node.Content)
			if !ok {
				return nil, newErr(ErrUnknownNode, fmt.Sprintf("Cannot find property %s on %s", node.Content, value.Type()), node.Debug)
			}
			if len(node.Children) == 1 {
				return i.getValueFromObjByNode(obj, node.Children[0])
			}
			return value, nil
		}
	case astnode.NodeTypeFunctionCall:
		fn, ok := proto.GetObject(node.Content)
		if fn.Type() != language.TypeFunction {
			return nil, newErr(ErrExpectedFunction, fmt.Sprintf("got %s", node.Type), node.Debug)
		}

		var args = make([]language.Object, len(node.Args))
		for j, arg := range node.Args {
			value, err := i.evaluateExpression(arg)
			if err != nil {
				return nil, err
			}
			args[j] = value
		}

		okFn, ok := fn.(*language.Function)
		if !ok {
			return nil, newErr(ErrExpectedFunction, fmt.Sprintf("got %s", node.Type), node.Debug)
		}

		value, err := okFn.Data(args)
		if err != nil {
			return nil, err
		}

		if len(node.Children) == 1 {
			return i.getValueFromObjByNode(value, node.Children[0])
		}

		return value, nil
	}

	return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot get prototype for type %s with node %s", value.Type(), node.Type), value.Debug())
}
