package interpreter

import "github.com/nubolang/nubo/language"

// logObjectType returns a human readable type string without
// dereferencing nil language.Object values.
func logObjectType(obj language.Object) string {
	if obj == nil {
		return language.TypeVoid.String()
	}
	typ := obj.Type()
	if typ == nil {
		return "<unknown>"
	}
	return typ.String()
}

// logObjectString returns a printable representation of a language.Object
// and gracefully handles nil values.
func logObjectString(obj language.Object) string {
	if obj == nil {
		return "<nil>"
	}
	return obj.String()
}
