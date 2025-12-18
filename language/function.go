package language

import (
	"context"
	"fmt"
	"strings"

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

type fnVersion int

const (
	fnVersionBase fnVersion = iota
	fnVersionTyped
)

type Function struct {
	Data       func(ctx context.Context, args []Object) (Object, error)
	ArgTypes   []FnArg
	ReturnType *Type
	typ        *Type
	debug      *debug.Debug
	proto      *FunctionPrototype
	ver        fnVersion
}

func NewFunction(data func(context.Context, []Object) (Object, error), debug *debug.Debug) *Function {
	return &Function{
		Data:  data,
		debug: debug,
		typ:   &Type{BaseType: ObjectTypeFunction},
		ver:   fnVersionBase,
	}
}

func NewTypedFunction(argTypes []FnArg, returnType *Type, data func(context.Context, []Object) (Object, error), debug *debug.Debug) *Function {
	args := make([]*Type, len(argTypes))
	for i, arg := range argTypes {
		args[i] = arg.Type()
	}

	typ := &Type{
		BaseType: ObjectTypeFunction,
		Value:    returnType,
		Args:     args,
	}

	fn := func(ctx context.Context, args []Object) (Object, error) {
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

		value, err := data(ctx, provideArgs)
		if err != nil {
			return nil, err
		}

		if value == nil {
			if !returnType.Compare(TypeVoid) {
				return nil, fmt.Errorf("expected return type %s, got %s", returnType.String(), TypeVoid.String())
			}
		} else if !returnType.Compare(value.Type()) {
			if returnType.BaseType == ObjectTypeVoid && value.Type().BaseType == ObjectTypeSignal && value.Value() == "return" {
				return value, nil
			}

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
		ver:        fnVersionTyped,
	}
}

func (i *Function) Call(ctx context.Context, args []Object, namedArgs ...map[string]Object) (Object, error) {
	if i.ver == fnVersionBase {
		return i.Data(ctx, args)
	}

	var realArgs = make([]Object, len(i.ArgTypes))

	copy(realArgs, args)

	if len(namedArgs) > 0 {
		for name, arg := range namedArgs[0] {
			for inx, argType := range i.ArgTypes {
				if argType.Name() == name {
					realArgs[inx] = arg
					break
				}
			}
		}
	}

	for inx, argType := range i.ArgTypes {
		if realArgs[inx] == nil {
			realArgs[inx] = argType.Default()
		}
	}

	return i.Data(ctx, realArgs)
}

func (i *Function) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Function) Type() *Type {
	return i.typ
}

func (i *Function) Inspect() string {
	objs := i.GetPrototype().Objects()
	if len(objs) == 0 {
		return fmt.Sprintf("(fn) %s {}", i.Type().String())
	}

	var items []string = make([]string, 0, len(objs))
	for name, item := range objs {
		items = append(items, fmt.Sprintf("%s: %s", name, indentString(item.Type().String(), "\t")))
	}

	return fmt.Sprintf(
		"(fn) %s {\n\t%s\n}",
		i.Type().String(),
		strings.Join(items, ",\n\t"),
	)
}

func (i *Function) TypeString() string {
	return "<Object(fn)>"
}

func (i *Function) String() string {
	var argTypes []string
	for _, arg := range i.Type().Args {
		argTypes = append(argTypes, arg.String())
	}

	var rt string
	if i.Type().Value == nil {
		rt = "void"
	} else {
		rt = i.Type().Value.String()
	}

	return fmt.Sprintf("(closure) (%v -> %s)", argTypes, rt)
}

func (i *Function) GetPrototype() Prototype {
	if i.proto == nil {
		i.proto = NewFunctionPrototype(i)
	}
	return i.proto
}

func (i *Function) Value() any {
	return i.Data
}

func (i *Function) Debug() *debug.Debug {
	return i.debug
}

func (i *Function) Clone() Object {
	return &Function{
		Data:       i.Data,
		ArgTypes:   i.ArgTypes,
		ReturnType: i.ReturnType,
		typ:        i.typ,
		debug:      i.debug,
	}
}
