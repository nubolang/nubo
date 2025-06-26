package http

import (
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var httpStruct *language.Struct

func NewHttp(dg *debug.Debug) language.Object {
	if httpStruct == nil {
		httpStruct = language.NewStruct("http", []language.StructField{
			{Name: "baseUrl", Type: n.TString},
		}, dg)
	}

	pkg := n.NewPackage("http", dg)
	proto := pkg.GetPrototype()

	proto.SetObject("instance", httpStruct)
	proto.SetObject("config", config(dg))
	proto.SetObject("create", n.Function(n.Describe().Returns(httpStruct.Type()), func(a *n.Args) (any, error) {
		return NewInstance(dg)
	}))

	return pkg
}
