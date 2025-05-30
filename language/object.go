package language

import "github.com/nubogo/nubo/internal/debug"

type ObjectType int

const (
	TypeInt ObjectType = iota
	TypeFloat
	TypeBool
	TypeString
	TypeChar
	TypeByte
	TypeList
	TypeDict
	TypeStructInstance
	TypeFunction
	TypeNil
	TypeAny
)

func (ot ObjectType) String() string {
	switch ot {
	case TypeInt:
		return "Int"
	case TypeFloat:
		return "Float"
	case TypeBool:
		return "Bool"
	case TypeString:
		return "String"
	case TypeChar:
		return "Char"
	case TypeByte:
		return "Byte"
	case TypeList:
		return "List"
	case TypeDict:
		return "Dict"
	case TypeStructInstance:
		return "Struct"
	case TypeFunction:
		return "Function"
	case TypeNil:
		return "Nil"
	case TypeAny:
		return "Any"
	default:
		return "Unknown"
	}
}

func (ot ObjectType) Hashable() bool {
	switch ot {
	case TypeInt, TypeFloat, TypeBool, TypeString, TypeChar, TypeByte:
		return true
	default:
		return false
	}
}

type Prototype interface {
	GetMethod(name string) (Object, bool)
	SetMethod(name string, value Object) error

	GetObject(name string) (Object, bool)
	SetObject(name string, value Object) error
}

type Object interface {
	ID() string

	Type() ObjectType
	Inspect() string

	TypeString() string
	String() string

	GetPrototype() Prototype
	Value() any

	Debug() *debug.Debug
	Clone() Object
}
