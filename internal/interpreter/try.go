package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleTry(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.try.start", zap.Uint("id", i.ID), zap.String("file", i.currentFile))

	ret, err := i.Run(node.Body)
	if err != nil {
		zap.L().Error("interpreter.try.body.error", zap.Uint("id", i.ID), zap.Error(err))

		caught, buildErr := language.ErrorFrom(err, node.Debug)
		if buildErr != nil {
			return nil, wrapRunExc(buildErr, node.Debug)
		}

		if err := i.Declare(node.Content, caught, language.Nullable(language.ErrorStruct().Type()), true); err != nil {
			return nil, wrapRunExc(err, node.Debug)
		}

		return nil, nil
	}

	if err := i.Declare(node.Content, language.Nil, language.Nullable(language.ErrorStruct().Type()), true); err != nil {
		return nil, wrapRunExc(err, node.Debug)
	}

	return ret, nil
}
