package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
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
	default:
		ob, ok := i.GetObject(s)
		if !ok || ob.Type().Base() != language.ObjectTypeStructDefinition {
			return nil, newErr(ErrTypeMismatch, fmt.Sprintf("unknown type: %s", s), dg)
		}

		return ob.Type(), nil
	}
}

func (i *Interpreter) parseTypeNode(n *astnode.Node) (*language.Type, error) {
	if n == nil {
		return language.TypeAny, nil
	}

	if n.Type != astnode.NodeTypeType {
		return nil, fmt.Errorf("expected node type 'type', got '%s'", n.Type)
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
			return nil, err
		}
		t.Element = elem
		t.BaseType = language.ObjectTypeList
		return t, nil
	case "DICT":
		if len(n.Body) != 2 {
			return nil, fmt.Errorf("DICT must have exactly two body elements")
		}
		key, err := i.parseTypeNode(n.Body[0])
		if err != nil {
			return nil, err
		}
		val, err := i.parseTypeNode(n.Body[1])
		if err != nil {
			return nil, err
		}
		t.Key = key
		t.Value = val
		t.BaseType = language.ObjectTypeDict
		return t, nil
	case "FUNCTION":
		if n.ValueType == nil {
			return nil, fmt.Errorf("FUNCTION must have a return value_type")
		}
		ret, err := i.parseTypeNode(n.ValueType)
		if err != nil {
			return nil, err
		}
		t.Value = ret

		for _, arg := range n.Args {
			argType, err := i.parseTypeNode(arg)
			if err != nil {
				return nil, err
			}
			t.Args = append(t.Args, argType)
		}

		t.BaseType = language.ObjectTypeFunction
		return t, nil
	}

	baseType, err := i.stringToType(n.Content, n.Debug)
	if err != nil {
		return nil, err
	}

	t.BaseType = baseType.Base()

	if len(n.Children) > 0 {
		next, err := i.parseTypeNode(n.Children[0])
		if err != nil {
			return nil, err
		}
		t = language.NewUnionType(t, next)
	}

	return t, nil
}
