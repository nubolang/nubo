package language

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type StructField struct {
	Name    string
	Type    *Type
	Private bool
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
	privateMap map[string]struct{}
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
		privateMap: make(map[string]struct{}, len(fields)),
	}

	for _, field := range fields {
		if field.Private {
			s.privateMap[field.Name] = struct{}{}
		}
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
	objs := i.GetPrototype().Objects()
	if len(objs) == 0 {
		return fmt.Sprintf("(struct %s) %s{}", i.Type().ID, i.Name)
	}

	var items []string = make([]string, 0, len(objs))
	for name, item := range objs {
		if _, ok := i.privateMap[name]; ok {
			name = fmt.Sprintf("(private) %s", name)
		}

		items = append(items, fmt.Sprintf("%s: %s", name, indentString(item.Type().String(), "\t")))
	}

	return fmt.Sprintf(
		"(struct %s) %s{\n\t%s\n}",
		i.Type().ID,
		i.Name,
		strings.Join(items, ",\n\t"),
	)
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
	objs := i.Data
	if len(objs) == 0 {
		return fmt.Sprintf("(struct) %s{}", i.Name)
	}

	var items []string = make([]string, len(objs))
	for i, item := range objs {
		name := item.Name
		if item.Private {
			name = fmt.Sprintf("(private) %s", name)
		}
		items[i] = fmt.Sprintf("%s: %s", name, indentString(item.Type.String(), "\t"))
	}

	return fmt.Sprintf(
		"(struct) %s{\n\t%s\n}",
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
