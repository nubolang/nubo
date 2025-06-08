package formatter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
)

func (f *Formatter) formatNode(node *astnode.Node) {
	switch node.Type {
	case astnode.NodeTypeVariableDecl:
		f.formatVariableDeclaration(node)
	}
}

func (f *Formatter) formatVariableDeclaration(node *astnode.Node) {
	if node.Kind == "LET" {
		f.sb.WriteString("let ")
	} else {
		f.sb.WriteString("const ")
	}
	f.sb.WriteString(node.Content)

	if node.Value != nil {
		f.sb.WriteString(" = ")
		f.formatExpression(node.Value.(*astnode.Node))
	}
	f.sb.WriteRune('\n')
}
