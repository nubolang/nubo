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

func NewFunctionType(returnType *Type, argsType ...*Type) *Type {
	return &Type{BaseType: ObjectTypeFunction, Value: returnType, Args: argsType}
}

func (t *Type) Base() ObjectType {
	return t.BaseType.Base()
}

func (t *Type) String() string {
	if t == nil {
		return "<invalid>"
	}

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
	if t == nil || other == nil {
		return false
	}

	if t.BaseType == ObjectTypeAny {
		return true
	}

	if other.Base() == ObjectTypeNil {
		return t.BaseType == ObjectTypeNil ||
			t.BaseType == ObjectTypeList ||
			t.BaseType == ObjectTypeDict ||
			t.BaseType == ObjectTypeStructInstance ||
			t.BaseType == ObjectTypeFunction
	}

	if t.BaseType != other.BaseType {
		return false
	}

	switch t.BaseType {
	case ObjectTypeList:
		return t.Element.Compare(TypeAny) || t.Element.Compare(other.Element)

	case ObjectTypeDict:
		keyMatch := t.Key.Compare(TypeAny) || t.Key.Compare(other.Key)
		valueMatch := t.Value.Compare(TypeAny) || t.Value.Compare(other.Value)
		return keyMatch && valueMatch

	case ObjectTypeFunction:
		if len(t.Args) != len(other.Args) {
			return false
		}
		for i := range t.Args {
			if !t.Args[i].Compare(other.Args[i]) {
				return false
			}
		}
		return t.Value.Compare(other.Value)
	}

	return t.Content == other.Content
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
