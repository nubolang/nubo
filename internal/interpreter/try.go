package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
	"go.uber.org/zap"
)

func (i *Interpreter) handleTry(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.try.start", zap.Uint("id", i.ID), zap.String("file", i.currentFile))

	ret, err := i.Run(node.Body)
	if err != nil {
		zap.L().Error("interpreter.try.body.error", zap.Uint("id", i.ID), zap.Error(err))

		excp, ok := exception.Unwrap(err)
		var (
			base              string
			msg               string
			file              string
			line              int
			column, columnEnd int
			stack             []language.Object
		)

		if ok {
			base = excp.Base
			msg = excp.Message
			file = excp.Debug.File
			line = excp.Debug.Line
			column = excp.Debug.Column
			columnEnd = excp.Debug.ColumnEnd

			for _, frame := range excp.StackTrace {
				stackDict, err := n.Dict(map[any]any{
					"file":      frame.File,
					"line":      frame.Line,
					"column":    frame.Column,
					"columnEnd": frame.ColumnEnd,
				})
				if err != nil {
					zap.L().Error("interpreter.try.stack.error", zap.Uint("id", i.ID), zap.Error(err))
					return nil, wrapRunExc(err, node.Debug)
				}
				stack = append(stack, stackDict)
			}
		}

		metaData, err := n.Dict(map[any]any{
			"file":      file,
			"line":      line,
			"column":    column,
			"columnEnd": columnEnd,
		})
		if err != nil {
			zap.L().Error("interpreter.try.metadata.error", zap.Uint("id", i.ID), zap.Error(err))
			return nil, wrapRunExc(err, node.Debug)
		}

		dictErr, err := n.Dict(map[any]any{
			"base":       base,
			"message":    msg,
			"metaData":   metaData,
			"stackTrace": language.NewList(stack, language.TypeAny, node.Debug),
		}, node.Debug)
		if err != nil {
			zap.L().Error("interpreter.try.dict.error", zap.Uint("id", i.ID), zap.Error(err))
			return nil, wrapRunExc(err, node.Debug)
		}

		if err := i.Declare(node.Content, dictErr, language.TypeAny, true); err != nil {
			zap.L().Error("interpreter.try.declare.error", zap.Uint("id", i.ID), zap.Error(err))
			return nil, wrapRunExc(err, node.Debug)
		}
		zap.L().Debug("interpreter.try.catch", zap.Uint("id", i.ID), zap.String("variable", node.Content))
		return nil, nil
	}

	if err := i.Declare(node.Content, language.Nil, language.TypeAny, true); err != nil {
		zap.L().Error("interpreter.try.declare.nil", zap.Uint("id", i.ID), zap.Error(err))
		return nil, wrapRunExc(err, node.Debug)
	}
	zap.L().Debug("interpreter.try.success", zap.Uint("id", i.ID))
	return ret, nil
}
