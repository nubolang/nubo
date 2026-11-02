package interpreter

import "github.com/nubolang/nubo/language"

func propertyAccess(obj language.Object, name string) (prototype, field, setter bool) {
	for {
		ref, ok := obj.(*language.Ref)
		if !ok {
			break
		}

		obj = ref.Data
	}

	switch value := obj.(type) {
	case *language.Struct:
		return value.UserDefined(), false, false

	case *language.StructInstance:
		definition := value.Definition()
		for _, declared := range definition.Data {
			if declared.Name == name {
				field = true
				break
			}
		}

		return definition.UserDefined(), field, definition.UserDefined() || field

	case *language.Dict, *language.Element:
		return false, false, true

	default:
		return false, false, false
	}
}
