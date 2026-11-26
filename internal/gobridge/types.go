package gobridge

import (
	"fmt"
	"reflect"

	"github.com/nubolang/nubo/language"
)

const maxDepth = 256

var listType = reflect.TypeFor[[]any]()
var stringMapType = reflect.TypeFor[map[string]any]()
var mapType = reflect.TypeFor[map[any]any]()
var goHandleType = reflect.TypeFor[*language.GoHandle]()

func valueType(typ reflect.Type, active map[reflect.Type]bool) (*language.Type, error) {
	if typ == goHandleType {
		return language.TypeGoHandle, nil
	}

	if active[typ] {
		return language.TypeAny, nil
	}

	active[typ] = true
	defer delete(active, typ)

	switch typ.Kind() {
	case reflect.Interface:
		if typ.NumMethod() == 0 {
			return language.TypeAny, nil
		}

	case reflect.Slice, reflect.Array:
		element, err := valueType(typ.Elem(), active)
		if err != nil {
			return nil, err
		}

		return language.NewListType(element), nil

	case reflect.Map:
		key, err := valueType(typ.Key(), active)
		if err != nil {
			return nil, err
		}

		if !key.Base().Hashable() && key.Base() != language.ObjectTypeAny {
			return nil, fmt.Errorf("unsupported Go map key type %s", typ.Key())
		}

		value, err := valueType(typ.Elem(), active)
		if err != nil {
			return nil, err
		}

		return language.NewDictType(key, value), nil
	}

	return scalarType(typ)
}

func argumentType(typ reflect.Type) (*language.Type, error) {
	value, err := valueType(typ, make(map[reflect.Type]bool))
	if err != nil {
		return nil, err
	}

	// Check collection entries during conversion.
	switch typ.Kind() {
	case reflect.Slice, reflect.Array:
		return language.NewListType(language.TypeAny), nil
	case reflect.Map:
		return language.NewDictType(language.TypeAny, language.TypeAny), nil
	default:
		return value, nil
	}
}
