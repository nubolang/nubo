package language

import (
	"strings"
)

// Type represents a type in the language.
type Type struct {
	BaseType ObjectComplexType
	Kind     string  // "", "LIST", "DICT"
	Content  string  // "int", "string", etc. (used for simple types)
	Key      *Type   // if Kind == "DICT", represents the key type
	Value    *Type   // if Kind == "DICT", represents the value type
	Element  *Type   // if Kind == "LIST", represents the element type
	Args     []*Type // if Kind == "FUNCTION", represents the function argument types
	Next     *Type   // if it's an union type, represents the next type in the union
}

func (t *Type) Base() ObjectType {
	return t.BaseType.Base()
}

func (t *Type) String() string {
	switch t.Kind {
	case "LIST":
		if t.Element != nil {
			return "List<" + t.Element.String() + ">"
		}
		return "List<any>"
	case "DICT":
		if t.Key != nil && t.Value != nil {
			return "Dict<" + t.Key.String() + ", " + t.Value.String() + ">"
		}
		return "Dict<any, any>"
	case "FUNCTION":
		args := make([]string, len(t.Args))
		for i, a := range t.Args {
			args[i] = a.String()
		}
		return "Function(" + strings.Join(args, ", ") + ") -> " + t.Value.String()
	default:
		if t.Content != "" {
			return t.Content
		}
		return t.BaseType.String()
	}
}

func (t *Type) Compare(other ObjectComplexType) bool {
	if t.Kind == "" {
		return t.Base().Compare(other)
	}

	o, ok := other.(*Type)
	if !ok {
		return false
	}

	if t.BaseType != o.BaseType || t.Kind != o.Kind || t.Content != o.Content {
		return false
	}

	switch t.Kind {
	case "LIST":
		return t.Element != nil && o.Element != nil && t.Element.Compare(o.Element)
	case "DICT":
		return t.Key != nil && o.Key != nil && t.Key.Compare(o.Key) &&
			t.Value != nil && o.Value != nil && t.Value.Compare(o.Value)
	case "FUNCTION":

		if len(t.Args) != len(o.Args) {
			return false
		}
		for i := range t.Args {
			if !t.Args[i].Compare(o.Args[i]) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

type StructDef struct {
	Name   string
	Fields []StructField
}

type StructFieldDef struct {
	Name string
	Type *Type
}
