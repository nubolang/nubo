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

	var typ *language.Type

	if parent.ValueType != nil {
		typ, err = i.parseTypeNode(parent.ValueType)

		if err != nil {
			return err
		}
	}

	if node.Type == astnode.NodeTypeElement {
		value, err = i.evaluateElement(node)
	} else if node.Type == astnode.NodeTypeList {
		var elType = language.TypeAny
		if typ != nil && typ.Base() == language.ObjectTypeList {
			elType = typ.Element
		}
		value, err = i.evalList(node, elType)
	} else if node.Type == astnode.NodeTypeDict {
		var (
			keyType   *language.Type
			valueType *language.Type
		)

		if typ != nil && typ.Base() == language.ObjectTypeDict {
			keyType = typ.Key
			valueType = typ.Element
		}

		value, err = i.evalDict(node, keyType, valueType)
	} else {
		value, err = i.evaluateExpression(node)
	}

	if err != nil {
		return err
	}

	if value == nil {
		return newErr(ErrValueError, fmt.Sprint("void is not assignable to a variable"), node.Debug)
	}

	if typ == nil {
		typ = value.Type()
	}

	if !typ.Compare(value.Type()) {
		return newErr(ErrTypeMismatch, fmt.Sprintf("expected %s, got %s", typ.String(), value.Type().String()), parent.Debug)
	}

	zap.L().Info("Variable Declaration", zap.String("variableName", variableName), zap.Any("value", value), zap.Bool("mutable", mutable))

	return i.Declare(variableName, value.Clone(), typ, mutable)
}

func (i *Interpreter) handleAssignment(node *astnode.Node) error {
	var (
		variableName string = node.Content
		arrayAccess         = node.ArrayAccess
	)

	if node.Flags.Contains("NODEVALUE") {
		node = node.Value.(*astnode.Node)
	}

	value, err := i.eval(node)
	if err != nil {
		return err
	}

	zap.L().Info("Variable Assignment", zap.String("variableName", variableName), zap.Any("value", value))

	if len(arrayAccess) > 0 {
		lookUp, ok := i.GetObject(variableName)
		if !ok {
			return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", variableName), nil)
		}

		return i.setAccess(lookUp, arrayAccess, value)
	}

	return i.Assign(variableName, value)
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

func (i *Interpreter) setAccess(current language.Object, access []*astnode.Node, value language.Object) error {
	for _, part := range access[:len(access)-1] {
		obj, err := i.eval(part)
		if err != nil {
			return err
		}

		proto := current.GetPrototype()
		if proto == nil {
			return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined property %s", obj.String()), nil)
		}

		getter, ok := current.GetPrototype().GetObject("__get__")
		if !ok {
			return newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
		}

		getterFn, ok := getter.(*language.Function)
		if !ok {
			return newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
		}

		value, err := getterFn.Data([]language.Object{obj})
		if err != nil {
			return err
		}

		current = value
	}

	last := access[len(access)-1]
	obj, err := i.eval(last)
	if err != nil {
		return err
	}

	proto := current.GetPrototype()
	if proto == nil {
		return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined property %s", obj.String()), nil)
	}

	setter, ok := proto.GetObject("__set__")
	if !ok {
		return newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
	}

	setterFn, ok := setter.(*language.Function)
	if !ok {
		return newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
	}

	_, err = setterFn.Data([]language.Object{obj, value})
	return err
}
