package language

import (
	"fmt"
	"strconv"

	"github.com/nubolang/nubo/internal/debug"
)

type Char struct {
	Data  rune
	debug *debug.Debug
}

func NewChar(value rune, debug *debug.Debug) *Char {
	return &Char{
		Data:  value,
		debug: debug,
	}
}

func (i *Char) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Char) Type() ObjectComplexType {
	return TypeChar
}

func (i *Char) Inspect() string {
	return fmt.Sprintf("<Object(char @ %s)>", strconv.Quote(i.String()))
}

func (i *Char) TypeString() string {
	return "<Object(char)>"
}

func (i *Char) String() string {
	return string(i.Data)
}

func (i *Char) GetPrototype() Prototype {
	return nil
}

func (i *Char) Value() any {
	return i.Data
}

func (i *Char) Debug() *debug.Debug {
	return i.debug
}

func (i *Char) Clone() Object {
	return NewChar(i.Data, i.debug)
}
