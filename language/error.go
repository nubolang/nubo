package language

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
)

const maxErrorDepth = 64

var errorDefinitions = sync.OnceValues(newErrorDefinitions)

func ErrorStruct() *Struct {
	definition, _ := errorDefinitions()
	return definition
}

func SourceLocationStruct() *Struct {
	_, definition := errorDefinitions()
	return definition
}

func newErrorDefinitions() (*Struct, *Struct) {
	ctx := context.Background()

	location := NewStruct("SourceLocation", []StructField{
		{Name: "file", Type: TypeString},
		{Name: "line", Type: TypeInt},
		{Name: "column", Type: TypeInt},
		{Name: "columnEnd", Type: TypeInt},
	}, nil)

	definition := NewStructBetter("Error", nil)
	definition.DefineFieldset([]StructField{
		{Name: "message", Type: TypeString},
		{Name: "base", Type: Nullable(definition.Type())},
		{Name: "metaData", Type: Nullable(location.Type())},
		{Name: "stackTrace", Type: NewListType(location.Type())},
	})

	location.GetPrototype().SetObject(ctx, "init", NewTypedFunction([]FnArg{
		&BasicFnArg{NameVal: "self", TypeVal: location.Type()},
		&BasicFnArg{NameVal: "file", TypeVal: TypeString},
		&BasicFnArg{NameVal: "line", TypeVal: TypeInt},
		&BasicFnArg{NameVal: "column", TypeVal: TypeInt, DefaultVal: NewInt(0, nil)},
		&BasicFnArg{NameVal: "columnEnd", TypeVal: TypeInt, DefaultVal: NewInt(0, nil)},
	}, TypeVoid, func(ctx context.Context, args []Object) (Object, error) {
		instance := args[0].(*StructInstance)

		for index, name := range []string{"file", "line", "column", "columnEnd"} {
			if err := instance.GetPrototype().SetObject(ctx, name, args[index+1]); err != nil {
				return nil, err
			}
		}

		return nil, nil
	}, nil))

	location.GetPrototype().SetObject(ctx, "__string__", NewTypedFunction([]FnArg{
		&BasicFnArg{NameVal: "self", TypeVal: location.Type()},
	}, TypeString, func(ctx context.Context, args []Object) (Object, error) {
		return NewString(formatSourceLocation(ctx, args[0].(*StructInstance)), nil), nil
	}, nil))

	definition.GetPrototype().SetObject(ctx, "init", NewTypedFunction([]FnArg{
		&BasicFnArg{NameVal: "self", TypeVal: definition.Type()},
		&BasicFnArg{NameVal: "message", TypeVal: TypeString},
		&BasicFnArg{NameVal: "base", TypeVal: Nullable(definition.Type()), DefaultVal: Nil},
		&BasicFnArg{NameVal: "metaData", TypeVal: Nullable(location.Type()), DefaultVal: Nil},
		&BasicFnArg{NameVal: "stackTrace", TypeVal: NewListType(location.Type()), DefaultVal: Nil},
	}, TypeVoid, func(ctx context.Context, args []Object) (Object, error) {
		var frames []Object

		if list, ok := errorObject(args[4]).(*List); ok {
			next := list.Iterator()
			for {
				_, frame, ok := next()
				if !ok {
					break
				}
				frames = append(frames, frame)
			}
		}

		values := []Object{args[1], args[2], args[3], NewList(frames, location.Type(), nil)}
		return nil, setErrorFields(ctx, args[0].(*StructInstance), values)
	}, nil))

	definition.GetPrototype().SetObject(ctx, "__string__", NewTypedFunction([]FnArg{
		&BasicFnArg{NameVal: "self", TypeVal: definition.Type()},
	}, TypeString, func(ctx context.Context, args []Object) (Object, error) {
		return NewString(formatError(ctx, args[0].(*StructInstance)), nil), nil
	}, nil))

	return definition, location
}

func NewSourceLocation(source *debug.Debug) (*StructInstance, error) {
	if source == nil {
		return nil, nil
	}

	instance, err := SourceLocationStruct().NewInstance()
	if err != nil {
		return nil, err
	}
	instance.debug = source

	ctx := context.Background()
	values := []Object{
		NewString(source.File, source),
		NewInt(int64(source.Line), source),
		NewInt(int64(source.Column), source),
		NewInt(int64(source.ColumnEnd), source),
	}

	for index, name := range []string{"file", "line", "column", "columnEnd"} {
		if err := instance.GetPrototype().SetObject(ctx, name, values[index]); err != nil {
			return nil, err
		}
	}

	return instance, nil
}

func NewError(message string, base *StructInstance, source *debug.Debug, trace []*debug.Debug) (*StructInstance, error) {
	instance, err := ErrorStruct().NewInstance()
	if err != nil {
		return nil, err
	}
	instance.debug = source

	var cause Object = Nil
	if base != nil {
		cause = base
	}

	var metadata Object = Nil
	location, err := NewSourceLocation(source)
	if err != nil {
		return nil, err
	}
	if location != nil {
		metadata = location
	}

	frames := make([]Object, 0, len(trace))
	for _, frame := range trace {
		location, err := NewSourceLocation(frame)
		if err != nil {
			return nil, err
		}
		if location != nil {
			frames = append(frames, location)
		}
	}

	err = setErrorFields(context.Background(), instance, []Object{
		NewString(message, source),
		cause,
		metadata,
		NewList(frames, SourceLocationStruct().Type(), source),
	})
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func setErrorFields(ctx context.Context, instance *StructInstance, values []Object) error {
	for index, name := range []string{"message", "base", "metaData", "stackTrace"} {
		if err := instance.GetPrototype().SetObject(ctx, name, values[index]); err != nil {
			return err
		}
	}

	return nil
}

func ErrorFrom(err error, fallback *debug.Debug) (*StructInstance, error) {
	return errorFrom(err, fallback, make(map[error]*StructInstance), 0)
}

func errorFrom(err error, fallback *debug.Debug, seen map[error]*StructInstance, depth int) (*StructInstance, error) {
	if err == nil {
		return nil, nil
	}
	if depth >= maxErrorDepth {
		return NewError("<error chain limit>", nil, nil, nil)
	}

	value := reflect.ValueOf(err)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return NewError("<nil error>", nil, fallback, nil)
	}

	comparable := value.Comparable()
	if comparable {
		if instance := seen[err]; instance != nil {
			return instance, nil
		}
	}

	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) > 0 {
			primary, buildErr := errorFrom(children[0], fallback, seen, depth+1)
			if buildErr != nil {
				return nil, buildErr
			}
			if comparable {
				seen[err] = primary
			}

			for _, child := range children[1:] {
				additional, buildErr := errorFrom(child, nil, seen, depth+1)
				if buildErr != nil {
					return nil, buildErr
				}
				if additional == nil {
					continue
				}
				if primary == nil {
					primary = additional
					continue
				}

				ctx := context.Background()
				message := errorField(ctx, primary, "message").Value().(string)
				message += "\nAdditional error: " + formatError(ctx, additional)
				if buildErr := primary.GetPrototype().SetObject(ctx, "message", NewString(message, primary.Debug())); buildErr != nil {
					return nil, buildErr
				}
			}

			return primary, nil
		}
	}

	var (
		message string
		trace   []*debug.Debug
		cause   error
	)
	source := fallback

	if caught, ok := err.(*exception.Expection); ok {
		details := caught.Details()
		message = details.Message
		trace = details.StackTrace
		cause = details.Cause
		if details.Debug != nil {
			source = details.Debug
		}
	} else {
		message = err.Error()
		cause = errors.Unwrap(err)
	}

	instance, buildErr := NewError(message, nil, source, trace)
	if buildErr != nil {
		return nil, buildErr
	}
	if comparable {
		seen[err] = instance
	}

	base, buildErr := errorFrom(cause, nil, seen, depth+1)
	if buildErr != nil {
		return nil, buildErr
	}
	if base != nil {
		if buildErr := instance.GetPrototype().SetObject(context.Background(), "base", base); buildErr != nil {
			return nil, buildErr
		}
	}

	return instance, nil
}

func errorObject(value Object) Object {
	for range maxErrorDepth {
		switch wrapped := value.(type) {
		case *Ref:
			value = wrapped.Data
		case *Any:
			value = wrapped.Data
		default:
			return value
		}
	}

	return Nil
}

func errorField(ctx context.Context, instance *StructInstance, name string) Object {
	value, _ := instance.GetPrototype().GetObject(ctx, name)
	return errorObject(value)
}

func formatSourceLocation(ctx context.Context, instance *StructInstance) string {
	file := errorField(ctx, instance, "file").Value().(string)
	line := errorField(ctx, instance, "line").Value().(int64)
	column := errorField(ctx, instance, "column").Value().(int64)
	end := errorField(ctx, instance, "columnEnd").Value().(int64)

	var result strings.Builder
	result.WriteString(file)
	if line > 0 {
		fmt.Fprintf(&result, ":%d", line)
	}
	if column > 0 {
		fmt.Fprintf(&result, ":%d", column)
		if end > column {
			fmt.Fprintf(&result, "-%d", end)
		}
	}

	return result.String()
}

func formatError(ctx context.Context, instance *StructInstance) string {
	var result strings.Builder
	seen := make(map[*StructInstance]bool)

	for instance != nil {
		if seen[instance] {
			result.WriteString("\nCaused by: <circular Error>")
			break
		}
		if len(seen) >= maxErrorDepth {
			result.WriteString("\nCaused by: <error chain limit>")
			break
		}

		if len(seen) > 0 {
			result.WriteString("\nCaused by: ")
		}
		seen[instance] = true

		result.WriteString("Error: ")
		result.WriteString(errorField(ctx, instance, "message").Value().(string))

		if location, ok := errorField(ctx, instance, "metaData").(*StructInstance); ok {
			result.WriteString("\n  at ")
			result.WriteString(formatSourceLocation(ctx, location))
		}

		if frames, ok := errorField(ctx, instance, "stackTrace").(*List); ok {
			next := frames.Iterator()
			for {
				_, frame, ok := next()
				if !ok {
					break
				}
				if location, ok := errorObject(frame).(*StructInstance); ok {
					result.WriteString("\n  at ")
					result.WriteString(formatSourceLocation(ctx, location))
				}
			}
		}

		instance, _ = errorField(ctx, instance, "base").(*StructInstance)
	}

	return result.String()
}
