package interpreter

import "github.com/nubolang/nubo/language"

// cloneForBinding is for implicit runtime copies (var binds/arg passing).
// Struct instances must stay by reference here; explicit clone() still uses
// the object's Clone implementation.
func cloneForBinding(value language.Object) language.Object {
	if value == nil {
		return nil
	}

	if value.Type().Base() == language.ObjectTypeStructInstance {
		return value
	}

	return value.Clone()
}

func (i *Interpreter) cloneBinding(value language.Object) (language.Object, error) {
	if value != nil && value.Type().Base() == language.ObjectTypeStructInstance {
		return value, nil
	}

	return language.Clone(i.ctx, value)
}
