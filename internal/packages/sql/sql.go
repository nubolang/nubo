package sql

import (
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func NewSQL() language.Object {
	obj := language.NewStruct("@std/sql", nil, nil)
	inst := obj.GetPrototype()

	inst.SetObject("open", n.Function(n.Describe(n.Arg("provider", n.TStruct)).Returns(n.TStruct), func(args *n.Args) (any, error) {
		rawProvider := args.Name("provider")
		provider, ok := rawProvider.(*SQLConn)
		if !ok {
			return nil, n.Err("provider should be a SQLConn type that holds the database connection state", provider.Debug())
		}

		return NewDB(provider)
	}))

	return obj
}
