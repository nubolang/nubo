package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/packages/iter"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

type Iterator interface {
	// Iterator is an interface that returns a Next function which
	// returns a key, value, and ok (if ok is false, both key and value are nil and the cycle ends)
	Iterator() func() (language.Object, language.Object, bool)
}

func (i *Interpreter) handleFor(node *astnode.Node) (language.Object, error) {
	kv, ok := node.Value.(*astnode.ForValue)
	if !ok {
		return nil, runExc("expected a valid for cycle").WithDebug(node.Debug)
	}

	expr, err := i.eval(node.Args[0])
	if err != nil {
		return nil, exception.From(err, node.Debug, "failed to evaluate expression")
	}

	var iterate func() (language.Object, language.Object, bool, error)
	if it, ok := i.getIterator(expr); ok {
		iterate = it
	} else {
		iterator, ok := expr.(Iterator)
		if !ok {
			return nil, runExc("expected iterator, got '%s' with value '%v'", expr.Type(), expr.Value()).WithDebug(expr.Debug())
		}

		fn := iterator.Iterator()
		iterate = func() (language.Object, language.Object, bool, error) {
			key, value, ok := fn()
			return key, value, ok, nil
		}
	}

	// Create loop scope only once
	ir := NewWithParent(i, ScopeBlock, "for")

	var keyName, valName string
	if kv.Iterator != nil {
		keyName = kv.Iterator.Value.(string)
		// declare once
		if err := ir.Declare(keyName, language.Nil, n.TAny, true); err != nil {
			return nil, wrapRunExc(err, expr.Debug())
		}
	}
	if kv.Value != nil {
		valName = kv.Value.Value.(string)
		// declare once
		if err := ir.Declare(valName, language.Nil, n.TAny, true); err != nil {
			return nil, wrapRunExc(err, expr.Debug())
		}
	}

	for {
		key, value, ok, err := iterate()
		if err != nil {
			return nil, wrapRunExc(err, expr.Debug())
		}
		if !ok {
			break
		}

		// only assign instead of redeclare
		if keyName != "" {
			if err := ir.Assign(keyName, key); err != nil {
				return nil, wrapRunExc(err, key.Debug())
			}
		}
		if valName != "" {
			if err := ir.Assign(valName, value); err != nil {
				return nil, wrapRunExc(err, value.Debug())
			}
		}

		ob, err := ir.Run(node.Body)
		if err != nil {
			return nil, wrapRunExc(err, node.Debug)
		}
		if ob != nil {
			if ob.Type().Base() == language.ObjectTypeSignal {
				switch ob.String() {
				case "break":
					break
				case "continue":
					continue
				default:
					return nil, runExc("invalid language signal: %s", ob.String()).WithDebug(ob.Debug())
				}
			} else {
				return ob, nil
			}
		}
	}

	return nil, nil
}

func (i *Interpreter) getIterator(expr language.Object) (func() (language.Object, language.Object, bool, error), bool) {
	proto := expr.GetPrototype()
	if proto == nil {
		return nil, false
	}

	it, ok := proto.GetObject(i.ctx, "__iterate__")
	if !ok {
		return nil, false
	}

	f, ok := it.(*language.Function)
	if !ok {
		return nil, false
	}

	iteratorCreator := iter.NewIter(expr.Debug())
	iterProto := iteratorCreator.GetPrototype()
	if iterProto == nil {
		return nil, false
	}

	iterator, ok := iterProto.GetObject(i.ctx, "Iterator")
	if !ok {
		return nil, false
	}

	if !language.TypeCheck(n.TTFn(iterator.Type()), f.Type()) {
		return nil, false
	}

	realIterObj, err := f.Data(language.StructAllowPrivateCtx(i.ctx), nil)
	if err != nil {
		return nil, false
	}

	realIterProto := realIterObj.GetPrototype()
	if realIterProto == nil {
		return nil, false
	}

	nextFn, ok := realIterProto.GetObject(i.ctx, "next")
	if !ok {
		return nil, false
	}

	next := nextFn.(*language.Function)

	return func() (language.Object, language.Object, bool, error) {
		current, err := next.Data(i.ctx, nil)
		if err != nil {
			return nil, nil, false, err
		}

		currentProto := current.GetPrototype()
		if currentProto == nil {
			return nil, nil, false, runExc("prototype is empty for iterable")
		}

		end, ok := currentProto.GetObject(i.ctx, "end")
		if !ok {
			return nil, nil, false, runExc("end property not found")
		}

		if end.Value().(bool) {
			return nil, nil, false, nil
		}

		key, ok := currentProto.GetObject(i.ctx, "key")
		if !ok {
			return nil, nil, false, runExc("key property not found")
		}

		value, ok := currentProto.GetObject(i.ctx, "value")
		if !ok {
			return nil, nil, false, runExc("value property not found")
		}

		return key, value, true, nil
	}, true
}
