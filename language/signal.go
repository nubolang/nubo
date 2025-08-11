package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

type Signal struct {
	Data  string
	Carry Object
	debug *debug.Debug
}

func NewSignal(signal string, debug *debug.Debug) *Signal {
	return &Signal{
		Data:  signal,
		debug: debug,
	}
}

func (i *Signal) SetCarry(carry Object) *Signal {
	i.Carry = carry
	return i
}

func (i *Signal) GetCarry() Object {
	return i.Carry
}

func (i *Signal) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Signal) Type() *Type {
	return &Type{BaseType: ObjectTypeSignal}
}

func (i *Signal) Inspect() string {
	return fmt.Sprintf("<control-signal %s>", i.Data)
}

func (i *Signal) TypeString() string {
	return "<control-signal>"
}

func (i *Signal) String() string {
	return i.Data
}

func (i *Signal) GetPrototype() Prototype {
	return nil
}

func (i *Signal) Value() any {
	return i.Data
}

func (i *Signal) Debug() *debug.Debug {
	return i.debug
}

func (i *Signal) Clone() Object {
	return NewSignal(i.Data, i.debug)
}
