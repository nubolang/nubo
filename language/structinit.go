package language

import "context"

func (s *StructPrototype) defaultInitializer() *Function {
	var fields []StructField
	var args []FnArg

	for _, field := range s.base.Data {
		if field.Private {
			continue
		}

		fields = append(fields, field)
		args = append(args, &BasicFnArg{
			NameVal:    field.Name,
			TypeVal:    field.Type,
			DefaultVal: s.data[field.Name],
		})
	}

	fn := NewStrictFunction(args, TypeVoid, func(ctx context.Context, values []Object) (Object, error) {
		for index, field := range fields {
			if err := s.SetObject(ctx, field.Name, values[index]); err != nil {
				return nil, err
			}
		}

		return nil, nil
	}, s.base.Debug())

	call := fn.Data
	fn.Data = func(ctx context.Context, values []Object) (Object, error) {
		prepared := append([]Object(nil), values...)
		for index, field := range fields {
			if index < len(prepared) {
				prepared[index] = structFieldValue(field.Type, prepared[index])
			}
		}

		return call(ctx, prepared)
	}

	return fn
}

func structFieldValue(typ *Type, value Object) Object {
	if typ.Base() == ObjectTypeList {
		if list, ok := value.(*List); ok && len(list.Data) == 0 {
			if list.ItemType.Compare(TypeAny) || list.ItemType.Compare(TypeVoid) {
				if typed := DefaultValue(typ); typed != nil {
					return typed
				}
			}
		}
	}

	if typ.Base() == ObjectTypeDict {
		if dict, ok := value.(*Dict); ok && dict.Data.Len() == 0 {
			if (dict.KeyType.Compare(TypeAny) || dict.KeyType.Compare(TypeVoid)) &&
				(dict.ValueType.Compare(TypeAny) || dict.ValueType.Compare(TypeVoid)) {
				if typed := DefaultValue(typ); typed != nil {
					return typed
				}
			}
		}
	}

	return value
}
