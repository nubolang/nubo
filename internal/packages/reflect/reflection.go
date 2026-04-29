package reflect

import (
	"context"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

func NewReflect(dg *debug.Debug) language.Object {
	instance := n.NewPackage("reflect", dg)
	proto := instance.GetPrototype()
	ctx := context.Background()

	structMirror := StructReflect(dg, ctx)
	proto.SetObject(ctx, "Struct", structMirror)

	proto.SetObject(ctx, "Of", native.NewTypedFunction(
		ctx,
		native.OneArg("value", language.TypeAny),
		structMirror.Type(),
		func(args native.FnCtx) (language.Object, error) {
			value, err := args.Get("value")
			if err != nil {
				return nil, err
			}
			return NewStructReflection(value, dg)
		},
	))

	proto.SetObject(ctx, "kind", native.NewTypedFunction(
		ctx,
		native.OneArg("value", language.TypeAny),
		language.TypeString,
		func(args native.FnCtx) (language.Object, error) {
			value, err := args.Get("value")
			if err != nil {
				return nil, err
			}
			return n.String(kindOf(value), value.Debug()), nil
		},
	))

	proto.SetObject(ctx, "type", native.NewTypedFunction(
		ctx,
		native.OneArg("value", language.TypeAny),
		language.TypeString,
		func(args native.FnCtx) (language.Object, error) {
			value, err := args.Get("value")
			if err != nil {
				return nil, err
			}
			return n.String(value.Type().String(), value.Debug()), nil
		},
	))

	proto.SetObject(ctx, "typeOf", native.NewTypedFunction(
		ctx,
		native.OneArg("value", language.TypeAny),
		language.TypeString,
		func(args native.FnCtx) (language.Object, error) {
			value, err := args.Get("value")
			if err != nil {
				return nil, err
			}
			return n.String(value.Type().String(), value.Debug()), nil
		},
	))

	proto.SetObject(ctx, "getType", native.NewTypedFunction(
		ctx,
		native.OneArg("value", language.TypeAny),
		language.TypeString,
		func(args native.FnCtx) (language.Object, error) {
			value, err := args.Get("value")
			if err != nil {
				return nil, err
			}
			return n.String(value.Type().String(), value.Debug()), nil
		},
	))

	return instance
}
