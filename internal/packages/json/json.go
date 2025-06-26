package json

import (
	"encoding/json"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

func NewJSON(dg *debug.Debug) language.Object {
	instance := n.NewPackage("json", dg)
	proto := instance.GetPrototype()

	proto.SetObject("parse", native.NewTypedFunction(native.OneArg("string", language.TypeString), language.TypeAny, parseFn))
	proto.SetObject("stringify", native.NewTypedFunction(native.OneArg("object", language.TypeAny), language.TypeString, stringifyFn))

	return instance
}

func parseFn(ctx native.FnCtx) (language.Object, error) {
	value, err := ctx.Get("string")
	if err != nil {
		return nil, err
	}

	var data any
	err = json.Unmarshal([]byte(value.Value().(string)), &data)
	if err != nil {
		return nil, err
	}

	obj, err := language.FromValue(data, false, value.Debug())
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func stringifyFn(ctx native.FnCtx) (language.Object, error) {
	value, err := ctx.Get("object")
	if err != nil {
		return nil, err
	}

	goValue, err := language.ToValue(value, true)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(goValue)
	if err != nil {
		return nil, err
	}

	return language.NewString(string(data), value.Debug()), nil
}
