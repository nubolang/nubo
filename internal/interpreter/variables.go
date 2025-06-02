package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleVariableDecl(node *astnode.Node) error {
	var (
		variableName string = node.Content
		mutable             = node.Kind != "CONST"
		value        language.Object
		err          error
	)

	if node.Flags.Contains("NODEVALUE") {
		node = node.Value.(*astnode.Node)
	}

	if node.Type == astnode.NodeTypeElement {
		value, err = i.evaluateElement(node)
	} else {
		value, err = i.evaluateExpression(node)
	}

	if err != nil {
		return err
	}

	zap.L().Info("Variable Declaration", zap.String("variableName", variableName), zap.Any("value", value), zap.Bool("mutable", mutable))

	return i.BindObject(variableName, value, mutable, true)
}

func (i *Interpreter) handleAssignment(node *astnode.Node) error {
	var (
		variableName string = node.Content
		value        language.Object
		err          error
	)

	if node.Flags.Contains("NODEVALUE") {
		node = node.Value.(*astnode.Node)
	}

	if node.Type == astnode.NodeTypeExpression {
		value, err = i.evaluateExpression(node)
	} else if node.Type == astnode.NodeTypeElement {
		value, err = i.evaluateElement(node)
	}

	if err != nil {
		return err
	}

	zap.L().Info("Variable Assignment", zap.String("variableName", variableName), zap.Any("value", value))

	return i.BindObject(variableName, value, false)
}

func (i *Interpreter) handleIncrement(node *astnode.Node) error {
	value, ok := i.GetObject(node.Content)
	if !ok {
		return newErr(ErrUndefinedFunction, node.Content, node.Debug)
	}

	if value.Type() != language.TypeInt {
		return newErr(ErrTypeMismatch, fmt.Sprintf("cannot increment variable with type %s", value.Type()), node.Debug)
	}

	proto := value.GetPrototype()
	incr, ok := proto.GetObject("increment")
	if !ok {
		return newErr(ErrUndefinedFunction, "cannot increment variable", node.Debug)
	}

	_, err := incr.(*language.Function).Data(nil)
	return err
}

func (i *Interpreter) handleDecrement(node *astnode.Node) error {
	value, ok := i.GetObject(node.Content)
	if !ok {
		return newErr(ErrUndefinedFunction, node.Content, node.Debug)
	}

	if value.Type() != language.TypeInt {
		return newErr(ErrTypeMismatch, fmt.Sprintf("cannot increment variable with type %s", value.Type()), node.Debug)
	}

	proto := value.GetPrototype()
	decr, ok := proto.GetObject("decrement")
	if !ok {
		return newErr(ErrUndefinedFunction, "cannot increment variable", node.Debug)
	}

	_, err := decr.(*language.Function).Data(nil)
	return err
}
