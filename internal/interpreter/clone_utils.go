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
