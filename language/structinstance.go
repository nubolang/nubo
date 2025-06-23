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
	cloned := i.base.structType.DeepClone()
	cloned.BaseType = ObjectTypeStructInstance
	return cloned
}

func (i *StructInstance) Inspect() string {
	return fmt.Sprintf("<Object(struct @ %s)>", i.String())
}

func (i *StructInstance) TypeString() string {
	return "<Object(struct)>"
}

func (i *StructInstance) String() string {
	if len(i.base.Data) == 0 {
		return fmt.Sprintf("%s[%s]{}", i.Name, i.base.structType.ID)
	}

	var items = make([]string, len(i.base.Data))
	for inx, field := range i.base.Data {
		ob, ok := i.GetPrototype().GetObject(field.Name)
		if !ok {
			items[inx] = fmt.Sprintf("%s: <invalid>", field.Name)
		} else {
			items[inx] = fmt.Sprintf("%s: %s", field.Name, indentString(ob.String(), "\t"))
		}
	}

	return fmt.Sprintf(
		"%s[%s]{\n\t%s\n}",
		i.Name,
		i.base.structType.ID,
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
