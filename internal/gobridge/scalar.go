package gobridge

import (
	"fmt"
	"math"
	"reflect"

	"github.com/nubolang/nubo/language"
)

func scalarType(typ reflect.Type) (*language.Type, error) {
	switch typ.Kind() {
	case reflect.Bool:
		return language.TypeBool, nil
	case reflect.String:
		return language.TypeString, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return language.TypeInt, nil
	case reflect.Float32, reflect.Float64:
		return language.TypeFloat, nil
	default:
		return nil, fmt.Errorf("unsupported Go type %s", typ)
	}
}

func integerValue(obj language.Object) (int64, bool) {
	switch value := obj.(type) {
	case *language.Int:
		return value.Data, true
	case *language.Byte:
		return int64(value.Data), true
	case *language.Char:
		return int64(value.Data), true
	default:
		return 0, false
	}
}

func scalarValue(obj language.Object, typ reflect.Type) (reflect.Value, error) {
	converted := reflect.New(typ).Elem()

	switch typ.Kind() {
	case reflect.Bool:
		value, ok := obj.(*language.Bool)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected bool")
		}

		converted.SetBool(value.Data)

	case reflect.String:
		value, ok := obj.(*language.String)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected string")
		}

		converted.SetString(value.Data)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, ok := integerValue(obj)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected int")
		}

		if converted.OverflowInt(value) {
			return reflect.Value{}, fmt.Errorf("integer %d overflows %s", value, typ)
		}

		converted.SetInt(value)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, ok := integerValue(obj)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected int")
		}

		if value < 0 || converted.OverflowUint(uint64(value)) {
			return reflect.Value{}, fmt.Errorf("integer %d overflows %s", value, typ)
		}

		converted.SetUint(uint64(value))

	case reflect.Float32, reflect.Float64:
		value, ok := obj.(*language.Float)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected float")
		}

		if converted.OverflowFloat(value.Data) {
			return reflect.Value{}, fmt.Errorf("float %g overflows %s", value.Data, typ)
		}

		converted.SetFloat(value.Data)

	default:
		return reflect.Value{}, fmt.Errorf("unsupported Go type %s", typ)
	}

	return converted, nil
}

func scalarResult(value reflect.Value) (language.Object, error) {
	switch value.Kind() {
	case reflect.Bool:
		return language.NewBool(value.Bool(), nil), nil
	case reflect.String:
		return language.NewString(value.String(), nil), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return language.NewInt(value.Int(), nil), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value.Uint() > math.MaxInt64 {
			return nil, fmt.Errorf("Go result %d overflows Nubo int", value.Uint())
		}

		return language.NewInt(int64(value.Uint()), nil), nil
	case reflect.Float32, reflect.Float64:
		return language.NewFloat(value.Float(), nil), nil
	default:
		return nil, fmt.Errorf("unsupported Go result type %s", value.Type())
	}
}
