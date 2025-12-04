package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleWhile(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.while.start", zap.Uint("id", i.ID), zap.String("file", i.currentFile))

	if len(node.Args) != 1 {
		err := runExc("expected a valid while statement").WithDebug(node.Debug)
		zap.L().Error("interpreter.while.invalidArgs", zap.Uint("id", i.ID), zap.Error(err))
		return nil, err
	}

	condition := func() (bool, error) {
		ok, err := i.eval(node.Args[0])
		if err != nil {
			zap.L().Error("interpreter.while.condition.evalError", zap.Uint("id", i.ID), zap.Error(err))
			return false, wrapRunExc(err, node.Debug)
		}
		if ok.Type() != language.TypeBool {
			err := valueExc("expected bool, got %s with value %s", ok.Type(), ok.Value()).WithDebug(ok.Debug())
			zap.L().Error("interpreter.while.condition.typeMismatch", zap.Uint("id", i.ID), zap.Error(err))
			return false, err
		}
		result := ok.Value().(bool)
		zap.L().Debug("interpreter.while.condition.result", zap.Uint("id", i.ID), zap.Bool("result", result))
		return result, nil
	}

	iterations := 0
	for {
		ok, err := condition()
		if err != nil {
			zap.L().Error("interpreter.while.condition.error", zap.Uint("id", i.ID), zap.Error(err))
			return nil, wrapRunExc(err, node.Debug)
		}

		if !ok {
			zap.L().Debug("interpreter.while.exit", zap.Uint("id", i.ID), zap.Int("iterations", iterations))
			break
		}

		zap.L().Debug("interpreter.while.iteration", zap.Uint("id", i.ID), zap.Int("iteration", iterations))

		ir := NewWithParent(i, ScopeBlock, "while")
		ob, err := ir.Run(node.Body)
		if err != nil {
			zap.L().Error("interpreter.while.body.error", zap.Uint("id", ir.ID), zap.Error(err))
			return nil, wrapRunExc(err, node.Debug)
		}
		if ob != nil {
			if ob.Type().Base() == language.ObjectTypeSignal {
				if ob.String() == "break" {
					zap.L().Debug("interpreter.while.break", zap.Uint("id", i.ID), zap.Int("iteration", iterations))
					break
				}
				if ob.String() == "continue" {
					zap.L().Debug("interpreter.while.continue", zap.Uint("id", i.ID), zap.Int("iteration", iterations))
					continue
				}
				err := runExc("invalid language signal: %q", ob.String()).WithDebug(ob.Debug())
				zap.L().Error("interpreter.while.signal.invalid", zap.Uint("id", i.ID), zap.Error(err))
				return nil, err
			} else {
				zap.L().Debug("interpreter.while.return", zap.Uint("id", i.ID), zap.Int("iteration", iterations), zap.String("returnType", logObjectType(ob)))
				return ob, nil
			}
		}
		iterations++
	}

	zap.L().Debug("interpreter.while.end", zap.Uint("id", i.ID), zap.Int("iterations", iterations))
	return nil, nil
}
