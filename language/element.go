package language

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/stoewer/go-strcase"
)

type ElementData struct {
	TagName   string
	Args      []Attribute
	Children  []ElementChild
	SelfClose bool
}

type Attribute struct {
	Name  string
	Kind  string // "DYNAMIC" or "TEXT"
	Value Object
}

type ElementChild struct {
	Type      astnode.NodeType // Element, ElementRawText, ElementDynamicText
	Content   string           // for RawText
	Value     *Element         // for DynamicText or nested Element
	IsEscaped bool
}

type Element struct {
	Data  *ElementData
	proto *ElementPrototype
	debug *debug.Debug
}

func NewElement(data *ElementData, debug *debug.Debug) *Element {
	return &Element{
		Data:  data,
		debug: debug,
	}
}

func (e *Element) ID() string {
	return fmt.Sprintf("%p", e)
}

func (e *Element) Type() *Type {
	return TypeString
}

func (e *Element) Inspect() string {
	return fmt.Sprintf("<Object(element @ %s)>", e.String())
}

func (e *Element) TypeString() string {
	return "<Object(Element(string))>"
}

func (e *Element) String() string {
	return e.Value().(string)
}

func (e *Element) GetPrototype() Prototype {
	if e.proto == nil {
		e.proto = NewElementPrototype(e)
	}
	return e.proto
}

func (e *Element) Value() any {
	var sb strings.Builder

	sb.WriteRune('<')
	sb.WriteString(e.Data.TagName)

	for _, arg := range e.Data.Args {
		attrName := strcase.KebabCase(arg.Name)

		if e.Data.TagName == "a" && attrName == "n-to" {
			sb.WriteRune(' ')
			sb.WriteString("href=")

			var valueStr string
			if arg.Value != nil {
				valueStr = html.EscapeString(arg.Value.String())
			}

			sb.WriteString(strconv.Quote(valueStr))

			sb.WriteRune(' ')
			sb.WriteString("n-to=\"\"")
			continue
		}

		sb.WriteRune(' ')
		sb.WriteString(attrName)

		sb.WriteRune('=')

		var valueStr string
		if arg.Value != nil {
			valueStr = html.EscapeString(arg.Value.String())
		}

		sb.WriteString(strconv.Quote(valueStr))
	}

	if e.Data.SelfClose {
		if len(e.Data.Args) > 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString("/>")
		return sb.String()
	}

	sb.WriteRune('>')

	for _, child := range e.Data.Children {
		switch child.Type {
		case astnode.NodeTypeElement:
			if child.Value != nil {
				sb.WriteString(child.Value.String())
			}
		case astnode.NodeTypeElementRawText:
			if child.IsEscaped {
				sb.WriteString(html.EscapeString(child.Content))
			} else {
				sb.WriteString(child.Content)
			}
		}
	}

	sb.WriteString("</")
	sb.WriteString(e.Data.TagName)
	sb.WriteString(">")

	return sb.String()
}

func (e *Element) Debug() *debug.Debug {
	return e.debug
}

func (e *Element) Clone() Object {
	// Deep copy attributes
	attrs := make([]Attribute, len(e.Data.Args))
	for i, a := range e.Data.Args {
		attrs[i] = Attribute{
			Name:  a.Name,
			Kind:  a.Kind,
			Value: a.Value,
		}
	}

	// Deep copy children
	children := make([]ElementChild, len(e.Data.Children))
	for i, c := range e.Data.Children {
		children[i] = ElementChild{
			Type:      c.Type,
			Content:   c.Content,
			IsEscaped: c.IsEscaped,
		}
		if c.Value != nil {
			// recursively clone nested elements
			clone := NewElement(c.Value.Data, e.debug)
			children[i].Value = clone
		}
	}

	// Deep copy data
	clonedData := &ElementData{
		TagName:   e.Data.TagName,
		Args:      attrs,
		Children:  children,
		SelfClose: e.Data.SelfClose,
	}

	return NewElement(clonedData, e.debug)
}
