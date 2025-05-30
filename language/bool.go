package language

import (
	"fmt"
	"strconv"

	"github.com/nubogo/nubo/internal/debug"
)

type Bool struct {
	Data  bool
	debug *debug.Debug
}

func NewBool(value bool, debug *debug.Debug) *Bool {
	return &Bool{
		Data:  value,
		debug: debug,
	}
}

func (i *Bool) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Bool) Type() ObjectType {
	return TypeBool
}

func (i *Bool) Inspect() string {
	return fmt.Sprintf("<Object(Bool @ %s)>", i.String())
}

func (i *Bool) TypeString() string {
	return "<Object(Bool)>"
}

func (i *Bool) String() string {
	return strconv.FormatBool(i.Data)
}

func (i *Bool) GetPrototype() Prototype {
	return nil
}

func (i *Bool) Value() any {
	return i.Data
}

func (i *Bool) Debug() *debug.Debug {
	return i.debug
}

func (i *Bool) Clone() Object {
	return NewBool(i.Data, i.debug)
}
