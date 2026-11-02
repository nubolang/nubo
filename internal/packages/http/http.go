package http

import (
	"context"
	"sync"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

var (
	clientOnce sync.Once
	clientType *language.Struct
)

func getClientType(dg *debug.Debug) *language.Struct {
	clientOnce.Do(func() {
		clientType = language.NewStruct("http", []language.StructField{
			{Name: "baseUrl", Type: n.TString},
		}, dg)

		proto := clientType.GetPrototype().(*language.StructPrototype)
		proto.Unlock()
		defer func() {
			proto.Lock()
			proto.Implement()
		}()

		ctx := context.Background()
		_ = proto.SetObject(ctx, "request", n.Function(n.Describe(
			n.Arg("self", clientType.Type()),
			n.Arg("method", n.TString),
			n.Arg("url", n.TString),
			n.Arg("options", n.TAny, language.Nil),
		).Returns(getResp(dg).Type()), func(args *n.Args) (any, error) {
			return doRequest(
				args.Context(),
				args.Name("self"),
				args.Name("method").String(),
				args.Name("url").String(),
				args.Name("options"),
			)
		}))
	})

	return clientType
}

func NewHttp(dg *debug.Debug) language.Object {
	ctx := context.Background()
	pkg := n.NewPackage("http", dg)
	proto := pkg.GetPrototype()
	kind := getClientType(dg)
	_ = proto.SetObject(ctx, "instance", kind)

	_ = proto.SetObject(ctx, "create", native.NewTypedFunction(
		ctx,
		native.OneArg("options", n.TAny, language.Nil),
		kind.Type(),
		func(args native.FnCtx) (language.Object, error) {
			options, err := args.Get("options")
			if err != nil {
				return nil, err
			}
			parsed, err := readOptions(options, true)
			if err != nil {
				return nil, err
			}
			return newInstance(dg, parsed)
		},
	))

	base, err := NewInstance(dg)
	if err == nil {
		_ = proto.SetObject(ctx, "base", base)
	}

	return pkg
}
