package gobridge

import (
	"context"
	"fmt"
	"reflect"

	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
)

var contextType = reflect.TypeFor[context.Context]()
var errorType = reflect.TypeFor[error]()

func Wrap(fn any) (*language.Function, error) {
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func || value.IsNil() {
		return nil, fmt.Errorf("expected a non-nil Go function")
	}

	typ := value.Type()
	if typ.IsVariadic() {
		return nil, fmt.Errorf("variadic Go functions are not supported yet")
	}

	start := 0
	if typ.NumIn() > 0 && typ.In(0) == contextType {
		start = 1
	}

	args := make([]language.FnArg, typ.NumIn()-start)
	for i := range args {
		argType, err := argumentType(typ.In(i + start))
		if err != nil {
			return nil, fmt.Errorf("argument %d: %w", i+1, err)
		}

		args[i] = &language.BasicFnArg{NameVal: fmt.Sprintf("arg%d", i+1), TypeVal: argType}
	}

	returns := typ.NumOut()
	hasError := returns > 0 && typ.Out(returns-1) == errorType
	if hasError {
		returns--
	}

	if returns > 1 {
		return nil, fmt.Errorf("Go functions may return one value and an optional error")
	}

	returnType := language.TypeVoid
	if returns == 1 {
		var err error
		returnType, err = valueType(typ.Out(0), make(map[reflect.Type]bool))
		if err != nil {
			return nil, fmt.Errorf("return value: %w", err)
		}
	}

	return language.NewStrictFunction(args, returnType, func(ctx context.Context, args []language.Object) (result language.Object, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				result = nil
				if cause, ok := recovered.(error); ok {
					err = fmt.Errorf("Go function panicked: %w", cause)
				} else {
					err = fmt.Errorf("Go function panicked: %v", recovered)
				}
			}

			if err != nil {
				result = nil
				err = exception.From(err, nil).WithLevel(exception.LevelRuntime)
			}
		}()

		input := make([]reflect.Value, typ.NumIn())
		if start == 1 {
			if ctx == nil {
				ctx = context.Background()
			}

			input[0] = reflect.ValueOf(ctx)
		}

		for i, arg := range args {
			converted, err := argument(arg, typ.In(i+start), make(map[any]bool), 0)
			if err != nil {
				return nil, fmt.Errorf("argument %d: %w", i+1, err)
			}

			input[i+start] = converted
		}

		output := value.Call(input)
		if hasError && !output[len(output)-1].IsNil() {
			return nil, output[len(output)-1].Interface().(error)
		}

		if returns == 0 {
			return nil, nil
		}

		return resultValue(output[0])
	}, nil), nil
}

func resultValue(value reflect.Value) (language.Object, error) {
	return result(value, 0)
}
