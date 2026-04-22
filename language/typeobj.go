package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

type TypeObject struct {
	Data  *Type
	debug *debug.Debug
}

func NewTypeObject(t *Type, debug *debug.Debug) *TypeObject {
	return &TypeObject{
		Data:  t,
		debug: debug,
	}
}

func (t *TypeObject) ID() string {
	return fmt.Sprintf("%p", t)
}

func (t *TypeObject) Type() *Type {
	return withObject(t, TypeTypeObj)
}

func (t *TypeObject) Inspect() string {
	return fmt.Sprintf("(type) %s", t.String())
}

func (t *TypeObject) TypeString() string {
	return t.Data.String()
}

func (t *TypeObject) String() string {
	return t.Data.Content
}

func (t *TypeObject) GetPrototype() Prototype {
	return nil
}

func (t *TypeObject) Value() any {
	return t.Data
}

func (t *TypeObject) Debug() *debug.Debug {
	return t.debug
}

func (t *TypeObject) Clone() Object {
	return NewTypeObject(t.Data.DeepClone(), t.debug)
}
