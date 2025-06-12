package layoutjs

import (
	"embed"
	"io"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

//go:embed layout.min.js
var jsFile embed.FS

func NewLayoutJS() language.Object {
	instance := language.NewStruct("@std/sdkjs", nil, nil)
	proto := instance.GetPrototype()

	proto.SetObject("create", native.NewTypedFunction([]language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "children"},
		&language.BasicFnArg{TypeVal: language.NewFunctionType(language.TypeString, language.TypeString), NameVal: "handler"},
		&language.BasicFnArg{TypeVal: language.TypeStructInstance, NameVal: "request"},
	}, language.TypeString, fnCreate))

	proto.SetObject("script", native.NewTypedFunction(nil, language.TypeString, func(ctx native.FnCtx) (language.Object, error) {
		script, err := jsFile.Open("layout.min.js")
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
	request, _ := ctx.Get("request")
	is := isSdk(request)

	children, _ := ctx.Get("children")
	if !is {
		handler, _ := ctx.Get("handler")
		handl := handler.Value().(func(args []language.Object) (language.Object, error))
		return handl([]language.Object{children})
	}

	return children, nil
}

func isSdk(request language.Object) bool {
	query, ok := request.GetPrototype().GetObject("query")
	if !ok {
		return false
	}

	get := query.(*language.Function)
	val, err := get.Data([]language.Object{n.String("__nubo_fragment")})
	if err != nil {
		return false
	}

	return val.String() == "partial"
}
