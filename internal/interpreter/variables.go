package interpreter

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/debug"
	"github.com/nubogo/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleVariableDecl(node *astnode.Node) error {
	var (
		variableName string = node.Content
		mutable             = node.Kind != "CONST"
		value        language.Object
		err          error
	)

	if node.Flags.Contains("NODEVALUE") {
		node = node.Value.(*astnode.Node)
	}

	if node.Type == astnode.NodeTypeExpression {
		value, err = i.fromExpression(node)
	} else if node.Type == astnode.NodeTypeElement {
		value, err = i.fromElement(node)
	}

	if err != nil {
		return err
	}

	zap.L().Info("Variable Declaration", zap.String("variableName", variableName), zap.Any("value", value), zap.Bool("mutable", mutable))

	return i.BindObject(variableName, value, mutable)
}

func (i *Interpreter) handleAssignment(node *astnode.Node) error {
	var (
		variableName string = node.Content
		value        language.Object
		err          error
	)

	if node.Flags.Contains("NODEVALUE") {
		node = node.Value.(*astnode.Node)
	}

	if node.Type == astnode.NodeTypeExpression {
		value, err = i.fromExpression(node)
	} else if node.Type == astnode.NodeTypeElement {
		value, err = i.fromElement(node)
	}

	if err != nil {
		return err
	}

	zap.L().Info("Variable Assignment", zap.String("variableName", variableName), zap.Any("value", value))

	return i.BindObject(variableName, value, false)
}

func (i *Interpreter) fromExpression(node *astnode.Node) (language.Object, error) {
	var (
		sb  strings.Builder
		env = make(map[string]any)
		inx = 0
	)

	for _, child := range node.Body {
		if child.Type == astnode.NodeTypeValue {
			id := "var_" + fmt.Sprintf("%d", inx)
			inx++
			sb.WriteString(id)

			if child.IsReference {
				obj, ok := i.GetObject(child.Value.(string))
				if !ok {
					return nil, i.exprEvalHumanError(node.Body, node.Debug)
				}
				env[id] = obj.Value()
			} else {
				env[id] = child.Value
			}
		} else if child.Type == astnode.NodeTypeOperator {
			sb.WriteString(child.Kind)
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

func (i *Interpreter) exprEvalHumanError(children []*astnode.Node, debug *debug.Debug) error {
	var humanExpr strings.Builder

	for i, child := range children {
		humanExpr.WriteString(fmt.Sprint(child.Value))
		if i < len(children)-1 {
			humanExpr.WriteString(" ")
		}
	}

	if debug != nil {
		return newErr(ErrExpression, fmt.Sprintf("Failed to evaluate expression: %s", humanExpr.String()), debug)
	}

	return newErr(ErrExpression, fmt.Sprintf("Failed to evaluate expression: %s", humanExpr.String()))
}

func (i *Interpreter) fromElement(node *astnode.Node) (language.Object, error) {
	var (
		sb strings.Builder
	)

	sb.WriteRune('<')
	sb.WriteString(node.Content)
	for _, arg := range node.Args {
		if arg.Kind == "DYNAMIC" {
			sb.WriteRune(' ')
			sb.WriteString(arg.Content)
			sb.WriteRune('=')
			nodeValue := arg.Value.(*astnode.Node)
			value, err := i.fromExpression(nodeValue)
			if err != nil {
				return nil, err
			}
			sb.WriteString(strconv.Quote(html.EscapeString(value.String())))
		} else if arg.Kind == "TEXT" {
			sb.WriteRune(' ')
			sb.WriteString(arg.Content)
			sb.WriteRune('=')
			sb.WriteString(strconv.Quote(html.EscapeString(arg.Value.(string))))
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
			childValue, err := i.fromElement(child)
			if err != nil {
				return nil, err
			}
			sb.WriteString(childValue.String())
		case astnode.NodeTypeElementRawText:
			sb.WriteString(html.EscapeString(child.Content))
		case astnode.NodeTypeElementDynamicText:
			childValue, err := i.fromExpression(child.Value.(*astnode.Node))
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
