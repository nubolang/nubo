package interpreter

import (
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/language"
)

func (i *Interpreter) handleNode(node *astnode.Node) (language.Object, error) {
	switch node.Type {
	case astnode.NodeTypeImport:
		return nil, i.handleImport(node)
	case astnode.NodeTypeVariableDecl:
		return nil, i.handleVariableDecl(node)
	case astnode.NodeTypeAssign:
		return nil, i.handleAssignment(node)
	case astnode.NodeTypeFunctionCall:
		return i.handleFunctionCall(node)
	case astnode.NodeTypeEvent:
		return i.handleEventDecl(node)
	case astnode.NodeTypeSubscribe:
		return i.handleSubscribe(node)
	case astnode.NodeTypePublish:
		return i.handlePublish(node)
	case astnode.NodeTypeFunction:
		return i.handleFunctionDecl(node)
	default:
		return nil, newErr(ErrUnknownNode, fmt.Sprintf("%s", node.Type), node.Debug)
	}
}
