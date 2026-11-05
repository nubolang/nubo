package gobridge

import (
	"fmt"
	"reflect"

	"github.com/nubolang/nubo/language"
)

func result(value reflect.Value, depth int) (language.Object, error) {
	if depth >= maxDepth {
		return nil, fmt.Errorf("Go result nesting limit exceeded")
	}

	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return language.Nil, nil
		}

		return result(value.Elem(), depth+1)
	}

	if (value.Kind() == reflect.Slice || value.Kind() == reflect.Map) && value.IsNil() {
		return language.Nil, nil
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		elementType, err := valueType(value.Type().Elem(), make(map[reflect.Type]bool))
		if err != nil {
			return nil, err
		}

		items := make([]language.Object, value.Len())
		for i := range items {
			item, err := result(value.Index(i), depth+1)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}

			items[i] = item
		}

		return language.NewList(items, elementType, nil), nil

	case reflect.Map:
		typ, err := valueType(value.Type(), make(map[reflect.Type]bool))
		if err != nil {
			return nil, err
		}

		keys := make([]language.Object, 0, value.Len())
		values := make([]language.Object, 0, value.Len())
		seen := make(map[any]bool)

		iterator := value.MapRange()
		for iterator.Next() {
			key, err := result(iterator.Key(), depth+1)
			if err != nil {
				return nil, fmt.Errorf("map key: %w", err)
			}

			if !key.Type().Base().Hashable() {
				return nil, fmt.Errorf("Go map key cannot be represented in Nubo")
			}

			if seen[key.Value()] {
				return nil, fmt.Errorf("duplicate Nubo dictionary key after conversion")
			}
			seen[key.Value()] = true

			item, err := result(iterator.Value(), depth+1)
			if err != nil {
				return nil, fmt.Errorf("map value for %v: %w", key.Value(), err)
			}

			keys = append(keys, key)
			values = append(values, item)
		}

		return language.NewDict(keys, values, typ.Key, typ.Value, nil)

	default:
		return scalarResult(value)
	}
}
