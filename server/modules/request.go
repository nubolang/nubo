package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
)

func NewRequest(r *http.Request) (language.Object, error) {
	inst := language.NewStruct("@server/request", nil, nil)
	proto := inst.GetPrototype()

	headers, err := newHeadersDict(r)
	if err != nil {
		return nil, err
	}

	var (
		body   []byte
		params = make(map[string]string)
	)

	if raw := r.Context().Value("__params__"); raw != nil {
		value, ok := raw.(map[string]string)
		if ok {
			params = value
		}
	}

	proto.SetObject("method", language.NewString(r.Method, nil))
	proto.SetObject("headers", headers)
	proto.SetObject("secure", language.NewBool(r.TLS != nil, nil))
	proto.SetObject("accepts", native.NewTypedFunction(native.OneArg("contentType", language.TypeString), language.TypeBool, func(ctx native.FnCtx) (language.Object, error) {
		contentType, _ := ctx.Get("contentType")
		accept := strings.ToLower(r.Header.Get("Accept"))
		return language.NewBool(strings.ToLower(contentType.String()) == accept, nil), nil
	}))
	proto.SetObject("param", native.NewTypedFunction(native.OneArg("name", language.TypeString), language.Nullable(language.TypeString), func(ctx native.FnCtx) (language.Object, error) {
		name, _ := ctx.Get("name")
		param, ok := params[name.String()]
		if ok {
			return language.NewString(param, nil), nil
		}

		return language.Nil, nil
	}))
	proto.SetObject("body", native.NewTypedFunction(nil, language.TypeString, func(ctx native.FnCtx) (language.Object, error) {
		if body == nil {
			var err error
			body, err = io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("could not read body content: '%v'", err)
			}
		}

		return language.NewString(string(body), nil), nil
	}))
	proto.SetObject("bodyJSON", native.NewTypedFunction(nil, language.TypeAny, func(ctx native.FnCtx) (language.Object, error) {
		if body == nil {
			var err error
			body, err = io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("could not read body content: '%v'", err)
			}
		}

		if len(body) == 0 {
			return language.Nil, nil
		}

		var data any
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("could not parse JSON body: '%v'", err)
		}

		return language.FromValue(data)
	}))

	return inst, nil
}

func newHeadersDict(r *http.Request) (*language.Dict, error) {
	var (
		keys   = make([]language.Object, 0, len(r.Header))
		values = make([]language.Object, 0, len(r.Header))
	)

	for key, val := range r.Header {
		keys = append(keys, language.NewString(key, nil))

		var items = make([]language.Object, len(val))
		for j, item := range val {
			items[j] = language.NewString(item, nil)
		}

		values = append(values, language.NewList(items, language.TypeString, nil))
	}

	return language.NewDict(keys, values, language.TypeString, language.NewListType(language.TypeString), nil)
}
