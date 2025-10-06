package language

import (
	"fmt"
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
		var li = make([]Object, len(value))
		for i, val := range value {
			value, err := FromValue(val, voidNil, dbg)
			if err != nil {
				return nil, err
			}

			li[i] = value
		}
		return NewList(li, TypeAny, dbg), nil
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

	return nil, fmt.Errorf("unsupported type %T", data)
}

func ToValue(obj Object, json ...bool) (any, error) {
	jsonMode := len(json) > 0 && json[0]

	switch v := obj.(type) {
	case *Int:
		return v.Data, nil
	case *Float:
		return v.Data, nil
	case *String:
		return v.Data, nil
	case *Bool:
		return v.Data, nil
	case *List:
		out := make([]any, len(v.Data))
		for i, elem := range v.Data {
			val, err := ToValue(elem, jsonMode)
			if err != nil {
				return nil, err
			}
			out[i] = val
		}
		return out, nil
	case *Dict:
		if v.KeyType == TypeString || jsonMode {
			out := make(map[string]any)
			err := v.Data.IterateErr(func(key Object, value Object) error {
				val, err := ToValue(value, jsonMode)
				if err != nil {
					return err
				}
				out[key.String()] = val
				return nil
			})
			if err != nil {
				return nil, err
			}
			return out, nil
		}

		out := make(map[any]any)
		err := v.Data.IterateErr(func(key Object, value Object) error {
			val, err := ToValue(value, jsonMode)
			if err != nil {
				return err
			}
			out[key.Value()] = val
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	default:
		if val, ok := obj.(Object); ok {
			return ToValue(val, jsonMode)
		}
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}
}
