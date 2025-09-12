package interpreter

import (
	"fmt"
	"html"
	"strings"
	"unicode"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/packages/component"
	"github.com/nubolang/nubo/language"
	"github.com/stoewer/go-strcase"
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
			val, err := i.eval(child.Value.(*astnode.Node))
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

	tagName := node.Content
	if !endsWithCapitalTag(tagName) {
		return language.NewElement(elem, node.Debug), nil
	}

	ob, ok := i.GetObject(tagName)
	if !ok {
		return language.NewElement(elem, node.Debug), nil
	}

	fn, ok := ob.(*language.Function)
	if !ok {
		return language.NewElement(elem, node.Debug), nil
	}

	cctxRaw, _ := component.NewComponent(node.Debug).GetPrototype().GetObject("Context")
	cctx := cctxRaw.(*language.Struct)
	fnType := language.NewFunctionType(language.TypeHtml, cctx.Type())

	if !fnType.Compare(fn.Type()) {
		return nil, newErr(ErrInvalid, fmt.Sprintf("Invalid element, expected type: %s, got %s", fnType, fn.Type()), node.Debug)
	}

	var (
		argKeys   = make([]language.Object, len(elem.Args))
		argValues = make([]language.Object, len(elem.Args))
		children  = make([]language.Object, len(elem.Children))
	)

	for i, attr := range elem.Args {
		argKeys[i] = language.NewString(attr.Name, node.Debug)
		argValues[i] = attr.Value
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
		return nil, newErr(ErrTypeMismatch, err.Error(), node.Debug)
	}

	c := language.NewList(children, language.NewUnionType(language.TypeString, language.TypeHtml), node.Debug)

	inst, _ := cctx.NewInstance()
	initFunc, _ := inst.GetPrototype().GetObject("init")
	init := initFunc.(*language.Function)
	cctxInstance, err := init.Data([]language.Object{d, c})
	if err != nil {
		return nil, newErr(ErrTypeMismatch, err.Error(), node.Debug)
	}

	return fn.Data([]language.Object{cctxInstance})
}
