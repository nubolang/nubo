package iter

import (
	"context"
	"errors"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var (
	iterStruct     *language.Struct
	progressStruct *language.Struct
)

func NewIter(dg *debug.Debug) language.Object {
	pkg := n.NewPackage("iter", nil)
	proto := pkg.GetPrototype()
	ctx := context.Background()

	if progressStruct == nil {
		progressStruct = language.NewStruct("Progress", []language.StructField{
			{Name: "key", Type: n.TAny},
			{Name: "value", Type: n.TAny},
			{Name: "end", Type: n.TBool},
		}, dg)

		sp := progressStruct.GetPrototype().(*language.StructPrototype)

		sp.Unlock()
		sp.SetObject(ctx, "init", n.Function(n.Describe(
			n.Arg("self", progressStruct.Type()),
			n.Arg("key", n.TAny),
			n.Arg("value", n.TAny),
		).Returns(progressStruct.Type()),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				proto := self.GetPrototype()
				proto.SetObject(ctx, "key", a.Name("key"))
				proto.SetObject(ctx, "value", a.Name("value"))
				proto.SetObject(ctx, "end", n.Bool(false, dg))
				return self, nil
			}))

		sp.Lock()
		sp.Implement()
	}

	if iterStruct == nil {
		iterStruct = language.NewStruct("Iterator", []language.StructField{
			{Name: "iterable", Type: n.TTFn(progressStruct.Type()), Private: true},
		}, dg)

		iterSp := iterStruct.GetPrototype().(*language.StructPrototype)

		iterSp.Unlock()
		iterSp.SetObject(ctx, "init", n.Function(n.Describe(
			n.Arg("self", iterStruct.Type()),
			n.Arg("iterable", n.TTFn(progressStruct.Type())),
		).Returns(iterStruct.Type()),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				proto := self.GetPrototype()
				return self, proto.SetObject(language.StructAllowPrivateCtx(ctx), "iterable", a.Name("iterable"))
			}))

		iterSp.SetObject(ctx, "next", n.Function(n.Describe(n.Arg("self", iterStruct.Type())).Returns(progressStruct.Type()), func(a *n.Args) (any, error) {
			ctx := language.StructAllowPrivateCtx(ctx)
			self := a.Name("self")
			proto := self.GetPrototype()

			it, _ := proto.GetObject(ctx, "iterable")
			itFn, ok := it.(*language.Function)

			if !ok {
				return nil, errors.New("iterable is not a function")
			}

			iter, err := itFn.Data(ctx, nil)
			if err != nil {
				return nil, err
			}
			return iter, nil
		}))

		iterSp.Lock()
		iterSp.Implement()
	}

	end, _ := progressStruct.NewInstance()
	end.GetPrototype().SetObject(ctx, "end", n.Bool(true, dg))

	proto.SetObject(ctx, "Progress", progressStruct)
	proto.SetObject(ctx, "End", end)
	proto.SetObject(ctx, "Iterator", iterStruct)

	return pkg
}
