package language

import (
	"context"
	"fmt"
	"strings"
)

type IfaceTypeFn struct {
	Name    string
	Args    []*Type
	Returns *Type
}

type IfaceType struct {
	Functions []IfaceTypeFn
}

// Type represents a type in the language.
type Type struct {
	BaseType ObjectType
	Content  string     // "int", "string", etc. (used for simple types)
	Key      *Type      // if Kind == "DICT", represents the key type
	Value    *Type      // if Kind == "DICT", represents the value type or if Kind == "FUNCTION", represents the return type
	Element  *Type      // if Kind == "LIST", represents the element type
	Args     []*Type    // if Kind == "FUNCTION", represents the function argument types
	ID       string     // if BaseType == ObjectTypeStructInstance, represents the struct ID
	Next     *Type      // if it's an union type, represents the next type in the union
	Iface    *IfaceType // if Kind == "IFACE", represents the compare methods

	Object Object
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
	TypeHtml             = &Type{BaseType: ObjectTypeHtml, Content: "html"}
	TypeIface            = Nullable(&Type{BaseType: ObjectTypeIface, Content: "iface"})
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
	return NewUnionType(typ.DeepClone(), TypeNil.DeepClone())
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
	case ObjectTypeVoid:
		return nil
	case ObjectTypeHtml:
		return NewElement(nil, nil)
	case ObjectTypeIface:
		return Nil
	default:
		return Nil
	}
}

func (t *Type) Base() ObjectType {
	return t.BaseType.Base()
}

func (t *Type) String() string {
	if t == nil {
		return "void"
	}

	var next = ""
	if t.Next != nil {
		next = "|" + t.Next.String()
	}

	switch t.BaseType {
	default:
		return t.BaseType.String() + next
	case ObjectTypeNil:
		return "(nil)"
	case ObjectTypeString:
		return t.Content
	case ObjectTypeFunction:
		args := make([]string, len(t.Args))
		for i, arg := range t.Args {
			args[i] = arg.String()
		}
		return fmt.Sprintf("%s(%s) %s%s", t.BaseType.String(), strings.Join(args, ", "), t.Value.String(), next)
	case ObjectTypeList:
		if t.Element.Next != nil {
			return fmt.Sprintf("[](%s)%s", t.Element.String(), next)
		}
		return fmt.Sprintf("[]%s%s", t.Element.String(), next)
	case ObjectTypeDict:
		return fmt.Sprintf("dict[%s, %s]%s", t.Key.String(), t.Value.String(), next)
	case ObjectTypeStructDefinition:
		return fmt.Sprintf("(struct) %s%s", t.Content, next)
	case ObjectTypeStructInstance:
		return fmt.Sprintf("%s{}%s", t.Content, next)
	case ObjectTypeIface:
		var sb strings.Builder

		fnsLen := len(t.Iface.Functions) - 1
		for i, fn := range t.Iface.Functions {
			sb.WriteString(fn.Name)
			sb.WriteRune('(')

			argsLen := len(fn.Args) - 1
			for j, arg := range fn.Args {
				sb.WriteString(arg.String())
				if j < argsLen {
					sb.WriteString(", ")
				}
			}

			sb.WriteString(") ")
			sb.WriteString(fn.Returns.String())

			if i < fnsLen {
				sb.WriteString("; ")
			}
		}

		defer sb.Reset()
		return fmt.Sprintf("iface{%s}%s", sb.String(), next)
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
		return false
	}

	if t.BaseType == ObjectTypeAny {
		return true
	}

	// If other is an iface, check if t satisfies it.
	if other.BaseType == ObjectTypeIface {
		return t.SatisfiesIface(other)
	}
	// If t is an iface, check if other satisfies it.
	if t.BaseType == ObjectTypeIface {
		return other.SatisfiesIface(t)
	}

	if other.Base() == ObjectTypeNil {
		ok := t.BaseType == ObjectTypeNil ||
			t.BaseType == ObjectTypeList ||
			t.BaseType == ObjectTypeDict ||
			t.BaseType == ObjectTypeStructInstance ||
			t.BaseType == ObjectTypeStructDefinition ||
			t.BaseType == ObjectTypeFunction
		if !ok {
			return t.NextMatch(other)
		}
		return ok
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

		// Parameter types are checked in reverse direction.
		for i := range t.Args {
			if !other.Args[i].Compare(t.Args[i]) {
				return t.NextMatch(other)
			}
		}

		// Return type is checked in normal direction.
		ok := t.Value.Compare(other.Value)
		if !ok {
			return t.NextMatch(other)
		}
		return ok

	case ObjectTypeStructDefinition, ObjectTypeStructInstance:
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
		return types[0].DeepClone()
	}

	head := types[0].DeepClone()
	current := head

	for _, t := range types[1:] {
		current.Next = t.DeepClone()
		current = current.Next
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
			if arg != nil {
				clone.Args[i] = arg.DeepClone()
			}
		}
	}

	return clone
}

func (t *Type) SatisfiesIface(iface *Type) bool {
	if iface == nil || iface.Iface == nil {
		return true
	}

	for _, required := range iface.Iface.Functions {
		if !t.hasIfaceMethod(required) {
			return false
		}
	}

	return true
}

func (t *Type) hasIfaceMethod(fn IfaceTypeFn) bool {
	if t.Object == nil {
		return false
	}

	proto := t.Object.GetPrototype()
	if proto == nil {
		return false
	}

	ob, ok := proto.GetObject(context.Background(), fn.Name)
	if !ok {
		return false
	}

	obType := ob.Type()

	if obType.BaseType != ObjectTypeFunction {
		return false
	}

	if !obType.Value.Compare(fn.Returns) {
		return false
	}

	if len(fn.Args) != len(obType.Args) {
		return false
	}

	for i, arg := range fn.Args {
		if !arg.Compare(obType.Args[i]) {
			return false
		}
	}

	return true
}
