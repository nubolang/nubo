package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

type StructField struct {
	Name  string
	Type  ObjectComplexType
	Value Object
}

type Struct struct {
	Data  []StructField
	debug *debug.Debug
}

func NewStruct(fields []StructField, debug *debug.Debug) *Struct {
	return &Struct{
		Data:  fields,
		debug: debug,
	}
}

func (i *Struct) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Struct) Type() ObjectComplexType {
	return TypeStructInstance
}

func (i *Struct) Inspect() string {
	return fmt.Sprintf("<Object(struct @ %s)>", i.String())
}

func (i *Struct) TypeString() string {
	return "<Object(struct)>"
}

func (i *Struct) String() string {
	return "struct string"
}

func (i *Struct) GetPrototype() Prototype {
	return nil
}

func (i *Struct) Value() any {
	return i.Data
}

func (i *Struct) Debug() *debug.Debug {
	return i.debug
}

func (i *Struct) Clone() Object {
	return NewStruct(i.Data, i.debug)
}
