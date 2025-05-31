package language

import (
	"fmt"
	"strconv"

	"github.com/nubogo/nubo/internal/debug"
)

type Int struct {
	Data  int64
	debug *debug.Debug
}

func NewInt(value int64, debug *debug.Debug) *Int {
	return &Int{
		Data:  value,
		debug: debug,
	}
}

func (i *Int) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Int) Type() ObjectComplexType {
	return TypeInt
}

func (i *Int) Inspect() string {
	return fmt.Sprintf("<Object(int @ %s)>", i.String())
}

func (i *Int) TypeString() string {
	return "<Object(int)>"
}

func (i *Int) String() string {
	return strconv.Itoa(int(i.Data))
}

func (i *Int) GetPrototype() Prototype {
	return nil
}

func (i *Int) Value() any {
	return i.Data
}

func (i *Int) Debug() *debug.Debug {
	return i.debug
}

func (i *Int) Clone() Object {
	return NewInt(i.Data, i.debug)
}
