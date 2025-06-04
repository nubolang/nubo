package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"github.com/stoewer/go-strcase"
)

func (i *Interpreter) evaluateElement(node *astnode.Node) (language.Object, error) {
	elem := &language.ElementData{
		TagName:   node.Content,
		SelfClose: node.Flags.Contains("SELFCLOSING"),
	}

	for _, arg := range node.Args {
		attr := language.Attribute{
			Name: strcase.KebabCase(arg.Content),
			Kind: arg.Kind,
		}

		if arg.Kind == "DYNAMIC" && arg.Value != nil {
			valNode := arg.Value.(*astnode.Node)
			val, err := i.evaluateExpression(valNode)
			if err != nil {
				return nil, err
			}
			attr.Value = val
		} else if arg.Kind == "TEXT" && arg.Value != nil {
			attr.Value = language.NewString(arg.Value.(string), node.Debug)
		}

		elem.Args = append(elem.Args, attr)
	}

	for _, child := range node.Children {
		switch child.Type {
		case astnode.NodeTypeElement:
			val, err := i.evaluateElement(child)
			if err != nil {
				return nil, err
			}
			childElem := val.(*language.Element)
			elem.Children = append(elem.Children, language.ElementChild{
				Type:      astnode.NodeTypeElement,
				Value:     childElem,
				IsEscaped: !child.Flags.Contains("UNESCAPED"),
			})
		case astnode.NodeTypeElementRawText:
			elem.Children = append(elem.Children, language.ElementChild{
				Type:      astnode.NodeTypeElementRawText,
				Content:   child.Content,
				IsEscaped: !child.Flags.Contains("UNESCAPED"),
			})
		case astnode.NodeTypeElementDynamicText:
			val, err := i.evaluateExpression(child.Value.(*astnode.Node))
			if err != nil {
				return nil, err
			}

			elem.Children = append(elem.Children, language.ElementChild{
				Type:      astnode.NodeTypeElementRawText,
				Content:   val.String(),
				IsEscaped: !child.Flags.Contains("UNESCAPED"),
			})
		}
	}

	return language.NewElement(elem, node.Debug), nil
}
