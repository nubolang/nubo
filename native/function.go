package native

import "github.com/nubogo/nubo/language"

type Arg struct {
	Name     string
	Type     language.ObjectType
	Optional bool
	Default  any
}

type FnCtx struct {
	typedArgs    []Arg
	providedArgs []language.Object
}

func (ctx FnCtx) Get(name string) language.Object {
	for i, arg := range ctx.typedArgs {
		if arg.Name == name {
			return ctx.providedArgs[i]
		}
	}
	return nil
}

func NewFunction(typedArgs []Arg, fn func(ctx FnCtx) (language.Object, error)) *language.Function {
	return language.NewFunction(func(args []language.Object) (language.Object, error) {
		ctx := FnCtx{
			typedArgs:    typedArgs,
			providedArgs: args,
		}

		return fn(ctx)
	}, nil)
}
