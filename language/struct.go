package language

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type StructField struct {
	Name string
	Type *Type
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

func (i *Struct) Type() *Type {
	return TypeStructInstance
}

func (i *Struct) Inspect() string {
	return fmt.Sprintf("<Object(struct @ %s)>", i.String())
}

func (i *Struct) TypeString() string {
	return "<Object(struct)>"
}

func (i *Struct) String() string {
	objs := i.GetPrototype().Objects()
	if len(objs) == 0 {
		return fmt.Sprintf("%s{objects=[]}", i.Name)
	}

	var items []string
	for name, item := range objs {
		items = append(items, fmt.Sprintf("%s: %s", name, indentString(item.String(), "\t")))
	}

	return fmt.Sprintf(
		"%s{objects=[\n\t%s\n]}",
		i.Name,
		strings.Join(items, ",\n\t"),
	)
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
	return i
}
