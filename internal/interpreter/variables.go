package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleVariableDecl(parent *astnode.Node) error {
	var (
		variableName string = parent.Content
		mutable             = parent.Kind != "CONST"
		value        language.Object
		err          error
	)

	node := parent
	if parent.Flags.Contains("NODEVALUE") {
		node = parent.Value.(*astnode.Node)
	}

	if node.Type == astnode.NodeTypeElement {
		value, err = i.evaluateElement(node)
	} else {
		value, err = i.evaluateExpression(node)
	}

	if parent.ValueType != nil {
		typ, err := i.parseTypeNode(parent.ValueType)
		if err != nil {
			return err
		}

		if !typ.Compare(value.Type()) {
			return newErr(ErrTypeMismatch, fmt.Sprintf("expected %s, got %s", typ.String(), value.Type().String()), parent.Debug)
		}
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
