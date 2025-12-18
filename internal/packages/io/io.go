package io

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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
	proto.SetObject(ctx, "read", native.NewTypedFunction(ctx, []language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "text", DefaultVal: language.NewString("", nil)},
		&language.BasicFnArg{TypeVal: language.TypeBool, NameVal: "trim", DefaultVal: language.NewBool(true, nil)},
		&language.BasicFnArg{TypeVal: language.TypeChar, NameVal: "endln", DefaultVal: language.NewChar('\n', nil)},
	}, language.TypeString, readFn))
	proto.SetObject(ctx, "open", native.NewTypedFunction(ctx, []language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "file"},
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "mode", DefaultVal: language.NewString("r", nil)},
		&language.BasicFnArg{TypeVal: language.TypeInt, NameVal: "perm", DefaultVal: language.NewInt(int64(os.ModePerm), nil)},
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

	endln, err := ctx.Get("endln")
	if err != nil {
		return nil, err
	}

	trim, err := ctx.Get("trim")
	if err != nil {
		return nil, err
	}

	end := endln.Value().(rune)

	fmt.Print(text.String())
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString(byte(end))
	if err != nil {
		return nil, err
	}

	value = strings.TrimSuffix(value, string(end))

	if trim.Value().(bool) {
		value = strings.TrimSpace(value)
	}

	return language.NewString(value, text.Debug()), nil
}
