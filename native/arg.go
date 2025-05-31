package native

import "github.com/nubogo/nubo/language"

type Arg struct {
	NameValue    string
	TypeValue    language.ObjectComplexType
	DefaultValue language.Object
}

func NewArg(name string, typ language.ObjectComplexType, defaultValue ...language.Object) *Arg {
	var def language.Object
	if len(defaultValue) > 0 && defaultValue[0] != nil {
		def = defaultValue[0]
	}

	return &Arg{
		NameValue:    name,
		TypeValue:    typ,
		DefaultValue: def,
	}
}

func OneArg(name string, typ language.ObjectComplexType, defaultValue ...language.Object) []language.FnArg {
	var def language.Object
	if len(defaultValue) > 0 && defaultValue[0] != nil {
		def = defaultValue[0]
	}

	return []language.FnArg{
		&Arg{
			NameValue:    name,
			TypeValue:    typ,
			DefaultValue: def,
		},
	}
}

func (arg *Arg) Name() string {
	return arg.NameValue
}

func (arg *Arg) Type() language.ObjectComplexType {
	return arg.TypeValue
}

func (arg *Arg) Default() language.Object {
	return arg.DefaultValue
}
