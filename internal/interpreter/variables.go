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
			return wrapRunExc(err, parent.ValueType.Debug)
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
			valueType = typ.Value
		}

		value, err = i.evalDict(node, keyType, valueType)
	} else {
		value, err = i.eval(node)
	}

	if err != nil {
		return wrapRunExc(err, node.Debug)
	}

	if value == nil {
		return valueExc("void is not assignable to a variable").WithDebug(node.Debug)
	}

	if typ == nil {
		typ = value.Type()
	}

	if !typ.Compare(value.Type()) {
		return typeError("expected %s, got %s", typ.String(), value.Type().String()).WithDebug(parent.Debug)
	}

	zap.L().Info("Variable Declaration", zap.String("variableName", variableName), zap.Any("value", value), zap.Bool("mutable", mutable))

	if err := i.Declare(variableName, value.Clone(), typ, mutable); err != nil {
		return wrapRunExc(err, node.Debug)
	}
	return nil
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
		return wrapRunExc(err, node.Debug)
	}

	zap.L().Info("Variable Assignment", zap.String("variableName", variableName), zap.Any("value", value))

	if len(arrayAccess) > 0 {
		lookUp, ok := i.GetObject(variableName)
		if !ok {
			return runExc("undefined variable %s", variableName).WithDebug(node.Debug)
		}

		return i.setAccess(lookUp, arrayAccess, value)
	}

	if err := i.Assign(variableName, value); err != nil {
		return wrapRunExc(err, node.Debug)
	}
	return nil
}

func (i *Interpreter) handleIncrement(node *astnode.Node) error {
	value, ok := i.GetObject(node.Content)
	if !ok {
		return runExc("variable %q not declared", node.Content).WithDebug(node.Debug)
	}

	proto := value.GetPrototype()
	incr, ok := proto.GetObject("increment")
	if !ok {
		return runExc("cannot increment variable %q: no implementation for increment()", node.Content).WithDebug(node.Debug)
	}

	if _, err := incr.(*language.Function).Data(nil); err != nil {
		return wrapRunExc(err, node.Debug, fmt.Sprintf("cannot increment %q: @err", node.Content))
	}
	return nil
}

func (i *Interpreter) handleDecrement(node *astnode.Node) error {
	value, ok := i.GetObject(node.Content)
	if !ok {
		return runExc("variable %q not declared", node.Content).WithDebug(node.Debug)
	}

	proto := value.GetPrototype()
	decr, ok := proto.GetObject("decrement")
	if !ok {
		return runExc("cannot decrement variable %q: no implementation for decrement()", node.Content).WithDebug(node.Debug)
	}

	_, err := decr.(*language.Function).Data(nil)
	return err
}

func (i *Interpreter) setAccess(current language.Object, access []*astnode.Node, value language.Object) error {
	for _, part := range access[:len(access)-1] {
		obj, err := i.eval(part)
		if err != nil {
			return wrapRunExc(err, current.Debug())
		}

		proto := current.GetPrototype()
		if proto == nil {
			return runExc("undefined property %q", obj.String()).WithDebug(current.Debug())
		}

		if current.GetPrototype() == nil {
			return runExc("cannot operate on variable: no implementation for __get__()").WithDebug(obj.Debug())
		}

		getter, ok := current.GetPrototype().GetObject("__get__")
		if !ok {
			return runExc("cannot operate on variable: no implementation for __get__()").WithDebug(obj.Debug())
		}

		getterFn, ok := getter.(*language.Function)
		if !ok {
			return runExc("cannot operate on variable: bad implementation for __get__()").WithDebug(obj.Debug())
		}

		value, err := getterFn.Data([]language.Object{obj})
		if err != nil {
			return wrapRunExc(err, obj.Debug())
		}

		current = value
	}

	last := access[len(access)-1]
	obj, err := i.eval(last)
	if err != nil {
		return wrapRunExc(err, current.Debug())
	}

	proto := current.GetPrototype()
	if proto == nil {
		return runExc("undefined property %s", obj.String()).WithDebug(current.Debug())
	}

	setter, ok := proto.GetObject("__set__")
	if !ok {
		return runExc("cannot operate on variable: no implementation for __set__()").WithDebug(obj.Debug())
	}

	setterFn, ok := setter.(*language.Function)
	if !ok {
		return runExc("cannot operate on variable: bad implementation for __set__()").WithDebug(obj.Debug())
	}

	if _, err = setterFn.Data([]language.Object{obj, value}); err != nil {
		return wrapRunExc(err, obj.Debug())
	}

	return nil
}
