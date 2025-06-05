package language

import (
	"fmt"
	"strings"
)

// Type represents a type in the language.
type Type struct {
	BaseType ObjectType
	Content  string  // "int", "string", etc. (used for simple types)
	Key      *Type   // if Kind == "DICT", represents the key type
	Value    *Type   // if Kind == "DICT", represents the value type or if Kind == "FUNCTION", represents the return type
	Element  *Type   // if Kind == "LIST", represents the element type
	Args     []*Type // if Kind == "FUNCTION", represents the function argument types
	Next     *Type   // if it's an union type, represents the next type in the union
}

var (
	TypeInt            = &Type{BaseType: ObjectTypeInt, Content: "int"}
	TypeFloat          = &Type{BaseType: ObjectTypeFloat, Content: "float"}
	TypeBool           = &Type{BaseType: ObjectTypeBool, Content: "bool"}
	TypeString         = &Type{BaseType: ObjectTypeString, Content: "string"}
	TypeChar           = &Type{BaseType: ObjectTypeChar, Content: "char"}
	TypeByte           = &Type{BaseType: ObjectTypeByte, Content: "byte"}
	TypeList           = &Type{BaseType: ObjectTypeList, Element: TypeAny}
	TypeDict           = &Type{BaseType: ObjectTypeDict}
	TypeStructInstance = &Type{BaseType: ObjectTypeStructInstance}
	TypeNil            = &Type{BaseType: ObjectTypeNil, Content: "nil"}
	TypeAny            = &Type{BaseType: ObjectTypeAny, Content: "any"}
	TypeVoid           = &Type{BaseType: ObjectTypeVoid, Content: "void"}
)

func (t *Type) Base() ObjectType {
	return t.BaseType.Base()
}

func (t *Type) String() string {
	switch t.BaseType {
	default:
		return t.BaseType.String()
	case ObjectTypeFunction:
		args := make([]string, len(t.Args))
		for i, arg := range t.Args {
			args[i] = arg.String()
		}
		return fmt.Sprintf("%s(%s) %s", t.BaseType.String(), strings.Join(args, ", "), t.Value.String())
	case ObjectTypeList:
		return fmt.Sprintf("[]%s", t.Element.String())
	case ObjectTypeDict:
		return fmt.Sprintf("dict[%s, %s]", t.Key.String(), t.Value.String())
	case ObjectTypeStructInstance:
		return fmt.Sprintf("%s{}", t.Content)
	}
}

func (t *Type) Compare(other *Type) bool {
	if t.BaseType == ObjectTypeAny {
		return true
	}

	if other.Base() == ObjectTypeNil {
		return t.BaseType == ObjectTypeNil || t.BaseType == ObjectTypeList || t.BaseType == ObjectTypeDict || t.BaseType == ObjectTypeStructInstance || t.BaseType == ObjectTypeFunction
	}

	switch t.BaseType {
	case ObjectTypeList:
		return t.Element.String() == TypeAny.String() || t.Element.String() == other.Element.String()
	case ObjectTypeDict:
		return t.Key.String() == TypeAny.String() || t.Key.String() == other.Key.String() && t.Value.String() == TypeAny.String() || t.Value.String() == other.Value.String()
	}

	return t.String() == other.String()
}

func NewUnionType(types ...*Type) *Type {
	if len(types) == 0 {
		return nil
	}

	head := types[0]
	current := head

	for _, t := range types[1:] {
		current.Next = t
		current = t
	}

	return head
}
