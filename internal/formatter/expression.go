package formatter

import (
	"fmt"
	"strconv"

	"github.com/nubolang/nubo/internal/ast/astnode"
)

func (f *Formatter) formatExpression(node *astnode.Node) {
	max := len(node.Body) - 1
	for i, child := range node.Body {
		switch child.Type {
		case astnode.NodeTypeValue:
			switch child.Kind {
			case "STRING":
				f.sb.WriteString(strconv.Quote(child.Value.(string)))
			case "INTEGER", "BOOLEAN", "FLOAT", "IDENTIFIER":
				f.sb.WriteString(fmt.Sprint(child.Value))
			}
		case astnode.NodeTypeOperator:
			f.sb.WriteString(child.Kind)
		case astnode.NodeTypeFunctionCall:
			f.sb.WriteString(child.Content)
			f.sb.WriteRune('(')
			argsLen := len(child.Args) - 1
			for i, arg := range child.Args {
				f.formatExpression(arg)
				if i < argsLen {
					f.sb.WriteString(", ")
				}
			}
			f.sb.WriteRune(')')
		case astnode.NodeTypeInlineFunction:
			f.sb.WriteString("fn(")
		}
		if i < max {
			f.sb.WriteRune(' ')
		}
	}
}
