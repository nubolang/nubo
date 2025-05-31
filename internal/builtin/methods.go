package builtin

import (
	"fmt"
	"strings"
	"time"

	"github.com/nubogo/nubo/language"
	"github.com/nubogo/nubo/native"
)

func GetBuiltins() map[string]language.Object {
	return map[string]language.Object{
		"println":   native.NewFunction(printlnFn),
		"type":      native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeString, typeFn),
		"inspect":   native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeString, inspectFn),
		"keepAlive": native.NewTypedFunction(native.OneArg("ms", language.TypeInt, language.NewInt(0, nil)), language.TypeVoid, keepAliveFn),
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

func keepAliveFn(ctx native.FnCtx) (language.Object, error) {
	ms, err := ctx.Get("ms")
	if err != nil {
		return nil, err
	}

	value := ms.Value().(int64)
	if value < 0 {
		return nil, fmt.Errorf("duration must be non-negative")
	}

	time.Sleep(time.Duration(value) * time.Millisecond)
	return nil, nil
}
