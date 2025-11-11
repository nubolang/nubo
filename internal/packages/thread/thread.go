package thread

import (
	"context"
	"fmt"
	"runtime"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
	"go.uber.org/zap"
)

func NewThread(dg *debug.Debug) language.Object {
	instance := n.NewPackage("thread", dg)
	proto := instance.GetPrototype()

	newPortalStruct(dg)

	ctx := context.Background()
	proto.SetObject(ctx, "Portal", portalStruct)

	proto.SetObject(ctx, "spawn", native.NewFunction(func(args []language.Object) (language.Object, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("Expected at least one argument")
		}

		fnObj, ok := args[0].(*language.Function)
		if !ok {
			return nil, fmt.Errorf("First argument must be a function")
		}

		if len(args)-1 != len(fnObj.ArgTypes) {
			return nil, fmt.Errorf("Provided function expected %d arguments, got %d", len(fnObj.ArgTypes), len(args)-1)
		}

		fn := fnObj.Data
		fnArgs := args[1:]

		for i, argType := range fnObj.ArgTypes {
			if !argType.Type().Compare(fnArgs[i].Type()) {
				return nil, fmt.Errorf("Argument %d has type %s, expected %s", i+1, fnArgs[i].Type(), argType.Type())
			}
		}

		go func(fn func(ctx context.Context, args []language.Object) (language.Object, error), fnArgs []language.Object) {
			_, err := fn(ctx, fnArgs)
			if err != nil {
				zap.L().Error("@std/thread.spawn: executing a function failed", zap.Error(err))
			}
		}(fn, fnArgs)

		return nil, nil
	}))

	proto.SetObject(ctx, "yield", native.NewFunction(func(args []language.Object) (language.Object, error) {
		runtime.Gosched()
		return nil, nil
	}))

	return instance
}
