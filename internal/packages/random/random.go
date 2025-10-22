package random

import (
	"context"
	"math/rand"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

func NewRandom(dg *debug.Debug) language.Object {
	instance := n.NewPackage("random", dg)
	proto := instance.GetPrototype()

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	ctx := context.Background()

	proto.SetObject(ctx, "between", native.NewTypedFunction(ctx,
		[]language.FnArg{
			&language.BasicFnArg{TypeVal: language.TypeInt, NameVal: "min"},
			&language.BasicFnArg{TypeVal: language.TypeInt, NameVal: "max"},
		},
		language.TypeInt,
		func(ctx native.FnCtx) (language.Object, error) {
			minObj, _ := ctx.Get("min")
			maxObj, _ := ctx.Get("max")
			min := minObj.Value().(int64)
			max := maxObj.Value().(int64)

			if min > max {
				min, max = max, min
			}
			val := random.Int63n(max-min+1) + min

			return language.NewInt(val, minObj.Debug()), nil
		},
	))

	proto.SetObject(ctx, "float", native.NewTypedFunction(ctx,
		nil,
		language.TypeFloat,
		func(ctx native.FnCtx) (language.Object, error) {
			return language.NewFloat(random.Float64(), nil), nil
		},
	))

	proto.SetObject(ctx, "bool", native.NewTypedFunction(ctx,
		nil,
		language.TypeBool,
		func(ctx native.FnCtx) (language.Object, error) {
			return language.NewBool(random.Intn(2) == 0, nil), nil
		},
	))

	proto.SetObject(ctx, "choice", native.NewTypedFunction(ctx,
		[]language.FnArg{
			&language.BasicFnArg{TypeVal: language.TypeList, NameVal: "list"},
		},
		language.TypeAny,
		func(ctx native.FnCtx) (language.Object, error) {
			arr, _ := ctx.Get("list")
			list := arr.Value().([]language.Object)
			if len(list) == 0 {
				return language.Nil, nil
			}
			i := random.Intn(len(list))
			return list[i], nil
		},
	))

	proto.SetObject(ctx, "seed", native.NewTypedFunction(ctx,
		[]language.FnArg{
			&language.BasicFnArg{TypeVal: language.TypeInt, NameVal: "value"},
		},
		language.TypeVoid,
		func(ctx native.FnCtx) (language.Object, error) {
			val, _ := ctx.Get("value")
			source := rand.NewSource(val.Value().(int64))
			random = rand.New(source)
			return nil, nil
		},
	))

	return instance
}
