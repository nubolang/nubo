package thread

import (
	"fmt"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"go.uber.org/zap"
)

func NewThread() language.Object {
	instance := language.NewStruct("@std/thread", nil, nil)
	proto := instance.GetPrototype()

	proto.SetObject("spawn", native.NewFunction(func(args []language.Object) (language.Object, error) {
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

		go func(fn func(args []language.Object) (language.Object, error), fnArgs []language.Object) {
			_, err := fn(fnArgs)
			if err != nil {
				zap.L().Error("@std/thread.spawn: executing a function failed", zap.Error(err))
			}
		}(fn, fnArgs)

		return nil, nil
	}))

	return instance
}
