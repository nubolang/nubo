package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
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
	default:
		return nil, newErr(ErrUnknownNode, fmt.Sprintf("%s", node.Type), node.Debug)
	}
}
