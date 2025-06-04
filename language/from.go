package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

func FromValue(data any, dg ...*debug.Debug) (Object, error) {
	var dbg *debug.Debug
	if len(dg) > 0 {
		dbg = dg[0]
	}

	switch value := data.(type) {
	case Object:
		return value, nil
	case map[any]any:
		var (
			keys []Object
			vals []Object
		)

		for k, v := range value {
			key, err := FromValue(k)
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)

			val, err := FromValue(v)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}

		return NewDict(keys, vals, TypeAny, TypeAny, dbg)
	case []any:
		var li = make([]Object, len(value))
		for i, val := range value {
			value, err := FromValue(val)
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
	case string:
		return NewString(value, dbg), nil
	case bool:
		return NewBool(value, dbg), nil
	case nil:
		return NewNil(), nil
	}
	return nil, fmt.Errorf("unsupported type %T", data)
}
