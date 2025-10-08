package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) stringToType(s string, dg *debug.Debug) (*language.Type, error) {
	switch s {
	case "void":
		return language.TypeVoid, nil
	case "int":
		return language.TypeInt, nil
	case "string":
		return language.TypeString, nil
	case "bool":
		return language.TypeBool, nil
	case "float":
		return language.TypeFloat, nil
	case "byte":
		return language.TypeByte, nil
	case "char":
		return language.TypeChar, nil
	case "any":
		return language.TypeAny, nil
	case "html":
		return language.TypeHtml, nil
	case "nil":
		return language.TypeNil, nil
	default:
		return nil, typeError("invalid type '%s'", s).WithDebug(dg)
	}
}

func (i *Interpreter) parseTypeNode(n *astnode.Node) (*language.Type, error) {
	if n == nil {
		return language.TypeAny, nil
	}

	if n.Type != astnode.NodeTypeType {
		return nil, runExc("expected node type 'type', got '%s'", n.Type).WithDebug(n.Debug)
	}

	t := &language.Type{
		Content: n.Content,
	}

	switch n.Kind {
	case "LIST":
		if len(n.Body) != 1 {
			return nil, fmt.Errorf("LIST must have exactly one body element")
		}
		elem, err := i.parseTypeNode(n.Body[0])
		if err != nil {
			return nil, wrapRunExc(err, n.Body[0].Debug)
		}
		t.Element = elem
		t.BaseType = language.ObjectTypeList
		return i.checkAddUnionType(t, n)
	case "DICT":
		if len(n.Body) != 2 {
			return nil, exception.Create("DICT must have exactly two body elements, got %d", len(n.Body)).WithLevel(exception.LevelType).WithDebug(n.Debug)
		}
		key, err := i.parseTypeNode(n.Body[0])
		if err != nil {
			return nil, wrapRunExc(err, n.Body[0].Debug)
		}
		val, err := i.parseTypeNode(n.Body[1])
		if err != nil {
			return nil, wrapRunExc(err, n.Body[1].Debug)
		}
		t.Key = key
		t.Value = val
		t.BaseType = language.ObjectTypeDict
		return i.checkAddUnionType(t, n)
	case "FUNCTION":
		if n.ValueType == nil {
			return nil, fmt.Errorf("FUNCTION must have a return value_type")
		}
		ret, err := i.parseTypeNode(n.ValueType)
		if err != nil {
			return nil, wrapRunExc(err, n.ValueType.Debug)
		}
		t.Value = ret

		for _, arg := range n.Args {
			argType, err := i.parseTypeNode(arg)
			if err != nil {
				return nil, wrapRunExc(err, arg.Debug)
			}
			t.Args = append(t.Args, argType)
		}

		t.BaseType = language.ObjectTypeFunction
		return i.checkAddUnionType(t, n)
	}

	baseType, err := i.stringToType(n.Content, n.Debug)
	if err != nil {
		ob, ok := i.GetObject(n.Content)
		if !ok || ob.Type().Base() != language.ObjectTypeStructDefinition {
			return nil, runExc("unknown type: %q", n.Content).WithDebug(n.Debug)
		}
		if n.Flags.Contains("OPTIONAL") {
			return language.Nullable(ob.Type()), nil
		}
		return ob.Type(), nil
	}

	t.BaseType = baseType.Base()
	if n.Flags.Contains("OPTIONAL") {
		t = language.Nullable(t)
	}

	return i.checkAddUnionType(t, n)
}

func (i *Interpreter) checkAddUnionType(t *language.Type, n *astnode.Node) (*language.Type, error) {
	if len(n.Children) > 0 {
		next, err := i.parseTypeNode(n.Children[0])
		if err != nil {
			return nil, wrapRunExc(err, n.Debug)
		}

		union := language.NewUnionType(t, next)
		return union, nil
	}

	return t, nil
}
