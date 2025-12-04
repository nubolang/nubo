package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleIf(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.if.start", zap.Uint("id", i.ID), zap.String("file", i.currentFile))

	if len(node.Args) != 1 {
		err := exception.Create("invalid or malformed if statement condition").WithDebug(node.Debug).WithLevel(exception.LevelSemantic)
		zap.L().Error("interpreter.if.invalidArgs", zap.Uint("id", i.ID), zap.Error(err))
		return nil, err
	}

	condition := func() (bool, error) {
		ok, err := i.eval(node.Args[0])
		if err != nil {
			zap.L().Error("interpreter.if.condition.evalError", zap.Uint("id", i.ID), zap.Error(err))
			return false, exception.From(err, node.Args[0].Debug, "failed to evaluate condition: @err")
		}
		if ok.Type() != language.TypeBool {
			err := typeError("condition expected to be type bool, got %s with value: %s", ok.Type(), ok.Value()).WithDebug(ok.Debug())
			zap.L().Error("interpreter.if.condition.typeMismatch", zap.Uint("id", i.ID), zap.Error(err))
			return false, err
		}
		result := ok.Value().(bool)
		zap.L().Debug("interpreter.if.condition.result", zap.Uint("id", i.ID), zap.Bool("result", result))
		return result, nil
	}

	ok, err := condition()
	if err != nil {
		zap.L().Error("interpreter.if.condition.error", zap.Uint("id", i.ID), zap.Error(err))
		return nil, exception.From(err, node.Debug, "failed to evaluate condition: @err")
	}

	var execNodes []*astnode.Node
	if ok {
		execNodes = node.Body
	} else {
		execNodes = node.Children
	}

	branch := "else"
	if ok {
		branch = "then"
	}
	zap.L().Debug("interpreter.if.branch", zap.Uint("id", i.ID), zap.String("branch", branch), zap.Int("nodes", len(execNodes)))

	if len(execNodes) > 0 {
		ir := NewWithParent(i, ScopeBlock)
		ob, err := ir.Run(execNodes)
		if err != nil {
			zap.L().Error("interpreter.if.body.error", zap.Uint("id", ir.ID), zap.Error(err))
			return nil, exception.From(err, node.Debug, "failed to execute statement body: @err")
		}
		if ob != nil {
			zap.L().Debug("interpreter.if.return", zap.Uint("id", i.ID), zap.String("returnType", logObjectType(ob)))
			return ob, nil
		}
	}

	zap.L().Debug("interpreter.if.end", zap.Uint("id", i.ID))
	return nil, nil
}
