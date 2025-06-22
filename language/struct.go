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

type Struct struct {
	Name       string
	structType *Type
	Data       []StructField
	prototype  *StructPrototype
	debug      *debug.Debug
}

func NewStruct(name string, fields []StructField, debug *debug.Debug) *Struct {
	structType := &Type{
		BaseType: ObjectTypeStructDefinition,
		Content:  name,
	}

	s := &Struct{
		Name:       name,
		structType: structType,
		Data:       fields,
		debug:      debug,
	}

	structType.ID = fmt.Sprintf("%p", s)

	return s
}

func (s *Struct) NewInstance() (*StructInstance, error) {
	if s.prototype == nil {
		s.prototype = NewStructPrototype(s)
	}

	return NewStructInstance(s, s.Name, s.Debug())
}

func (s *Struct) Settable() []string {
	var settable = make([]string, len(s.Data))
	for i, field := range s.Data {
		settable[i] = field.Name
	}
	return settable
}

func (i *Struct) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Struct) Type() *Type {
	return i.structType
}

func (i *Struct) Inspect() string {
	return fmt.Sprintf("<Object(struct @ %s)>", i.String())
}

func (i *Struct) TypeString() string {
	typ := i.Name + "{"
	for inx, field := range i.Data {
		typ += fmt.Sprintf("%s: %s", field.Name, field.Type.String())
		if inx < len(i.Data)-1 {
			typ += ", "
		}
	}
	return typ + "}"
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
