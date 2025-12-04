package interpreter

import (
	"html"
	"strings"
	"unicode"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/packages/component"
	"github.com/nubolang/nubo/language"
	"github.com/stoewer/go-strcase"
	"go.uber.org/zap"
)

func endsWithCapitalTag(tagName string) bool {
	parts := strings.Split(tagName, ".")
	last := parts[len(parts)-1]
	if last == "" {
		return false
	}
	return unicode.IsUpper(rune(last[0]))
}

func (i *Interpreter) evaluateElement(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.element.start", zap.Uint("id", i.ID), zap.String("tagName", node.Content), zap.Bool("selfClose", node.Flags.Contains("SELFCLOSING")))

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
			val, err := i.eval(valNode)
			if err != nil {
				zap.L().Error("interpreter.element.attr.dynamic.error", zap.Uint("id", i.ID), zap.String("name", attr.Name), zap.Error(err))
				return nil, exception.From(err, arg.Debug, "failed to evaluate dynamic area")
			}
			attr.Value = val
			zap.L().Debug("interpreter.element.attr.dynamic", zap.Uint("id", i.ID), zap.String("name", attr.Name))
		} else if arg.Kind == "TEXT" && arg.Value != nil {
			attr.Value = language.NewString(arg.Value.(string), node.Debug)
			zap.L().Debug("interpreter.element.attr.text", zap.Uint("id", i.ID), zap.String("name", attr.Name), zap.String("value", arg.Value.(string)))
		}

		elem.Args = append(elem.Args, attr)
		zap.L().Debug("interpreter.element.attr.append", zap.Uint("id", i.ID), zap.String("name", attr.Name))
	}

	for _, child := range node.Children {
		switch child.Type {
		case astnode.NodeTypeElement:
			val, err := i.evaluateElement(child)
			if err != nil {
				zap.L().Error("interpreter.element.child.element.error", zap.Uint("id", i.ID), zap.String("tagName", child.Content), zap.Error(err))
				return nil, exception.From(err, child.Debug, "failed to evaluate element")
			}
			childElem := val.(*language.Element)
			elem.Children = append(elem.Children, language.ElementChild{
				Type:      astnode.NodeTypeElement,
				Value:     childElem,
				IsEscaped: !child.Flags.Contains("UNESCAPED"),
			})
			zap.L().Debug("interpreter.element.child.element.append", zap.Uint("id", i.ID), zap.String("tagName", child.Content))
		case astnode.NodeTypeElementRawText:
			elem.Children = append(elem.Children, language.ElementChild{
				Type:      astnode.NodeTypeElementRawText,
				Content:   child.Content,
				IsEscaped: !child.Flags.Contains("UNESCAPED"),
			})
			zap.L().Debug("interpreter.element.child.raw.append", zap.Uint("id", i.ID), zap.String("content", child.Content))
		case astnode.NodeTypeElementDynamicText:
			val, err := i.eval(child.Value.(*astnode.Node))
			if err != nil {
				zap.L().Error("interpreter.element.child.dynamic.error", zap.Uint("id", i.ID), zap.Error(err))
				return nil, exception.From(err, child.Debug, "failed to evaluate dynamic text")
			}

			elem.Children = append(elem.Children, language.ElementChild{
				Type:      astnode.NodeTypeElementRawText,
				Content:   val.String(),
				IsEscaped: !child.Flags.Contains("UNESCAPED"),
			})
			zap.L().Debug("interpreter.element.child.dynamic.append", zap.Uint("id", i.ID), zap.String("content", logObjectString(val)))
		}
	}

	tagName := node.Content
	if !endsWithCapitalTag(tagName) {
		zap.L().Debug("interpreter.element.return.raw", zap.Uint("id", i.ID), zap.String("tagName", tagName))
		return language.NewElement(elem, node.Debug), nil
	}

	ob, ok := i.GetObject(tagName)
	if !ok {
		zap.L().Debug("interpreter.element.return.notFunction", zap.Uint("id", i.ID), zap.String("tagName", tagName))
		return language.NewElement(elem, node.Debug), nil
	}

	fn, ok := ob.(*language.Function)
	if !ok {
		zap.L().Debug("interpreter.element.return.nonCallable", zap.Uint("id", i.ID), zap.String("tagName", tagName))
		return language.NewElement(elem, node.Debug), nil
	}

	cctxRaw, _ := component.NewComponent(node.Debug).GetPrototype().GetObject(i.ctx, "Context")
	cctx := cctxRaw.(*language.Struct)
	fnType := language.NewFunctionType(language.TypeHtml, cctx.Type())

	if !fnType.Compare(fn.Type()) {
		zap.L().Error("interpreter.element.fn.typeMismatch", zap.Uint("id", i.ID), zap.String("expected", fnType.String()), zap.String("got", logObjectType(fn)))
		return nil, exception.Create("invalid element, expected type %s, got %s", fnType, fn.Type()).WithLevel(exception.LevelType).WithDebug(node.Debug)
	}

	var (
		argKeys   = make([]language.Object, len(elem.Args))
		argValues = make([]language.Object, len(elem.Args))
		children  = make([]language.Object, len(elem.Children))
	)

	ir := i
	for i, attr := range elem.Args {
		argKeys[i] = language.NewString(attr.Name, node.Debug)
		argValues[i] = attr.Value
		zap.L().Debug("interpreter.element.args.prepare", zap.Uint("id", ir.ID), zap.String("name", attr.Name))
	}

	for i, child := range elem.Children {
		if child.Type == astnode.NodeTypeElementRawText {
			value := child.Content
			if child.IsEscaped {
				value = html.EscapeString(value)
			}
			children[i] = language.NewString(value, node.Debug)
		} else {
			children[i] = child.Value
		}
	}

	d, err := language.NewDict(argKeys, argValues, language.TypeString, language.TypeAny, node.Debug)
	if err != nil {
		zap.L().Error("interpreter.element.dict.error", zap.Uint("id", i.ID), zap.Error(err))
		return nil, exception.From(err, node.Debug, "dict type mismatch")
	}

	c := language.NewList(children, language.NewUnionType(language.TypeString, language.TypeHtml), node.Debug)

	inst, _ := cctx.NewInstance()
	initFunc, _ := inst.GetPrototype().GetObject(i.ctx, "init")
	init := initFunc.(*language.Function)
	cctxInstance, err := init.Data(i.ctx, []language.Object{d, c})
	if err != nil {
		zap.L().Error("interpreter.element.context.error", zap.Uint("id", i.ID), zap.Error(err))
		return nil, exception.From(err, node.Debug, "Context instance creation failed")
	}
	zap.L().Debug("interpreter.element.context.ready", zap.Uint("id", i.ID))

	data, err := fn.Data(i.ctx, []language.Object{cctxInstance})
	if err != nil {
		zap.L().Error("interpreter.element.fn.error", zap.Uint("id", i.ID), zap.Error(err))
		return nil, exception.From(err, node.Debug, "function call failed")
	}

	zap.L().Debug("interpreter.element.fn.success", zap.Uint("id", i.ID), zap.String("tagName", tagName))
	return data, nil
}
