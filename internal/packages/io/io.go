package io

import (
	"fmt"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
)

func NewIO() language.Object {
	instance := language.NewStruct("@std/io", nil, nil)
	proto := instance.GetPrototype()

	proto.SetObject("read", native.NewTypedFunction(native.OneArg("text", language.TypeString, language.NewString("", nil)), language.TypeString, readFn))
	proto.SetObject("open", native.NewTypedFunction([]language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "file"},
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "encoding", DefaultVal: language.NewString("utf-8", nil)},
	}, language.TypeStructInstance, openFn))

	return instance
}

func readFn(ctx native.FnCtx) (language.Object, error) {
	text, err := ctx.Get("text")
	if err != nil {
		return nil, err
	}

	fmt.Print(text.Value())
	var value string
	fmt.Scanln(&value)

	return language.NewString(value, text.Debug()), nil
}
