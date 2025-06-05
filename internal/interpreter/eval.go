package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
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

	if node.Type == astnode.NodeTypeList {
		var (
			typ  language.ObjectComplexType
			list = make([]language.Object, len(node.Children))
		)

		for j, child := range node.Children {
			obj, err := i.evaluateExpression(child)
			if err != nil {
				return nil, err
			}
			list[j] = obj

			if typ == nil {
				typ = obj.Type()
			} else if typ != obj.Type() {
				typ = language.TypeAny
			}
		}

		if typ == nil {
			typ = language.TypeAny
		}

		return language.NewList(list, typ, node.Debug), nil
	}

	if node.Body == nil {
		return language.Nil, nil
	}

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

				if len(node.Body) == 1 {
					return obj, nil
				}

				if isNotEvaluable(obj.Type().Base()) {
					return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
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

			if len(node.Body) == 1 {
				return value, nil
			}

			if isNotEvaluable(value.Type().Base()) {
				return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", value.Type()), value.Debug())
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

	return language.FromValue(output, node.Debug)
}

func isNotEvaluable(typ language.ObjectType) bool {
	return typ == language.TypeDict || typ == language.TypeFunction || typ == language.TypeStructInstance || typ == language.TypeList
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
