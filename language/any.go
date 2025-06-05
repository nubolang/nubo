package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

type Any struct {
	Data  Object
	debug *debug.Debug
}

func NewAny(value Object, debug *debug.Debug) *Any {
	return &Any{
		Data:  value,
		debug: debug,
	}
}

func (i *Any) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Any) Type() *Type {
	return i.Data.Type()
}

func (i *Any) Inspect() string {
	return fmt.Sprintf("<Object(any => %s)>", i.Data.Inspect())
}

func (i *Any) TypeString() string {
	return i.Data.TypeString()
}

func (i *Any) String() string {
	return i.Data.String()
}

func (i *Any) GetPrototype() Prototype {
	return nil
}

func (i *Any) Value() any {
	return i.Data.Value()
}

func (i *Any) Debug() *debug.Debug {
	return i.debug
}

func (i *Any) Clone() *Any {
	return NewAny(i.Data.Clone(), i.debug)
}
