package gobridge

import (
	"fmt"
	"reflect"

	"github.com/nubolang/nubo/language"
)

func argument(obj language.Object, typ reflect.Type, active map[any]bool, depth int) (reflect.Value, error) {
	if depth >= maxDepth {
		return reflect.Value{}, fmt.Errorf("Go argument nesting limit exceeded")
	}

	var identity any
	switch value := obj.(type) {
	case *language.Ref, *language.Any, *language.List:
		identity = obj
	case *language.Dict:
		identity = value.Data
	}

	if typ.Kind() != reflect.Interface && identity != nil {
		if active[identity] {
			return reflect.Value{}, fmt.Errorf("cyclic Nubo value in Go argument")
		}

		active[identity] = true
		defer delete(active, identity)
	}

	switch value := obj.(type) {
	case nil:
		return reflect.Value{}, fmt.Errorf("cannot pass void to Go")
	case *language.Ref:
		return argument(value.Data, typ, active, depth+1)
	case *language.Any:
		return argument(value.Data, typ, active, depth+1)
	case *language.NilObj:
		if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map || (typ.Kind() == reflect.Interface && typ.NumMethod() == 0) {
			return reflect.Zero(typ), nil
		}

		return reflect.Value{}, fmt.Errorf("nil is not valid for %s", typ)
	}

	switch typ.Kind() {
	case reflect.Interface:
		if typ.NumMethod() != 0 {
			return reflect.Value{}, fmt.Errorf("unsupported Go type %s", typ)
		}

		concrete := dynamicType(obj)
		if concrete == nil {
			return reflect.Value{}, fmt.Errorf("unsupported Nubo value %T", obj)
		}

		value, err := argument(obj, concrete, active, depth+1)
		if err != nil {
			return reflect.Value{}, err
		}

		converted := reflect.New(typ).Elem()
		converted.Set(value)
		return converted, nil

	case reflect.Slice, reflect.Array:
		list, ok := obj.(*language.List)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected a list for %s", typ)
		}

		var items []language.Object
		next := list.Iterator()
		for {
			_, item, ok := next()
			if !ok {
				break
			}

			items = append(items, item)
		}

		if typ.Kind() == reflect.Array && typ.Len() != len(items) {
			return reflect.Value{}, fmt.Errorf("expected %d array entries, got %d", typ.Len(), len(items))
		}

		converted := reflect.New(typ).Elem()
		if typ.Kind() == reflect.Slice {
			converted = reflect.MakeSlice(typ, len(items), len(items))
		}

		for i, item := range items {
			value, err := argument(item, typ.Elem(), active, depth+1)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("element %d: %w", i, err)
			}

			converted.Index(i).Set(value)
		}

		return converted, nil

	case reflect.Map:
		dict, ok := obj.(*language.Dict)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected a dictionary for %s", typ)
		}

		converted := reflect.MakeMapWithSize(typ, dict.Data.Len())
		err := dict.Data.IterateErr(func(key, value language.Object) error {
			convertedKey, err := argument(key, typ.Key(), active, depth+1)
			if err != nil {
				return fmt.Errorf("map key: %w", err)
			}

			if !convertedKey.Comparable() {
				return fmt.Errorf("Go map key is not comparable")
			}

			if converted.MapIndex(convertedKey).IsValid() {
				return fmt.Errorf("duplicate Go map key after conversion")
			}

			convertedValue, err := argument(value, typ.Elem(), active, depth+1)
			if err != nil {
				return fmt.Errorf("map value for %v: %w", convertedKey.Interface(), err)
			}

			converted.SetMapIndex(convertedKey, convertedValue)
			return nil
		})
		if err != nil {
			return reflect.Value{}, err
		}

		return converted, nil
	default:
		return scalarValue(obj, typ)
	}
}

func dynamicType(obj language.Object) reflect.Type {
	switch value := obj.(type) {
	case *language.Int:
		return reflect.TypeFor[int64]()
	case *language.Float:
		return reflect.TypeFor[float64]()
	case *language.String:
		return reflect.TypeFor[string]()
	case *language.Bool:
		return reflect.TypeFor[bool]()
	case *language.Byte:
		return reflect.TypeFor[uint8]()
	case *language.Char:
		return reflect.TypeFor[int32]()
	case *language.List:
		return listType
	case *language.Dict:
		if value.KeyType != nil && value.KeyType.Base() == language.ObjectTypeString && value.KeyType.Next == nil {
			return stringMapType
		}

		return mapType
	default:
		return nil
	}
}
