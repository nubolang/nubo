package math

import (
	"math"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

func NewMath(dg *debug.Debug) language.Object {
	instance := n.NewPackage("math", dg)
	proto := instance.GetPrototype()

	proto.SetObject("abs", native.NewTypedFunction(native.OneArg("number", language.TypeNumber), language.TypeNumber, absFn))
	proto.SetObject("sqrt", native.NewTypedFunction(native.OneArg("number", language.TypeNumber), language.TypeFloat, sqrtFn))
	proto.SetObject("pow", native.NewTypedFunction([]language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeNumber, NameVal: "base"},
		&language.BasicFnArg{TypeVal: language.TypeNumber, NameVal: "exp"},
	}, language.TypeFloat, powFn))
	proto.SetObject("sin", native.NewTypedFunction(native.OneArg("x", language.TypeNumber), language.TypeFloat, sinFn))
	proto.SetObject("cos", native.NewTypedFunction(native.OneArg("x", language.TypeNumber), language.TypeFloat, cosFn))

	return instance
}

func absFn(ctx native.FnCtx) (language.Object, error) {
	number, err := ctx.Get("number")
	if err != nil {
		return nil, err
	}

	switch number.(type) {
	case *language.Int:
		n := number.Value().(int64)
		if n < 0 {
			return language.NewInt(-n, number.Debug()), nil
		}
		return number, nil
	case *language.Float:
		n := number.Value().(float64)
		if n < 0 {
			return language.NewFloat(-n, number.Debug()), nil
		}
		return number, nil
	}

	return number, nil
}

func sqrtFn(ctx native.FnCtx) (language.Object, error) {
	number, err := ctx.Get("number")
	if err != nil {
		return nil, err
	}

	return language.NewFloat(math.Sqrt(toFloat(number)), number.Debug()), nil
}

func powFn(ctx native.FnCtx) (language.Object, error) {
	base, err := ctx.Get("base")
	if err != nil {
		return nil, err
	}

	exp, err := ctx.Get("exp")
	if err != nil {
		return nil, err
	}

	result := math.Pow(toFloat(base), toFloat(exp))
	return language.NewFloat(result, base.Debug()), nil
}

func sinFn(ctx native.FnCtx) (language.Object, error) {
	x, err := ctx.Get("x")
	if err != nil {
		return nil, err
	}

	return language.NewFloat(math.Sin(toFloat(x)), x.Debug()), nil
}

func cosFn(ctx native.FnCtx) (language.Object, error) {
	x, err := ctx.Get("x")
	if err != nil {
		return nil, err
	}

	return language.NewFloat(math.Cos(toFloat(x)), x.Debug()), nil
}
