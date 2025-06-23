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
	ID       string  // if BaseType == ObjectTypeStructInstance, represents the struct ID
	Next     *Type   // if it's an union type, represents the next type in the union
}

var (
	TypeInt              = &Type{BaseType: ObjectTypeInt, Content: "int"}
	TypeFloat            = &Type{BaseType: ObjectTypeFloat, Content: "float"}
	TypeBool             = &Type{BaseType: ObjectTypeBool, Content: "bool"}
	TypeString           = &Type{BaseType: ObjectTypeString, Content: "string"}
	TypeChar             = &Type{BaseType: ObjectTypeChar, Content: "char"}
	TypeByte             = &Type{BaseType: ObjectTypeByte, Content: "byte"}
	TypeList             = &Type{BaseType: ObjectTypeList, Element: TypeAny}
	TypeDict             = &Type{BaseType: ObjectTypeDict, Key: TypeAny, Value: TypeAny}
	TypeStructInstance   = Nullable(&Type{BaseType: ObjectTypeStructInstance})
	TypeStructDefinition = &Type{BaseType: ObjectTypeStructDefinition}
	TypeNil              = &Type{BaseType: ObjectTypeNil, Content: "nil"}
	TypeAny              = &Type{BaseType: ObjectTypeAny, Content: "any"}
	TypeVoid             = &Type{BaseType: ObjectTypeVoid, Content: "void"}
	TypeHtml             = &Type{BaseType: ObjectTypeString, Content: "html"}
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
	return NewUnionType(typ.DeepClone(), TypeNil)
}

func DefaultValue(typ *Type) Object {
	switch typ.BaseType {
	case ObjectTypeByte:
		return NewByte(0, nil)
	case ObjectTypeInt:
		return NewInt(0, nil)
	case ObjectTypeFloat:
		return NewFloat(0.0, nil)
	case ObjectTypeString:
		return NewString("", nil)
	case ObjectTypeBool:
		return NewBool(false, nil)
	case ObjectTypeChar:
		return NewChar('\000', nil)
	case ObjectTypeList:
		return NewList(nil, typ.Element, nil)
	case ObjectTypeDict:
		dict, _ := NewDict(nil, nil, typ.Key, typ.Value, nil)
		return dict
	case ObjectTypeStructInstance:
		return nil
	case ObjectTypeStructDefinition:
		return nil
	case ObjectTypeNil:
		return Nil
	case ObjectTypeAny:
		return Nil
	case ObjectTypeVoid:
		return nil
	default:
		return nil
	}
}

func (t *Type) Base() ObjectType {
	return t.BaseType.Base()
}

func (t *Type) String() string {
	if t == nil {
		return "<invalid>"
	}

	var next = ""
	if t.Next != nil {
		next = "|" + t.Next.String()
	}

	switch t.BaseType {
	default:
		return t.BaseType.String() + next
	case ObjectTypeString:
		return t.Content
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
	case ObjectTypeStructInstance, ObjectTypeStructDefinition:
		return fmt.Sprintf("%s[%s]{}%s", t.Content, t.ID, next)
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

	if t.BaseType != ObjectTypeStructDefinition && t.BaseType != ObjectTypeStructInstance {
		if t.BaseType != other.BaseType {
			return t.NextMatch(other)
		}
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
	case ObjectTypeStructDefinition:
		return t.ID == other.ID
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

	if len(types) == 1 {
		return types[0]
	}

	head := types[0]
	current := head.DeepClone()

	for _, t := range types[1:] {
		current.Next = t.DeepClone()
		current = t.DeepClone()
	}

	return head
}

func (t *Type) DeepClone() *Type {
	if t == nil {
		return nil
	}

	clone := &Type{
		BaseType: t.BaseType,
		Content:  t.Content,
		ID:       t.ID,
	}

	if t.Key != nil {
		clone.Key = t.Key.DeepClone()
	}

	if t.Value != nil {
		clone.Value = t.Value.DeepClone()
	}

	if t.Element != nil {
		clone.Element = t.Element.DeepClone()
	}

	if t.Next != nil {
		clone.Next = t.Next.DeepClone()
	}

	if len(t.Args) > 0 {
		clone.Args = make([]*Type, len(t.Args))
		for i, arg := range t.Args {
			clone.Args[i] = arg.DeepClone()
		}
	}

	return clone
}
