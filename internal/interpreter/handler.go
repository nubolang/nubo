package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleNode(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.handler.node", zap.Uint("id", i.ID), zap.String("nodeType", string(node.Type)), zap.String("content", node.Content))

	switch node.Type {
	case astnode.NodeTypeImport:
		return nil, i.handleImport(node)
	case astnode.NodeTypeInclude:
		return nil, i.handleInclude(node)
	case astnode.NodeTypeVariableDecl:
		return nil, i.handleVariableDecl(node)
	case astnode.NodeTypeAssign:
		return nil, i.handleAssignment(node)
	case astnode.NodeTypeFunctionCall:
		_, err := i.handleFunctionCall(node)
		return nil, err
	case astnode.NodeTypeEvent:
		return i.handleEventDecl(node)
	case astnode.NodeTypeSubscribe:
		return i.handleSubscribe(node)
	case astnode.NodeTypePublish:
		return i.handlePublish(node)
	case astnode.NodeTypeFunction:
		return i.handleFunctionDecl(node)
	case astnode.NodeTypeWhile:
		return i.handleWhile(node)
	case astnode.NodeTypeIncrement:
		return nil, i.handleIncrement(node)
	case astnode.NodeTypeDecrement:
		return nil, i.handleDecrement(node)
	case astnode.NodeTypeIf:
		return i.handleIf(node)
	case astnode.NodeTypeReturn:
		return i.handleReturn(node)
	case astnode.NodeTypeFor:
		return i.handleFor(node)
	case astnode.NodeTypeStruct:
		return nil, i.handleStruct(node)
	case astnode.NodeTypeImpl:
		return nil, i.handleImpl(node)
	case astnode.NodeTypeTry:
		return i.handleTry(node)
	case astnode.NodeTypeDefer:
		i.deferred = append(i.deferred, node.Children)
		zap.L().Debug("interpreter.handler.defer", zap.Uint("id", i.ID), zap.Int("deferredCount", len(i.deferred)))
		return nil, nil
	case astnode.NodeTypeSpawn:
		go i.handleSpawn(node)
		zap.L().Debug("interpreter.handler.spawn", zap.Uint("id", i.ID))
		return nil, nil
	default:
		if node.Type == astnode.NodeTypeSignal {
			if i.isChildOf(ScopeBlock, "for") || i.isChildOf(ScopeBlock, "while") {
				return language.NewSignal(node.Content, node.Debug), nil
			} else {
				err := runExc("%s(%s) can only be used within a for or while loop", node.Type, node.Content).WithDebug(node.Debug)
				zap.L().Error("interpreter.handler.signal.invalid", zap.Uint("id", i.ID), zap.String("nodeType", string(node.Type)), zap.String("content", node.Content), zap.Error(err))
				return nil, err
			}
		}
		err := runExc("unknown node %s", node.Type).WithDebug(node.Debug)
		zap.L().Error("interpreter.handler.unknownNode", zap.Uint("id", i.ID), zap.String("nodeType", string(node.Type)), zap.Error(err))
		return nil, err
	}
}
