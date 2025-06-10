package layoutjs

import (
	"embed"
	"io"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
)

//go:embed layout.js
var jsFile embed.FS

func NewLayoutJS() language.Object {
	instance := language.NewStruct("@std/sdkjs", nil, nil)
	proto := instance.GetPrototype()

	proto.SetObject("create", native.NewTypedFunction([]language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "children"},
		&language.BasicFnArg{TypeVal: language.NewFunctionType(language.TypeString, language.TypeString), NameVal: "handler"},
		&language.BasicFnArg{TypeVal: language.NewDictType(language.TypeString, language.NewListType(language.TypeString)), NameVal: "headers"},
	}, language.TypeString, fnCreate))

	proto.SetObject("script", native.NewTypedFunction(nil, language.TypeString, func(ctx native.FnCtx) (language.Object, error) {
		script, err := jsFile.Open("layout.js")
		if err != nil {
			return nil, err
		}
		defer script.Close()

		data, err := io.ReadAll(script)
		if err != nil {
			return nil, err
		}

		return language.NewString("<script>"+string(data)+"</script>", nil), nil
	}))

	return instance
}

func fnCreate(ctx native.FnCtx) (language.Object, error) {
	headers, _ := ctx.Get("headers")
	is := isSdk(headers)

	children, _ := ctx.Get("children")
	if !is {
		handler, _ := ctx.Get("handler")
		handl := handler.Value().(func(args []language.Object) (language.Object, error))
		return handl([]language.Object{children})
	}

	return children, nil
}

func isSdk(headers language.Object) bool {
	getter, ok := headers.GetPrototype().GetObject("get")
	if !ok {
		return false
	}

	get := getter.Value().(func([]language.Object) (language.Object, error))
	val, err := get([]language.Object{language.NewString("X-Nubo-Link", nil)})
	if err != nil {
		return false
	}

	vals := val.(*language.List)
	for _, value := range vals.Data {
		if value.String() == "true" {
			return true
		}
	}

	return false
}
