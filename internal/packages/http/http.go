package http

import (
	"context"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var httpStruct *language.Struct

func NewHttp(dg *debug.Debug) language.Object {
	ctx := context.Background()

	if httpStruct == nil {
		httpStruct = language.NewStruct("http", []language.StructField{
			{Name: "baseUrl", Type: n.TString},
		}, dg)
	}

	pkg := n.NewPackage("http", dg)
	proto := pkg.GetPrototype()

	proto.SetObject(ctx, "instance", httpStruct)
	proto.SetObject(ctx, "config", config(dg))
	proto.SetObject(ctx, "create", n.Function(n.Describe().Returns(httpStruct.Type()), func(a *n.Args) (any, error) {
		return NewInstance(dg)
	}))

	return pkg
}
