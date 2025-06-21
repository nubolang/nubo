package language

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type StructInstance struct {
	base      *Struct
	Name      string
	prototype *StructPrototype
	debug     *debug.Debug
}

func NewStructInstance(base *Struct, name string, debug *debug.Debug) (*StructInstance, error) {
	s := &StructInstance{
		base:  base,
		Name:  name,
		debug: debug,
	}

	proto, err := base.prototype.NewInstance(s)
	if err != nil {
		return nil, err
	}

	s.prototype = proto

	return s, nil
}

func (i *StructInstance) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *StructInstance) Type() *Type {
	return i.base.structType
}

func (i *StructInstance) Inspect() string {
	return fmt.Sprintf("<Object(struct @ %s)>", i.String())
}

func (i *StructInstance) TypeString() string {
	return "<Object(struct)>"
}

func (i *StructInstance) String() string {
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

func (i *StructInstance) GetPrototype() Prototype {
	return i.prototype
}

func (i *StructInstance) Value() any {
	return i
}

func (i *StructInstance) Debug() *debug.Debug {
	return i.debug
}

func (i *StructInstance) Clone() Object {
	return i
}
