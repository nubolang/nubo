package language

import "github.com/nubolang/nubo/internal/debug"

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
	TypeVoid
)

func (ot ObjectType) String() string {
	switch ot {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeBool:
		return "bool"
	case TypeString:
		return "string"
	case TypeChar:
		return "char"
	case TypeByte:
		return "byte"
	case TypeList:
		return "list"
	case TypeDict:
		return "dict"
	case TypeStructInstance:
		return "struct"
	case TypeFunction:
		return "function"
	case TypeNil:
		return "nil"
	case TypeAny:
		return "any"
	default:
		return "unknown"
	}
}

func (ot ObjectType) Base() ObjectType {
	return ot
}

func (ot ObjectType) Hashable() bool {
	switch ot {
	case TypeInt, TypeFloat, TypeBool, TypeString, TypeChar, TypeByte:
		return true
	default:
		return false
	}
}

type ObjectComplexType interface {
	Base() ObjectType
	String() string
}

type Prototype interface {
	Objects() map[string]Object
	GetObject(name string) (Object, bool)
	SetObject(name string, value Object) error
}

type Object interface {
	ID() string

	Type() ObjectComplexType
	Inspect() string

	TypeString() string
	String() string

	GetPrototype() Prototype
	Value() any

	Debug() *debug.Debug
	Clone() Object
}
