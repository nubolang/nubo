package interpreter

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/stoewer/go-strcase"
)

func (i *Interpreter) evaluateExpression(node *astnode.Node) (language.Object, error) {
	if node.Type == astnode.NodeTypeElement {
		return i.evaluateElement(node)
	}

	var (
		sb  strings.Builder
		env = make(map[string]any)
		inx = 0
	)

	for _, child := range node.Body {
		if child.Type == astnode.NodeTypeValue || child.Type == astnode.NodeTypeFunctionArgument {
			id := "var_" + fmt.Sprintf("%d", inx)
			inx++
			sb.WriteString(id)

			if child.IsReference {
				obj, ok := i.GetObject(child.Value.(string))
				if !ok {
					return nil, newErr(ErrUndefinedVariable, child.Value.(string), node.Debug)
				}
				if obj.Type() == language.TypeFunction || obj.Type() == language.TypeStructInstance {
					if len(node.Body) == 1 {
						return obj, nil
					} else {
						return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
					}
				}

				env[id] = obj.Value()
			} else {
				env[id] = child.Value
			}
		} else if child.Type == astnode.NodeTypeOperator {
			sb.WriteString(child.Kind)
		} else if child.Type == astnode.NodeTypeFunctionCall {
			value, err := i.handleFunctionCall(child)
			if err != nil {
				return nil, err
			}
			id := "var_" + fmt.Sprintf("%d", inx)
			inx++
			sb.WriteString(id)
			env[id] = value.Value()
		} else {
			sb.WriteString(child.Value.(string))
		}
	}

	code := sb.String()
	sb.Reset()

	program, err := expr.Compile(code, expr.Env(env))
	if err != nil {
		return nil, i.exprEvalHumanError(node.Body, node.Debug)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return nil, i.exprEvalHumanError(node.Body, node.Debug)
	}

	return language.FromValue(output)
}

func (i *Interpreter) evaluateElement(node *astnode.Node) (language.Object, error) {
	var (
		sb strings.Builder
	)

	sb.WriteRune('<')
	sb.WriteString(node.Content)
	for _, arg := range node.Args {
		sb.WriteRune(' ')
		sb.WriteString(strcase.KebabCase(arg.Content))

		if arg.Kind == "DYNAMIC" {
			sb.WriteRune('=')
			var valueString string

			if arg.Value != nil {
				nodeValue := arg.Value.(*astnode.Node)
				value, err := i.evaluateExpression(nodeValue)
				if err != nil {
					return nil, err
				}
				valueString = value.String()
			}

			sb.WriteString(strconv.Quote(html.EscapeString(valueString)))
		} else if arg.Kind == "TEXT" {
			sb.WriteRune('=')

			var valueString string

			if arg.Value != nil {
				valueString = arg.Value.(string)
			}

			sb.WriteString(strconv.Quote(html.EscapeString(valueString)))
		}
	}

	if node.Flags.Contains("SELFCLOSING") {
		if len(node.Args) > 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString("/>")
		return language.FromValue(sb.String())
	}

	sb.WriteRune('>')
	for _, child := range node.Children {
		switch child.Type {
		case astnode.NodeTypeElement:
			childValue, err := i.evaluateElement(child)
			if err != nil {
				return nil, err
			}
			sb.WriteString(childValue.String())
		case astnode.NodeTypeElementRawText:
			sb.WriteString(html.EscapeString(child.Content))
		case astnode.NodeTypeElementDynamicText:
			childValue, err := i.evaluateExpression(child.Value.(*astnode.Node))
			if err != nil {
				return nil, err
			}
			sb.WriteString(html.EscapeString(childValue.String()))
		}
	}

	sb.WriteString("</")
	sb.WriteString(node.Content)
	sb.WriteString(">")

	return language.NewString(sb.String(), node.Debug), nil
}

func (i *Interpreter) exprEvalHumanError(children []*astnode.Node, debug *debug.Debug) error {
	var humanExpr strings.Builder

	for i, child := range children {
		humanExpr.WriteString(humanNode(child))

		if i < len(children)-1 {
			humanExpr.WriteString(" ")
		}
	}

	if debug != nil {
		return newErr(ErrExpression, fmt.Sprintf("Failed to evaluate expression: %s", humanExpr.String()), debug)
	}

	return newErr(ErrExpression, fmt.Sprintf("Failed to evaluate expression: %s", humanExpr.String()))
}

func humanNode(node *astnode.Node) string {
	var sb strings.Builder

	switch node.Type {
	case astnode.NodeTypeValue:
		if node.Kind == "STRING" {
			sb.WriteString(strconv.Quote(fmt.Sprint(node.Value)))
		} else {
			sb.WriteString(fmt.Sprint(node.Value))
		}
	case astnode.NodeTypeFunctionCall:
		sb.WriteString(fmt.Sprint(node.Content))
		sb.WriteString("(")
		for i, arg := range node.Args {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(humanNode(arg))
		}
		sb.WriteString(")")
	case astnode.NodeTypeExpression:
		for _, child := range node.Body {
			sb.WriteString(humanNode(child))
		}
	case astnode.NodeTypeOperator:
		sb.WriteString(node.Kind)
	}

	return sb.String()
}
