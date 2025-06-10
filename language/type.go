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

func NewListType(elementType *Type) *Type {
	return &Type{BaseType: ObjectTypeList, Element: elementType}
}

func NewDictType(key *Type, value *Type) *Type {
	return &Type{BaseType: ObjectTypeDict, Key: key, Value: value}
}

func Nullable(typ *Type) *Type {
	return NewUnionType(typ, TypeNil)
}

func (t *Type) Base() ObjectType {
	return t.BaseType.Base()
}

func (t *Type) String() string {
	if t == nil {
		return "<invalid>"
	}

	var next string
	if t.Next != nil {
		next = "|" + t.Next.String()
	}

	switch t.BaseType {
	default:
		return t.BaseType.String() + next
	case ObjectTypeFunction:
		args := make([]string, len(t.Args))
		for i, arg := range t.Args {
			args[i] = arg.String()
		}
		return fmt.Sprintf("%s(%s) %s%s", t.BaseType.String(), strings.Join(args, ", "), t.Value.String(), next)
	case ObjectTypeList:
		return fmt.Sprintf("[]%s%s", t.Element.String(), next)
	case ObjectTypeDict:
		return fmt.Sprintf("dict[%s, %s]%s", t.Key.String(), t.Value.String(), next)
	case ObjectTypeStructInstance:
		return fmt.Sprintf("%s{}%s", t.Content, next)
	}
}

func (t *Type) NextMatch(other *Type) bool {
	if t.Next == nil {
		return false
	}
	return t.Next.Compare(other)
}

func (t *Type) Compare(other *Type) bool {
	if t == nil || other == nil {
		return t.NextMatch(other)
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
		return t.NextMatch(other)
	}

	switch t.BaseType {
	case ObjectTypeList:
		ok := t.Element.Compare(TypeAny) || t.Element.Compare(other.Element)
		if !ok {
			return t.NextMatch(other)
		}
		return ok

	case ObjectTypeDict:
		keyMatch := t.Key.Compare(TypeAny) || t.Key.Compare(other.Key)
		valueMatch := t.Value.Compare(TypeAny) || t.Value.Compare(other.Value)
		ok := keyMatch && valueMatch
		if !ok {
			return t.NextMatch(other)
		}
		return ok

	case ObjectTypeFunction:
		if len(t.Args) != len(other.Args) {
			return t.NextMatch(other)
		}
		for i := range t.Args {
			if !t.Args[i].Compare(other.Args[i]) {
				return t.NextMatch(other)
			}
		}
		ok := t.Value.Compare(other.Value)
		if !ok {
			return t.NextMatch(other)
		}
		return ok
	}

	ok := t.Content == other.Content
	if !ok {
		return t.NextMatch(other)
	}
	return ok
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
