package sql

import (
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var dbStruct *language.Struct

func NewSQL(dg *debug.Debug) language.Object {
	obj := n.NewPackage("sql", dg)
	inst := obj.GetPrototype()

	if dbStruct == nil {
		dbStruct = language.NewStruct("sql.DB", nil, dg)
	}

	inst.SetObject("DB", dbStruct)

	inst.SetObject("open", n.Function(n.Describe(n.Arg("provider", n.TStruct)).Returns(dbStruct.Type()), func(args *n.Args) (any, error) {
		rawProvider := args.Name("provider")
		provider, ok := rawProvider.(*SQLConn)
		if !ok {
			return nil, n.Err("provider should be a SQLConn type that holds the database connection state", provider.Debug())
		}

		return NewDB(provider)
	}))

	return obj
}
