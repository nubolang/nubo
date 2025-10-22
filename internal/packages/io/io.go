package io

import (
	"context"
	"fmt"
	"os"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

var streamStruct *language.Struct

func NewIO(dg *debug.Debug) language.Object {
	instance := n.NewPackage("io", dg)
	proto := instance.GetPrototype()

	if streamStruct == nil {
		streamStruct = language.NewStruct("Stream", nil, dg)
	}

	ctx := context.Background()
	proto.SetObject(ctx, "Stream", streamStruct)
	proto.SetObject(ctx, "read", native.NewTypedFunction(ctx, native.OneArg("text", language.TypeString, language.NewString("", nil)), language.TypeString, readFn))
	proto.SetObject(ctx, "open", native.NewTypedFunction(ctx, []language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "file"},
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "encoding", DefaultVal: language.NewString("utf-8", nil)},
	}, streamStruct.Type(), openFn))
	proto.SetObject(ctx, "writeFile", n.Function(n.Describe(n.Arg("file", n.TString), n.Arg("data", n.TAny), n.Arg("perm", n.TInt, n.Int(int(os.ModePerm), nil))), writeFile))

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
