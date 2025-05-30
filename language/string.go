package language

import (
	"fmt"
	"strconv"

	"github.com/nubogo/nubo/internal/debug"
)

type String struct {
	Data  string
	debug *debug.Debug
}

func NewString(value string, debug *debug.Debug) *String {
	return &String{
		Data:  value,
		debug: debug,
	}
}

func (i *String) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *String) Type() ObjectComplexType {
	return TypeString
}

func (i *String) Inspect() string {
	return fmt.Sprintf("<Object(String @ %s)>", strconv.Quote(i.String()))
}

func (i *String) TypeString() string {
	return "<Object(String)>"
}

func (i *String) String() string {
	return i.Data
}

func (i *String) GetPrototype() Prototype {
	return nil
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
