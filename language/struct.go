package language

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type StructField struct {
	Name string
	Type*Type
}

type StructDefinition struct {
	Name   string
	Fields []StructField
	Debug  debug.Debug
}

/*func (sd *StructDefinition) NewStruct(values map[string]language.Object) *Struct {
	return &Struct{}
}*/

type Struct struct {
	Name      string
	Data      []StructField
	prototype *StructPrototype
	debug     *debug.Debug
}

func NewStruct(name string, fields []StructField, debug *debug.Debug) *Struct {
	return &Struct{
		Name:  name,
		Data:  fields,
		debug: debug,
	}
}

func (i *Struct) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Struct) Type()*Type {
	return TypeStructInstance
}

func (i *Struct) Inspect() string {
	return fmt.Sprintf("<Object(struct @ %s)>", i.String())
}

func (i *Struct) TypeString() string {
	return "<Object(struct)>"
}

func (i *Struct) String() string {
	var itemsString []string
	for name, item := range i.GetPrototype().Objects() {
		itemsString = append(itemsString, name+": "+item.String())
	}

	return fmt.Sprintf("%s{objects=[%s]}", i.Name, strings.Join(itemsString, ", "))
}

func (i *Struct) GetPrototype() Prototype {
	if i.prototype == nil {
		i.prototype = NewStructPrototype(i)
	}
	return i.prototype
}

func (i *Struct) Value() any {
	return i.Data
}

func (i *Struct) Debug() *debug.Debug {
	return i.debug
}

func (i *Struct) Clone() Object {
	return NewStruct(i.Name, i.Data, i.debug)
}
