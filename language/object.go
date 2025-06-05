package language

import "github.com/nubolang/nubo/internal/debug"

type ObjectType int

const (
	ObjectTypeInt ObjectType = iota
	ObjectTypeFloat
	ObjectTypeBool
	ObjectTypeString
	ObjectTypeChar
	ObjectTypeByte
	ObjectTypeList
	ObjectTypeDict
	ObjectTypeStructInstance
	ObjectTypeFunction
	ObjectTypeNil
	ObjectTypeAny
	ObjectTypeVoid
)

func (ot ObjectType) String() string {
	switch ot {
	case ObjectTypeInt:
		return "int"
	case ObjectTypeFloat:
		return "float"
	case ObjectTypeBool:
		return "bool"
	case ObjectTypeString:
		return "string"
	case ObjectTypeChar:
		return "char"
	case ObjectTypeByte:
		return "byte"
	case ObjectTypeList:
		return "list"
	case ObjectTypeDict:
		return "dict"
	case ObjectTypeStructInstance:
		return "struct"
	case ObjectTypeFunction:
		return "function"
	case ObjectTypeNil:
		return "nil"
	case ObjectTypeAny:
		return "any"
	case ObjectTypeVoid:
		return "void"
	default:
		return "unknown"
	}
}

func (ot ObjectType) Base() ObjectType {
	return ot
}

func (ot ObjectType) Hashable() bool {
	switch ot {
	case ObjectTypeInt, ObjectTypeFloat, ObjectTypeBool, ObjectTypeString, ObjectTypeChar, ObjectTypeByte:
		return true
	default:
		return false
	}
}

type Prototype interface {
	Objects() map[string]Object
	GetObject(name string) (Object, bool)
	SetObject(name string, value Object) error
}

type Object interface {
	ID() string

	Type() *Type
	Inspect() string

	TypeString() string
	String() string

	GetPrototype() Prototype
	Value() any

	Debug() *debug.Debug
	Clone() Object
}
