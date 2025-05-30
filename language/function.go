package language

import (
	"fmt"

	"github.com/nubogo/nubo/internal/debug"
)

type FnArg interface {
	Type() ObjectComplexType
	Name() string
}

type Function struct {
	Data       func(args []Object) (Object, error)
	ArgType    []FnArg
	ReturnType ObjectComplexType
	debug      *debug.Debug
}

func NewFunction(data func([]Object) (Object, error), debug *debug.Debug) *Function {
	return &Function{
		Data:  data,
		debug: debug,
	}
}

func (i *Function) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Function) Type() ObjectComplexType {
	return TypeFunction
}

func (i *Function) Inspect() string {
	return fmt.Sprintf("<Object(Function @ %s)>", i.String())
}

func (i *Function) TypeString() string {
	return "<Object(fn)>"
}

func (i *Function) String() string {
	var argTypes []string
	for _, arg := range i.ArgType {
		argTypes = append(argTypes, arg.Type().String())
	}
	return fmt.Sprintf("Closure(%p args=%v returns=%s)", i, argTypes, i.ReturnType)
}

func (i *Function) GetPrototype() Prototype {
	return nil
}

func (i *Function) Value() any {
	return i.Data
}

func (i *Function) Debug() *debug.Debug {
	return i.debug
}

func (i *Function) Clone() Object {
	return NewFunction(i.Data, i.debug)
}
