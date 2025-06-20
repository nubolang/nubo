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
		return i.evalList(node, nil)
	}

	if node.Type == astnode.NodeTypeDict {
		return i.evalDict(node, nil, nil)
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
					if len(node.Body[0].Children) > 0 {
						if ob, err := i.checkGetter(obj, child); err != nil {
							return nil, err
						} else {
							obj = ob
						}
					}

					return obj, nil
				}

				if isNotEvaluable(obj.Type().Base()) {
					return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
				}

				if len(child.Children) > 0 {
					if ob, err := i.checkGetter(obj, child); err != nil {
						return nil, err
					} else {
						obj = ob
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
		} else if child.Type == astnode.NodeTypeInlineFunction {
			if len(node.Body) != 1 {
				return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on inline function"), child.Debug)
			}

			return i.createInlineFunction(child)
		} else if child.Type == astnode.NodeTypeElement {
			if len(node.Body) != 1 {
				return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on element"), child.Debug)
			}
			return i.evaluateElement(child)
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
	return typ == language.ObjectTypeDict || typ == language.ObjectTypeFunction || typ == language.ObjectTypeStructInstance || typ == language.ObjectTypeList
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

func (i *Interpreter) evalList(node *astnode.Node, typ *language.Type) (language.Object, error) {
	var (
		baseTyp *language.Type
		list    = make([]language.Object, len(node.Children))
	)

	for j, child := range node.Children {
		obj, err := i.evaluateExpression(child)
		if err != nil {
			return nil, err
		}
		list[j] = obj

		if baseTyp == nil {
			baseTyp = obj.Type()
		} else if !baseTyp.Compare(obj.Type()) {
			baseTyp = language.TypeAny
		}
	}

	if baseTyp != nil {
		typ = baseTyp
	}

	return language.NewList(list, typ, node.Debug), nil
}

func (i *Interpreter) evalDict(node *astnode.Node, keyType, valueType *language.Type) (language.Object, error) {
	var (
		keys              []language.Object
		values            []language.Object
		inferredKeyType   *language.Type
		inferredValueType *language.Type
	)

	for _, pair := range node.Children {
		if len(pair.Children) != 1 {
			return nil, newErr(ErrInvalid, "Invalid dict entry", pair.Debug)
		}

		keyNode, ok := pair.Value.(*astnode.Node)
		if !ok {
			return nil, newErr(ErrInvalid, "Invalid dict entry", pair.Debug)
		}

		keyObj, err := i.evaluateExpression(keyNode)
		if err != nil {
			return nil, err
		}

		valueObj, err := i.evaluateExpression(pair.Children[0])
		if err != nil {
			return nil, err
		}

		keys = append(keys, keyObj)
		values = append(values, valueObj)

		if inferredKeyType == nil {
			inferredKeyType = keyObj.Type()
		} else if !inferredKeyType.Compare(keyObj.Type()) {
			inferredKeyType = language.TypeAny
		}

		if inferredValueType == nil {
			inferredValueType = valueObj.Type()
		} else if !inferredValueType.Compare(valueObj.Type()) {
			inferredValueType = language.TypeAny
		}
	}

	if keyType == nil {
		keyType = inferredKeyType
	}

	if valueType == nil {
		valueType = inferredValueType
	}

	if keyType == nil {
		keyType = language.TypeAny
	}

	if valueType == nil {
		valueType = language.TypeAny
	}

	dict, err := language.NewDict(keys, values, keyType, valueType, node.Debug)
	if err != nil {
		return nil, newErr(ErrTypeMismatch, err.Error(), node.Debug)
	}

	return dict, nil
}

func (i *Interpreter) checkGetter(obj language.Object, node *astnode.Node) (language.Object, error) {
	for _, child := range node.Children {
		val, err := i.evaluateExpression(child)
		if err != nil {
			return nil, err
		}

		if obj.GetPrototype() == nil {
			return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
		}

		getter, ok := obj.GetPrototype().GetObject("get")
		if !ok {
			return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
		}

		getterFn, ok := getter.(*language.Function)
		if !ok {
			return nil, newErr(ErrUnsupported, fmt.Sprintf("cannot operate on type %s", obj.Type()), obj.Debug())
		}

		value, err := getterFn.Data([]language.Object{val})
		if err != nil {
			return nil, err
		}

		obj = value
	}

	if obj == nil {
		return nil, newErr(ErrTypeMismatch, fmt.Sprintf("cannot operate on type %s", node.Type), node.Debug)
	}

	return obj, nil
}
