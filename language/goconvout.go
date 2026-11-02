package language

import (
	"context"
	"fmt"
)

func ToValue(obj Object, json ...bool) (any, error) {
	return ToValueContext(context.Background(), obj, json...)
}

func ToValueContext(ctx context.Context, obj Object, json ...bool) (any, error) {
	jsonMode := len(json) > 0 && json[0]

	ctx, leave, err := enterTraversal(ctx, obj, valueTraversal)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %T: %w", obj, err)
	}
	defer leave()

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
		return listToValue(ctx, v, jsonMode)

	case *Dict:
		if (v.KeyType != nil && v.KeyType.Base() == ObjectTypeString && v.KeyType.Next == nil) || jsonMode {
			out := make(map[string]any)

			for _, item := range v.traversalItems() {
				val, err := ToValueContext(ctx, item.value, jsonMode)
				if err != nil {
					return nil, err
				}

				out[Stringify(ctx, item.key)] = val
			}

			return out, nil
		}

		out := make(map[any]any)

		for _, item := range v.traversalItems() {
			keyVal, err := ToValueContext(ctx, item.key, jsonMode)
			if err != nil {
				return nil, err
			}

			val, err := ToValueContext(ctx, item.value, jsonMode)
			if err != nil {
				return nil, err
			}

			out[keyVal] = val
		}

		return out, nil

	case *StructInstance:
		proto := obj.GetPrototype()
		ctx = StructAllowPrivateCtx(ctx)

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

		return ToValueContext(ctx, converted, jsonMode)

	case *NilObj:
		return nil, nil

	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}
}

// listToValue converts a runtime list into the most specific Go slice type possible.
// In json mode it always returns []any, because JSON arrays are untyped.
func listToValue(ctx context.Context, v *List, jsonMode bool) (any, error) {
	if jsonMode {
		out := make([]any, len(v.Data))

		for i, elem := range v.Data {
			val, err := ToValueContext(ctx, elem, jsonMode)
			if err != nil {
				return nil, err
			}
			out[i] = val
		}

		return out, nil
	}

	itemKind := ObjectTypeAny
	if v.ItemType != nil && v.ItemType.Next == nil {
		itemKind = v.ItemType.Base()
	}

	switch itemKind {
	case ObjectTypeString:
		out := make([]string, len(v.Data))

		for i, elem := range v.Data {
			str, ok := elem.(*String)
			if !ok {
				val, err := ToValueContext(ctx, elem, jsonMode)
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

	case ObjectTypeInt:
		out := make([]int64, len(v.Data))

		for i, elem := range v.Data {
			num, ok := elem.(*Int)
			if !ok {
				val, err := ToValueContext(ctx, elem, jsonMode)
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

	case ObjectTypeFloat:
		out := make([]float64, len(v.Data))

		for i, elem := range v.Data {
			num, ok := elem.(*Float)
			if !ok {
				val, err := ToValueContext(ctx, elem, jsonMode)
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

	case ObjectTypeBool:
		out := make([]bool, len(v.Data))

		for i, elem := range v.Data {
			b, ok := elem.(*Bool)
			if !ok {
				val, err := ToValueContext(ctx, elem, jsonMode)
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
			val, err := ToValueContext(ctx, elem, jsonMode)
			if err != nil {
				return nil, err
			}
			out[i] = val
		}

		return out, nil
	}
}
