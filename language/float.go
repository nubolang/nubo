package language

import (
	"fmt"
	"strconv"

	"github.com/nubogo/nubo/internal/debug"
)

type Float struct {
	Data  float64
	debug *debug.Debug
}

func NewFloat(value float64, debug *debug.Debug) *Float {
	return &Float{
		Data:  value,
		debug: debug,
	}
}

func (i *Float) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Float) Type() ObjectComplexType {
	return TypeFloat
}

func (i *Float) Inspect() string {
	return fmt.Sprintf("<Object(float @ %s)>", i.String())
}

func (i *Float) TypeString() string {
	return "<Object(Float)>"
}

func (i *Float) String() string {
	return strconv.FormatFloat(i.Data, 'f', -1, 64)
}

func (i *Float) GetPrototype() Prototype {
	return nil
}

func (i *Float) Value() any {
	return i.Data
}

func (i *Float) Debug() *debug.Debug {
	return i.debug
}

func (i *Float) Clone() Object {
	return NewFloat(i.Data, i.debug)
}
