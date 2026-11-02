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

	if value == nil || value.Type().BaseType == language.ObjectTypeSignal {
		return valueExc("void is not assignable to a variable").WithDebug(node.Debug)
	}

	if typ == nil {
		typ = value.Type()
	}

	if !typ.Compare(value.Type()) {
		return typeError("expected %s, got %s", typ.String(), value.Type().String()).WithDebug(parent.Debug)
	}

	zap.L().Debug("interpreter.variables.declare", zap.Uint("id", i.ID), zap.String("name", variableName), zap.Stringer("value", loggedValue{i.ctx, value}), zap.Bool("mutable", mutable))

	bound, err := i.cloneBinding(value)
	if err != nil {
		return wrapRunExc(err, node.Debug)
	}

	if err := i.Declare(variableName, bound, typ, mutable); err != nil {
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

	value, err := i.evalValue(node)
	if err != nil {
		return wrapRunExc(err, node.Debug)
	}

	zap.L().Debug("interpreter.variables.assign", zap.Uint("id", i.ID), zap.String("name", variableName), zap.Stringer("value", loggedValue{i.ctx, value}))

	if len(arrayAccess) > 0 {
		lookUp, ok, err := i.lookupObject(variableName)
		if err != nil {
			return wrapRunExc(err, node.Debug)
		}

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
	value, ok, err := i.lookupObject(node.Content)
	if err != nil {
		return wrapRunExc(err, node.Debug)
	}

	if !ok {
		return runExc("variable %q not declared", node.Content).WithDebug(node.Debug)
	}

	proto := value.GetPrototype()
	if proto == nil {
		return runExc("cannot increment variable %q: no prototype", node.Content).WithDebug(node.Debug)
	}
	incr, ok := proto.GetObject(i.ctx, "increment")
	if !ok {
		return runExc("cannot increment variable %q: no implementation for increment()", node.Content).WithDebug(node.Debug)
	}

	incrFn, ok := incr.(*language.Function)
	if !ok {
		return runExc("cannot increment variable %q: bad implementation for increment()", node.Content).WithDebug(node.Debug)
	}

	if _, err := incrFn.Data(i.ctx, nil); err != nil {
		return wrapRunExc(err, node.Debug, fmt.Sprintf("cannot increment %q: @err", node.Content))
	}
	return nil
}

func (i *Interpreter) handleDecrement(node *astnode.Node) error {
	value, ok, err := i.lookupObject(node.Content)
	if err != nil {
		return wrapRunExc(err, node.Debug)
	}

	if !ok {
		return runExc("variable %q not declared", node.Content).WithDebug(node.Debug)
	}

	proto := value.GetPrototype()
	if proto == nil {
		return runExc("cannot decrement variable %q: no prototype", node.Content).WithDebug(node.Debug)
	}
	decr, ok := proto.GetObject(i.ctx, "decrement")
	if !ok {
		return runExc("cannot decrement variable %q: no implementation for decrement()", node.Content).WithDebug(node.Debug)
	}

	decrFn, ok := decr.(*language.Function)
	if !ok {
		return runExc("cannot decrement variable %q: bad implementation for decrement()", node.Content).WithDebug(node.Debug)
	}

	_, err = decrFn.Data(i.ctx, nil)
	return err
}

func (i *Interpreter) setAccess(current language.Object, access []*astnode.Node, value language.Object) error {
	for _, part := range access[:len(access)-1] {
		obj, err := i.evalValue(part)
		if err != nil {
			return wrapRunExc(err, part.Debug)
		}

		if err := requireValue(current, part.Debug); err != nil {
			return err
		}

		proto := current.GetPrototype()
		if proto == nil {
			return runExc("undefined property %q", language.Stringify(i.ctx, obj)).WithDebug(current.Debug())
		}

		if current.GetPrototype() == nil {
			return runExc("cannot operate on variable: no implementation for __get__()").WithDebug(obj.Debug())
		}

		getter, ok := current.GetPrototype().GetObject(i.ctx, "__get__")
		if !ok {
			return runExc("cannot operate on variable: no implementation for __get__()").WithDebug(obj.Debug())
		}

		getterFn, ok := getter.(*language.Function)
		if !ok {
			return runExc("cannot operate on variable: bad implementation for __get__()").WithDebug(obj.Debug())
		}

		value, err := callHook(i.ctx, getterFn, []language.Object{obj})
		if err != nil {
			return wrapRunExc(err, obj.Debug())
		}

		if err := requireValue(value, part.Debug); err != nil {
			return err
		}

		current = value
	}

	last := access[len(access)-1]
	obj, err := i.evalValue(last)
	if err != nil {
		return wrapRunExc(err, last.Debug)
	}

	if err := requireValue(current, last.Debug); err != nil {
		return err
	}

	proto := current.GetPrototype()
	if proto == nil {
		return runExc("undefined property %s", language.Stringify(i.ctx, obj)).WithDebug(current.Debug())
	}

	setter, ok := proto.GetObject(i.ctx, "__set__")
	if !ok {
		return runExc("cannot operate on variable: no implementation for __set__()").WithDebug(obj.Debug())
	}

	setterFn, ok := setter.(*language.Function)
	if !ok {
		return runExc("cannot operate on variable: bad implementation for __set__()").WithDebug(obj.Debug())
	}

	if _, err = callHook(i.ctx, setterFn, []language.Object{obj, value}); err != nil {
		return wrapRunExc(err, obj.Debug())
	}

	return nil
}
