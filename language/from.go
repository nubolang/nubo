package language

import "fmt"

func FromValue(data any) (Object, error) {
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

		return NewDict(keys, vals, TypeAny, TypeAny, nil)
	case []any:
		var li = make([]Object, len(value))
		for i, val := range value {
			value, err := FromValue(val)
			if err != nil {
				return nil, err
			}

			li[i] = value
		}
		return NewList(li, TypeAny, nil), nil
	case int:
		return NewInt(int64(value), nil), nil
	case int8:
		return NewInt(int64(value), nil), nil
	case int16:
		return NewInt(int64(value), nil), nil
	case int32:
		return NewInt(int64(value), nil), nil
	case int64:
		return NewInt(value, nil), nil
	case float32:
		return NewFloat(float64(value), nil), nil
	case float64:
		return NewFloat(value, nil), nil
	case string:
		return NewString(value, nil), nil
	case bool:
		return NewBool(value, nil), nil
	case nil:
		return nil, nil
	}
	return nil, fmt.Errorf("unsupported type %T", data)
}
