package language

import (
	"fmt"

	"github.com/nubogo/nubo/internal/debug"
)

type FnArg interface {
	Type() ObjectComplexType
	Name() string
	Default() Object
}

type Function struct {
	Data       func(args []Object) (Object, error)
	ArgTypes   []FnArg
	ReturnType ObjectComplexType
	debug      *debug.Debug
}

func NewFunction(data func([]Object) (Object, error), debug *debug.Debug) *Function {
	return &Function{
		Data:  data,
		debug: debug,
	}
}

func NewTypedFunction(argTypes []FnArg, returnType ObjectComplexType, data func([]Object) (Object, error), debug *debug.Debug) *Function {
	fn := func(args []Object) (Object, error) {
		if len(args) != len(argTypes) {
			return nil, fmt.Errorf("expected %d arguments, got %d", len(argTypes), len(args))
		}

		for i, arg := range args {
			if argTypes[i].Type() != TypeAny && arg.Type() != argTypes[i].Type() {
				return nil, fmt.Errorf("argument %d (%s) expected type %s, got %s", i, argTypes[i].Name(), argTypes[i].Type(), arg.Type())
			}
		}

		value, err := data(args)
		if err != nil {
			return nil, err
		}

		if value.Type() != returnType {
			return nil, fmt.Errorf("expected return type %s, got %s", returnType.String(), value.Type().String())
		}

		return value, nil
	}

	return &Function{
		Data:       fn,
		ArgTypes:   argTypes,
		ReturnType: returnType,
		debug:      debug,
	}
}

func (i *Function) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Function) Type() ObjectComplexType {
	return TypeFunction
}

func (i *Function) Inspect() string {
	return fmt.Sprintf("<Object(fn @ %s)>", i.String())
}

func (i *Function) TypeString() string {
	return "<Object(fn)>"
}

func (i *Function) String() string {
	var argTypes []string
	for _, arg := range i.ArgTypes {
		argTypes = append(argTypes, arg.Type().String())
	}

	var rt string
	if i.ReturnType == nil {
		rt = "void"
	} else {
		rt = i.ReturnType.String()
	}

	return fmt.Sprintf("Closure(%p args=%v returns=%s)", i, argTypes, rt)
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
