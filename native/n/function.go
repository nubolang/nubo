package n

import (
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
)

type FnDescriber struct {
	args    []*FnArg
	returns *language.Type
}

type FnArg struct {
	Type    *language.Type
	Name    string
	Default language.Object
}

func Arg(name string, typ *language.Type, def ...language.Object) *FnArg {
	arg := &FnArg{
		Type: typ,
		Name: name,
	}

	if len(def) > 0 {
		arg.Default = def[0]
	}

	return arg
}

func Describe(args ...*FnArg) *FnDescriber {
	return &FnDescriber{
		args:    args,
		returns: language.TypeVoid,
	}
}

func EmptyDescribe() *FnDescriber {
	return &FnDescriber{
		args:    nil,
		returns: language.TypeVoid,
	}
}

func (fd *FnDescriber) Returns(returns *language.Type) *FnDescriber {
	fd.returns = returns
	return fd
}

type Args struct {
	typedArgs    []language.FnArg
	providedArgs []language.Object
}

func (a *Args) Get(inx int) language.Object {
	if inx >= len(a.providedArgs) {
		return nil
	}
	return a.providedArgs[inx]
}

func (a *Args) Name(name string) language.Object {
	for i, k := range a.typedArgs {
		if k.Name() == name {
			if i >= len(a.providedArgs) {
				return nil
			}
			return a.providedArgs[i]
		}
	}
	return nil
}

func Function(describe *FnDescriber, fn func(*Args) (any, error)) *language.Function {
	var args = make([]language.FnArg, len(describe.args))
	for i, arg := range describe.args {
		args[i] = &language.BasicFnArg{TypeVal: arg.Type, NameVal: arg.Name, DefaultVal: arg.Default}
	}

	return language.NewTypedFunction(args, describe.returns, func(o []language.Object) (language.Object, error) {
		var dg *debug.Debug

		if len(o) > 0 {
			dg = o[0].Debug()
		}

		userArgs := &Args{
			typedArgs:    args,
			providedArgs: o,
		}

		value, err := fn(userArgs)
		if err != nil {
			return nil, err
		}

		return language.FromValue(value, dg)
	}, nil)
}
