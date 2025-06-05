package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

type FnArg interface {
	Type() *Type
	Name() string
	Default() Object
}

type BasicFnArg struct {
	TypeVal    *Type
	NameVal    string
	DefaultVal Object
}

func (b *BasicFnArg) Type() *Type {
	return b.TypeVal
}

func (b *BasicFnArg) Name() string {
	return b.NameVal
}

func (b *BasicFnArg) Default() Object {
	return b.DefaultVal
}

type Function struct {
	Data       func(args []Object) (Object, error)
	ArgTypes   []FnArg
	ReturnType *Type
	typ        *Type
	debug      *debug.Debug
}

func NewFunction(data func([]Object) (Object, error), debug *debug.Debug) *Function {
	return &Function{
		Data:  data,
		debug: debug,
		typ:   &Type{BaseType: ObjectTypeFunction},
	}
}

func NewTypedFunction(argTypes []FnArg, returnType *Type, data func([]Object) (Object, error), debug *debug.Debug) *Function {
	args := make([]*Type, len(argTypes))
	for i, arg := range argTypes {
		args[i] = arg.Type()
	}

	typ := &Type{
		BaseType: ObjectTypeFunction,
		Value:    returnType,
		Args:     args,
	}

	fn := func(args []Object) (Object, error) {
		minRequiredArgs := 0
		for _, arg := range argTypes {
			if arg.Default() == nil {
				minRequiredArgs++
			}
		}

		if len(args) < minRequiredArgs {
			return nil, fmt.Errorf("expected %d (minimum %d) arguments, got %d", len(argTypes), minRequiredArgs, len(args))
		}

		provideArgs := make([]Object, len(argTypes))
		for i, argType := range argTypes {
			if i < len(args) {
				arg := args[i]
				if !argType.Type().Compare(arg.Type()) {
					return nil, fmt.Errorf("argument %d (%s) expected type %s, got %s", i+1, argType.Name(), argType.Type(), arg.Type())
				}
				provideArgs[i] = arg
			} else {
				if argType.Default() != nil {
					provideArgs[i] = argType.Default()
				} else {
					return nil, fmt.Errorf("missing required argument %d (%s)", i+1, argType.Name())
				}
			}
		}

		value, err := data(provideArgs)
		if err != nil {
			return nil, err
		}

		if value == nil {
			if returnType != TypeVoid {
				return nil, fmt.Errorf("expected return type %s, got %s", returnType.String(), TypeVoid.String())
			}
		} else if !returnType.Compare(value.Type()) {
			return nil, fmt.Errorf("expected return type %s, got %s", returnType.String(), value.Type().String())
		}

		return value, nil
	}

	return &Function{
		Data:       fn,
		ArgTypes:   argTypes,
		ReturnType: returnType,
		typ:        typ,
		debug:      debug,
	}
}

func (i *Function) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Function) Type() *Type {
	return i.typ
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
