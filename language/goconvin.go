package language

import (
	"fmt"
	"reflect"
	"time"

	"github.com/nubolang/nubo/internal/debug"
)

func FromValue(data any, voidNil bool, dg ...*debug.Debug) (Object, error) {
	var dbg *debug.Debug
	if len(dg) > 0 {
		dbg = dg[0]
	}

	switch value := data.(type) {
	case Object:
		return value, nil

	case map[string]any:
		var (
			keys []Object
			vals []Object
		)

		for k, v := range value {
			key, err := FromValue(k, voidNil, dbg)
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)

			val, err := FromValue(v, voidNil, dbg)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}

		return NewDict(keys, vals, TypeString, TypeAny, dbg)

	case map[any]any:
		var (
			keys []Object
			vals []Object
		)

		for k, v := range value {
			key, err := FromValue(k, voidNil, dbg)
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)

			val, err := FromValue(v, voidNil, dbg)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}

		return NewDict(keys, vals, TypeAny, TypeAny, dbg)

	case []any:
		return fromSlice(value, voidNil, dbg)

	case []string:
		return fromSlice(value, voidNil, dbg)

	case []int:
		return fromSlice(value, voidNil, dbg)

	case []int8:
		return fromSlice(value, voidNil, dbg)

	case []int16:
		return fromSlice(value, voidNil, dbg)

	case []int32:
		return fromSlice(value, voidNil, dbg)

	case []int64:
		return fromSlice(value, voidNil, dbg)

	case []uint:
		return fromSlice(value, voidNil, dbg)

	case []uint8:
		return fromSlice(value, voidNil, dbg)

	case []uint16:
		return fromSlice(value, voidNil, dbg)

	case []uint32:
		return fromSlice(value, voidNil, dbg)

	case []uint64:
		return fromSlice(value, voidNil, dbg)

	case []float32:
		return fromSlice(value, voidNil, dbg)

	case []float64:
		return fromSlice(value, voidNil, dbg)

	case []bool:
		return fromSlice(value, voidNil, dbg)

	case []time.Time:
		return fromSlice(value, voidNil, dbg)

	case int:
		return NewInt(int64(value), dbg), nil
	case int8:
		return NewInt(int64(value), dbg), nil
	case int16:
		return NewInt(int64(value), dbg), nil
	case int32:
		return NewInt(int64(value), dbg), nil
	case int64:
		return NewInt(value, dbg), nil

	case float32:
		return NewFloat(float64(value), dbg), nil
	case float64:
		return NewFloat(value, dbg), nil

	case uint:
		return NewInt(int64(value), dbg), nil
	case uint8:
		return NewInt(int64(value), dbg), nil
	case uint16:
		return NewInt(int64(value), dbg), nil
	case uint32:
		return NewInt(int64(value), dbg), nil
	case uint64:
		return NewInt(int64(value), dbg), nil
	case uintptr:
		return NewInt(int64(value), dbg), nil

	case string:
		return NewString(value, dbg), nil
	case bool:
		return NewBool(value, dbg), nil

	case nil:
		if voidNil {
			return nil, nil
		}
		return Nil, nil

	case time.Time:
		return NewString(value.Format("2006-01-02T15:04:05-07:00"), dbg), nil
	}

	// Fallback for any slice/array type not covered above.
	rv := reflect.ValueOf(data)
	if rv.IsValid() {
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			return fromReflectSlice(rv, voidNil, dbg)
		}
	}

	return nil, fmt.Errorf("unsupported type %T", data)
}

// fromSlice converts a typed Go slice into a runtime list.
func fromSlice[T any](src []T, voidNil bool, dbg *debug.Debug) (Object, error) {
	items := make([]Object, len(src))

	for i, val := range src {
		obj, err := FromValue(val, voidNil, dbg)
		if err != nil {
			return nil, err
		}
		items[i] = obj
	}

	return NewList(items, inferListType(items), dbg), nil
}

// fromReflectSlice converts any slice/array via reflection.
func fromReflectSlice(rv reflect.Value, voidNil bool, dbg *debug.Debug) (Object, error) {
	items := make([]Object, rv.Len())

	for i := range rv.Len() {
		obj, err := FromValue(rv.Index(i).Interface(), voidNil, dbg)
		if err != nil {
			return nil, err
		}
		items[i] = obj
	}

	return NewList(items, inferListType(items), dbg), nil
}

// inferListType returns the common item type, or TypeAny if mixed.
func inferListType(items []Object) *Type {
	if len(items) == 0 {
		return TypeAny
	}

	var first *Type
	found := false

	for _, item := range items {
		if item == nil {
			return TypeAny
		}

		t := item.Type()
		if !found {
			first = t
			found = true
			continue
		}

		if t != first {
			return TypeAny
		}
	}

	return first
}
