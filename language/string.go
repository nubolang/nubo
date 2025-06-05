package language

import (
	"fmt"
	"strconv"

	"github.com/nubolang/nubo/internal/debug"
)

type String struct {
	Data      string
	prototype *StringPrototype
	debug     *debug.Debug
}

func NewString(value string, debug *debug.Debug) *String {
	return &String{
		Data:      value,
		prototype: nil,
		debug:     debug,
	}
}

func (i *String) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *String) Type() *Type {
	return TypeString
}

func (i *String) Inspect() string {
	return fmt.Sprintf("<Object(string @ %s)>", strconv.Quote(i.String()))
}

func (i *String) TypeString() string {
	return "<Object(string)>"
}

func (i *String) String() string {
	return i.Data
}

func (i *String) GetPrototype() Prototype {
	if i.prototype == nil {
		i.prototype = NewStringPrototype(i)
	}
	return i.prototype
}

func (i *String) Value() any {
	return i.Data
}

func (i *String) Debug() *debug.Debug {
	return i.debug
}

func (i *String) Clone() Object {
	return NewString(i.Data, i.debug)
}
