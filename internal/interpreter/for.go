package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/packages/iter"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
	"go.uber.org/zap"
)

type Iterator interface {
	// Iterator is an interface that returns a Next function which
	// returns a key, value, and ok (if ok is false, both key and value are nil and the cycle ends)
	Iterator() func() (language.Object, language.Object, bool)
}

func (i *Interpreter) handleFor(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.for.start", zap.Uint("id", i.ID), zap.String("file", i.currentFile))

	kv, ok := node.Value.(*astnode.ForValue)
	if !ok {
		err := runExc("expected a valid for cycle").WithDebug(node.Debug)
		zap.L().Error("interpreter.for.invalidNode", zap.Uint("id", i.ID), zap.Error(err))
		return nil, err
	}

	expr, err := i.eval(node.Args[0])
	if err != nil {
		zap.L().Error("interpreter.for.expression.error", zap.Uint("id", i.ID), zap.Error(err))
		return nil, exception.From(err, node.Debug, "failed to evaluate expression")
	}

	var iterate func() (language.Object, language.Object, bool, error)
	if it, ok := i.getIterator(expr); ok {
		zap.L().Debug("interpreter.for.iterator.native", zap.Uint("id", i.ID), zap.String("exprType", logObjectType(expr)))
		iterate = it
	} else {
		iterator, ok := expr.(Iterator)
		if !ok {
			err := runExc("expected iterator, got '%s' with value '%v'", expr.Type(), expr.Value()).WithDebug(expr.Debug())
			zap.L().Error("interpreter.for.iterator.invalid", zap.Uint("id", i.ID), zap.Error(err))
			return nil, err
		}

		fn := iterator.Iterator()
		iterate = func() (language.Object, language.Object, bool, error) {
			key, value, ok := fn()
			return key, value, ok, nil
		}
		zap.L().Debug("interpreter.for.iterator.custom", zap.Uint("id", i.ID), zap.String("exprType", logObjectType(expr)))
	}

	// Create loop scope only once
	ir := NewWithParent(i, ScopeBlock, "for")

	var keyName, valName string
	if kv.Iterator != nil {
		keyName = kv.Iterator.Value.(string)
		// declare once
		if err := ir.Declare(keyName, language.Nil, n.TAny, true); err != nil {
			zap.L().Error("interpreter.for.declare.key", zap.Uint("id", ir.ID), zap.String("name", keyName), zap.Error(err))
			return nil, wrapRunExc(err, expr.Debug())
		}
	}
	if kv.Value != nil {
		valName = kv.Value.Value.(string)
		// declare once
		if err := ir.Declare(valName, language.Nil, n.TAny, true); err != nil {
			zap.L().Error("interpreter.for.declare.value", zap.Uint("id", ir.ID), zap.String("name", valName), zap.Error(err))
			return nil, wrapRunExc(err, expr.Debug())
		}
	}

	iterations := 0
	for {
		key, value, ok, err := iterate()
		if err != nil {
			zap.L().Error("interpreter.for.iterate.error", zap.Uint("id", i.ID), zap.Error(err))
			return nil, wrapRunExc(err, expr.Debug())
		}
		if !ok {
			zap.L().Debug("interpreter.for.iterate.complete", zap.Uint("id", i.ID), zap.Int("iterations", iterations))
			break
		}

		zap.L().Debug("interpreter.for.iteration", zap.Uint("id", i.ID), zap.Int("iteration", iterations))

		// only assign instead of redeclare
		if keyName != "" {
			if err := ir.Assign(keyName, key); err != nil {
				zap.L().Error("interpreter.for.assign.key", zap.Uint("id", ir.ID), zap.String("name", keyName), zap.Error(err))
				return nil, wrapRunExc(err, key.Debug())
			}
		}
		if valName != "" {
			if err := ir.Assign(valName, value); err != nil {
				zap.L().Error("interpreter.for.assign.value", zap.Uint("id", ir.ID), zap.String("name", valName), zap.Error(err))
				return nil, wrapRunExc(err, value.Debug())
			}
		}

		ob, err := ir.Run(node.Body)
		if err != nil {
			zap.L().Error("interpreter.for.body.error", zap.Uint("id", ir.ID), zap.Error(err))
			return nil, wrapRunExc(err, node.Debug)
		}
		if ob != nil {
			if ob.Type().Base() == language.ObjectTypeSignal {
				switch ob.String() {
				case "break":
					zap.L().Debug("interpreter.for.break", zap.Uint("id", i.ID), zap.Int("iteration", iterations))
					break
				case "continue":
					zap.L().Debug("interpreter.for.continue", zap.Uint("id", i.ID), zap.Int("iteration", iterations))
					continue
				default:
					err := runExc("invalid language signal: %s", ob.String()).WithDebug(ob.Debug())
					zap.L().Error("interpreter.for.signal.invalid", zap.Uint("id", i.ID), zap.Error(err))
					return nil, err
				}
			} else {
				zap.L().Debug("interpreter.for.return", zap.Uint("id", i.ID), zap.Int("iteration", iterations), zap.String("returnType", logObjectType(ob)))
				return ob, nil
			}
		}
		iterations++
	}

	zap.L().Debug("interpreter.for.end", zap.Uint("id", i.ID), zap.Int("iterations", iterations))
	return nil, nil
}

func (i *Interpreter) getIterator(expr language.Object) (func() (language.Object, language.Object, bool, error), bool) {
	zap.L().Debug("interpreter.for.iterator.lookup", zap.Uint("id", i.ID), zap.String("exprType", logObjectType(expr)))

	proto := expr.GetPrototype()
	if proto == nil {
		zap.L().Debug("interpreter.for.iterator.missingPrototype", zap.Uint("id", i.ID))
		return nil, false
	}

	it, ok := proto.GetObject(i.ctx, "__iterate__")
	if !ok {
		zap.L().Debug("interpreter.for.iterator.methodNotFound", zap.Uint("id", i.ID))
		return nil, false
	}

	f, ok := it.(*language.Function)
	if !ok {
		zap.L().Debug("interpreter.for.iterator.invalidFn", zap.Uint("id", i.ID))
		return nil, false
	}

	iteratorCreator := iter.NewIter(expr.Debug())
	iterProto := iteratorCreator.GetPrototype()
	if iterProto == nil {
		zap.L().Debug("interpreter.for.iterator.creatorProtoMissing", zap.Uint("id", i.ID))
		return nil, false
	}

	iterator, ok := iterProto.GetObject(i.ctx, "Iterator")
	if !ok {
		zap.L().Debug("interpreter.for.iterator.creatorMissing", zap.Uint("id", i.ID))
		return nil, false
	}

	if !language.TypeCheck(n.TTFn(iterator.Type()), f.Type()) {
		zap.L().Debug("interpreter.for.iterator.typeMismatch", zap.Uint("id", i.ID))
		return nil, false
	}

	realIterObj, err := f.Data(language.StructAllowPrivateCtx(i.ctx), nil)
	if err != nil {
		return nil, false
	}

	realIterProto := realIterObj.GetPrototype()
	if realIterProto == nil {
		zap.L().Debug("interpreter.for.iterator.realProtoMissing", zap.Uint("id", i.ID))
		return nil, false
	}

	nextFn, ok := realIterProto.GetObject(i.ctx, "next")
	if !ok {
		zap.L().Debug("interpreter.for.iterator.nextMissing", zap.Uint("id", i.ID))
		return nil, false
	}

	next := nextFn.(*language.Function)

	zap.L().Debug("interpreter.for.iterator.ready", zap.Uint("id", i.ID))
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
