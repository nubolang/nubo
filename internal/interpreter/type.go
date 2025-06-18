package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) stringToType(s string) (*language.Type, error) {
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
		return nil, fmt.Errorf("unknown type: %s", s)
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

	baseType, err := i.stringToType(n.Content)
	if err != nil {
		return nil, err
	}

	t.BaseType = baseType.Base()
	return t, nil
}
