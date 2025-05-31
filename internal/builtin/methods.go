package builtin

import (
	"fmt"
	"strings"

	"github.com/nubogo/nubo/language"
	"github.com/nubogo/nubo/native"
)

func GetBuiltins() map[string]language.Object {
	return map[string]language.Object{
		"println": native.NewFunction(printlnFn),
		"type":    native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeString, typeFn),
		"inspect": native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeString, inspectFn),
	}
}

func printlnFn(args []language.Object) (language.Object, error) {
	var out []string
	for _, arg := range args {
		out = append(out, arg.String())
	}
	fmt.Println(strings.Join(out, " "))
	return nil, nil
}

func typeFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}
	return language.NewString(obj.TypeString(), nil), nil
}

func inspectFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}
	return language.NewString(obj.Inspect(), nil), nil
}
