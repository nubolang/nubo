package component

import (
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var contextStruct *language.Struct

func NewComponent(dg *debug.Debug) language.Object {
	pkg := n.NewPackage("component", nil)
	proto := pkg.GetPrototype()

	if contextStruct == nil {
		contextStruct = language.NewStruct("Context", []language.StructField{
			{Name: "props", Type: n.NewDictType(n.TString, n.TAny)},
			{Name: "children", Type: n.TTList(n.TUnion(n.TString, n.THtml))},
		}, dg)

		sp := contextStruct.GetPrototype().(*language.StructPrototype)

		sp.Unlock()
		sp.SetObject("init", n.Function(n.Describe(
			n.Arg("self", contextStruct.Type()),
			n.Arg("props", n.NewDictType(n.TString, n.TAny)),
			n.Arg("children", n.TTList(n.TUnion(n.TString, n.THtml))),
		).Returns(contextStruct.Type()),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				proto := self.GetPrototype()
				proto.SetObject("props", a.Name("props"))
				proto.SetObject("children", a.Name("children"))
				return self, nil
			}))

		sp.Lock()
		sp.Implement()
	}

	proto.SetObject("Context", contextStruct)

	return pkg
}
