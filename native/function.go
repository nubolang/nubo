package native

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
)

type FnCtx struct {
	typedArgs    []language.FnArg
	providedArgs []language.Object
}

type FunctionWrapper func(ctx FnCtx) (language.Object, error)

func (ctx FnCtx) Get(name string) (language.Object, error) {
	for i, arg := range ctx.typedArgs {
		if arg.Name() == name {
			return ctx.providedArgs[i], nil
		}
	}
	return nil, debug.NewError(fmt.Errorf("Undefined function argument"), name, nil)
}

func NewFunction(fn func(args []language.Object) (language.Object, error)) *language.Function {
	return language.NewFunction(fn, nil)
}

func NewTypedFunction(typedArgs []language.FnArg, returnType *language.Type, fn FunctionWrapper) *language.Function {
	return language.NewTypedFunction(typedArgs, returnType, func(args []language.Object) (language.Object, error) {
		ctx := FnCtx{
			typedArgs:    typedArgs,
			providedArgs: args,
		}

		return fn(ctx)
	}, nil)
}
