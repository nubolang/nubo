package interpreter

import (
	"context"

	"github.com/nubolang/nubo/language"
)

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

type loggedValue struct {
	ctx context.Context
	obj language.Object
}

func (v loggedValue) String() string {
	return language.Stringify(v.ctx, v.obj)
}
