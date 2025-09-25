package interpreter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) eval(node *astnode.Node) (language.Object, error) {
	switch node.Type {
	default:
		return i.evaluateExpression(node)
	case astnode.NodeTypeElement:
		return i.evaluateElement(node)
	case astnode.NodeTypeDict:
		return i.evalDict(node, nil, nil)
	case astnode.NodeTypeInclude:
		return i.includeValue(node)
	}
}

func (i *Interpreter) evaluateExpression(node *astnode.Node) (language.Object, error) {
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

			if child.IsReference {
				sb.WriteString(id + "()")
				obFn := func() (any, error) {
					obj, ok := i.GetObject(child.Value.(string))
					if !ok {
						return nil, undefinedVariable(child.Value.(string)).WithDebug(node.Debug)
					}

					if len(node.Body) == 1 {
						if len(node.Body[0].ArrayAccess) > 0 {
							if ob, err := i.checkGetter(obj, child); err != nil {
								return nil, exception.From(err, node.Debug, "accessing getter failed")
							} else {
								obj = ob
							}
						}

						return obj, nil
					}

					if obj.Type().Base() == language.ObjectTypeStructDefinition {
						if len(node.Body) == 1 {
							return obj, nil
						}
						return nil, cannotOperateOn("(struct) " + obj.Type().Content).WithDebug(obj.Debug())
					}

					if obj.Type().Base() == language.ObjectTypeStructInstance {
						value, ok := obj.GetPrototype().GetObject("__value__")
						if ok && language.NewFunctionType(language.TypeAny).Compare(value.Type()) {
							fn, ok := value.(*language.Function)
							if ok {
								value, err := fn.Data(nil)
								if err != nil {
									return nil, exception.From(err, obj.Debug(), "function call failed: @err")
								}
								obj = value
							}
						}
					}

					if len(child.ArrayAccess) > 0 {
						if ob, err := i.checkGetter(obj, child); err != nil {
							return nil, exception.From(err, child.Debug, "accessing getter failed")
						} else {
							obj = ob
						}
					}

					if isNotEvaluable(obj.Type().Base()) {
						return nil, cannotOperateOn(obj.Type()).WithDebug(obj.Debug())
					}

					return obj.Value(), nil
				}

				env[id] = obFn
			} else {
				sb.WriteString(id)
				env[id] = child.Value
			}
		} else if child.Type == astnode.NodeTypeOperator {
			sb.WriteString(child.Kind)
		} else if child.Type == astnode.NodeTypeFunctionCall {
			valueFn := func() (any, error) {
				value, err := i.handleFunctionCall(child)
				if err != nil {
					return nil, exception.From(err, child.Debug, "function call failed: @err")
				}

				if len(node.Body) == 1 {
					return value, nil
				}

				if isNotEvaluable(value.Type().Base()) {
					return nil, cannotOperateOn(value.Type()).WithDebug(value.Debug())
				}

				return value.Value(), nil
			}

			id := "var_" + fmt.Sprintf("%d", inx)
			inx++
			sb.WriteString(id + "()")
			env[id] = valueFn
		} else if child.Type == astnode.NodeTypeInlineFunction {
			if len(node.Body) != 1 {
				return nil, cannotOperateOn("<inline function>").WithDebug(node.Debug)
			}

			return i.createInlineFunction(child)
		} else if child.Type == astnode.NodeTypeElement {
			if len(node.Body) != 1 {
				return nil, cannotOperateOn("<element>").WithDebug(node.Debug)
			}
			ret, err := i.evaluateElement(child)
			if err != nil {
				return nil, exception.From(err, child.Debug, "element evaluation failed: @err")
			}
			return ret, nil
		} else if child.Type == astnode.NodeTypeTemplateLiteral {
			stFn := func() (string, error) {
				var st strings.Builder
				for _, ch := range child.Children {
					if ch.Type == astnode.NodeTypeRawText {
						st.WriteString(ch.Content)
					} else {
						val, err := i.eval(ch.Value.(*astnode.Node))
						if err != nil {
							return "", exception.From(err, ch.Debug, "", "template literal evaluation failed: @err")
						}
						st.WriteString(val.String())
					}
				}
				return st.String(), nil
			}
			id := "var_" + fmt.Sprintf("%d", inx)
			inx++
			sb.WriteString(id + "()")
			env[id] = stFn
		} else {
			sb.WriteString(child.Value.(string))
		}
	}

	code := sb.String()
	sb.Reset()

	program, err := expr.Compile(code, expr.Env(env))
	if err != nil {
		if exception.Is(err) {
			return nil, err
		}
		return nil, i.exprEvalHumanError(node.Body, node.Debug, err)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		if exception.Is(err) {
			return nil, err
		}
		return nil, i.exprEvalHumanError(node.Body, node.Debug, err)
	}

	return language.FromValue(output, false, node.Debug)
}

func isNotEvaluable(typ language.ObjectType) bool {
	return typ == language.ObjectTypeDict || typ == language.ObjectTypeFunction || typ == language.ObjectTypeStructInstance || typ == language.ObjectTypeList
}

func (i *Interpreter) exprEvalHumanError(children []*astnode.Node, debug *debug.Debug, isErr ...error) error {
	var humanExpr strings.Builder

	for i, child := range children {
		humanExpr.WriteString(humanNode(child))

		if i < len(children)-1 {
			humanExpr.WriteString(" ")
		}
	}

	var msgCtx string
	if len(isErr) > 0 {
		msgCtx = i.getEvalErr(isErr[0])
	}

	if msgCtx != "" {
		msgCtx = fmt.Sprintf(" (%s)", msgCtx)
	}

	excp := expressionError(humanExpr.String() + msgCtx)
	if debug != nil {
		return excp.WithDebug(debug)
	}

	return excp
}

func (i *Interpreter) getEvalErr(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	re := regexp.MustCompile(`mismatched types ([a-z0-9]+) and ([a-z0-9]+)`)
	if m := re.FindStringSubmatch(msg); len(m) == 3 {
		from, to := m[1], m[2]
		typeMap := map[string]string{
			"int64":   "int",
			"int":     "int",
			"float64": "float",
		}

		a, b := typeMap[from], typeMap[to]

		if a == "" {
			a = from
		}

		if b == "" {
			b = to
		}

		return fmt.Sprintf("type mismatch: %s and %s", a, b)
	}

	return fmt.Sprint(err)
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
		obj, err := i.eval(child)
		if err != nil {
			return nil, exception.From(err, child.Debug, "failed to evaluate list element: @err")
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
			return nil, runExc("invalid dict entry").WithDebug(pair.Debug)
		}

		keyNode, ok := pair.Value.(*astnode.Node)
		if !ok {
			return nil, runExc("invalid dict entry").WithDebug(pair.Debug)
		}

		keyObj, err := i.eval(keyNode)
		if err != nil {
			return nil, exception.From(err, keyNode.Debug, "failed to evaluate dict key: @err")
		}

		valueObj, err := i.eval(pair.Children[0])
		if err != nil {
			return nil, exception.From(err, pair.Debug, "failed to evaluate dict value: @err")
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
		return nil, typeError(err.Error()).WithDebug(node.Debug)
	}

	return dict, nil
}

func (i *Interpreter) checkGetter(obj language.Object, node *astnode.Node) (language.Object, error) {
	for _, child := range node.ArrayAccess {
		val, err := i.eval(child)
		if err != nil {
			return nil, exception.From(err, child.Debug, "failed to access getter: @err")
		}

		if obj.GetPrototype() == nil {
			return nil, cannotOperateOn(obj.Type()).WithDebug(obj.Debug())
		}

		getter, ok := obj.GetPrototype().GetObject("__get__")
		if !ok {
			return nil, cannotOperateOn(obj.Type()).WithDebug(obj.Debug())
		}

		getterFn, ok := getter.(*language.Function)
		if !ok {
			return nil, cannotOperateOn(obj.Type()).WithDebug(obj.Debug())
		}

		value, err := getterFn.Data([]language.Object{val})
		if err != nil {
			return nil, exception.From(err, child.Debug, "function call failed: @err")
		}

		obj = value
	}

	if obj == nil {
		return nil, cannotOperateOn(obj.Type()).WithDebug(obj.Debug())
	}

	return obj, nil
}
