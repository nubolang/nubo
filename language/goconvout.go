package language

import (
	"context"
	"fmt"
)

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
		return listToValue(v, jsonMode)

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
			keyVal, err := ToValue(key, jsonMode)
			if err != nil {
				return err
			}

			val, err := ToValue(value, jsonMode)
			if err != nil {
				return err
			}

			out[keyVal] = val
			return nil
		})
		if err != nil {
			return nil, err
		}

		return out, nil

	case *StructInstance:
		proto := obj.GetPrototype()
		ctx := StructAllowPrivateCtx(context.Background())

		fn, ok := proto.GetObject(ctx, "$convout")
		if !ok {
			return nil, fmt.Errorf("struct instance missing $convout method")
		}

		convFn, ok := fn.(*Function)
		if !ok {
			return nil, fmt.Errorf("$convout is not a function")
		}

		converted, err := convFn.Data(ctx, nil)
		if err != nil {
			return nil, err
		}

		return ToValue(converted, jsonMode)

	case *NilObj:
		return nil, nil

	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}
}

// listToValue converts a runtime list into the most specific Go slice type possible.
// In json mode it always returns []any, because JSON arrays are untyped.
func listToValue(v *List, jsonMode bool) (any, error) {
	if jsonMode {
		out := make([]any, len(v.Data))

		for i, elem := range v.Data {
			val, err := ToValue(elem, jsonMode)
			if err != nil {
				return nil, err
			}
			out[i] = val
		}

		return out, nil
	}

	switch v.ItemType {
	case TypeString:
		out := make([]string, len(v.Data))

		for i, elem := range v.Data {
			str, ok := elem.(*String)
			if !ok {
				val, err := ToValue(elem, jsonMode)
				if err != nil {
					return nil, err
				}

				casted, ok := val.(string)
				if !ok {
					return nil, fmt.Errorf("list item %d is not string", i)
				}
				out[i] = casted
				continue
			}

			out[i] = str.Data
		}

		return out, nil

	case TypeInt:
		out := make([]int64, len(v.Data))

		for i, elem := range v.Data {
			num, ok := elem.(*Int)
			if !ok {
				val, err := ToValue(elem, jsonMode)
				if err != nil {
					return nil, err
				}

				casted, ok := val.(int64)
				if !ok {
					return nil, fmt.Errorf("list item %d is not int64", i)
				}
				out[i] = casted
				continue
			}

			out[i] = num.Data
		}

		return out, nil

	case TypeFloat:
		out := make([]float64, len(v.Data))

		for i, elem := range v.Data {
			num, ok := elem.(*Float)
			if !ok {
				val, err := ToValue(elem, jsonMode)
				if err != nil {
					return nil, err
				}

				casted, ok := val.(float64)
				if !ok {
					return nil, fmt.Errorf("list item %d is not float64", i)
				}
				out[i] = casted
				continue
			}

			out[i] = num.Data
		}

		return out, nil

	case TypeBool:
		out := make([]bool, len(v.Data))

		for i, elem := range v.Data {
			b, ok := elem.(*Bool)
			if !ok {
				val, err := ToValue(elem, jsonMode)
				if err != nil {
					return nil, err
				}

				casted, ok := val.(bool)
				if !ok {
					return nil, fmt.Errorf("list item %d is not bool", i)
				}
				out[i] = casted
				continue
			}

			out[i] = b.Data
		}

		return out, nil

	default:
		out := make([]any, len(v.Data))

		for i, elem := range v.Data {
			val, err := ToValue(elem, jsonMode)
			if err != nil {
				return nil, err
			}
			out[i] = val
		}

		return out, nil
	}
}
