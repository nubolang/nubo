package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	stdio "github.com/nubolang/nubo/internal/packages/io"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

var requestStruct *language.Struct

func RequestStruct() *language.Struct {
	if requestStruct == nil {
		requestStruct = language.NewStruct("request", nil, nil)
	}

	return requestStruct
}

func NewRequest(r *http.Request) (language.Object, error) {
	if requestStruct == nil {
		requestStruct = language.NewStruct("request", nil, nil)
	}

	inst, err := requestStruct.NewInstance()
	if err != nil {
		return nil, err
	}

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
			return language.NewString(param, name.Debug()), nil
		}

		return language.Nil, nil
	}))

	proto.SetObject("query", native.NewTypedFunction(native.OneArg("name", language.TypeString), language.Nullable(language.TypeString), func(ctx native.FnCtx) (language.Object, error) {
		name, _ := ctx.Get("name")
		value := r.URL.Query().Get(name.String())
		if value == "" {
			return language.Nil, nil
		}
		return language.NewString(value, name.Debug()), nil
	}))

	proto.SetObject("cookie", native.NewTypedFunction(native.OneArg("name", language.TypeString), language.Nullable(language.TypeString), func(ctx native.FnCtx) (language.Object, error) {
		name, _ := ctx.Get("name")
		cookie, err := r.Cookie(name.String())
		if err != nil {
			return language.Nil, nil
		}
		return language.NewString(cookie.Value, name.Debug()), nil
	}))

	proto.SetObject("path", language.NewString(r.URL.Path, nil))
	proto.SetObject("url", language.NewString(r.URL.String(), nil))
	proto.SetObject("host", language.NewString(r.Host, nil))
	proto.SetObject("ip", language.NewString(r.RemoteAddr, nil))

	proto.SetObject("is", native.NewTypedFunction(native.OneArg("method", language.TypeString), language.TypeBool, func(ctx native.FnCtx) (language.Object, error) {
		method, _ := ctx.Get("method")
		return language.NewBool(strings.EqualFold(r.Method, method.String()), method.Debug()), nil
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

	proto.SetObject("json", native.NewTypedFunction(nil, language.TypeAny, func(ctx native.FnCtx) (language.Object, error) {
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

	proto.SetObject("form", native.NewTypedFunction(native.OneArg("name", language.TypeString), language.Nullable(language.TypeString), func(ctx native.FnCtx) (language.Object, error) {
		if err := r.ParseForm(); err != nil {
			return language.Nil, nil
		}
		name, _ := ctx.Get("name")
		value := r.FormValue(name.String())
		if value == "" {
			return language.Nil, nil
		}
		return language.NewString(value, nil), nil
	}))

	proto.SetObject("formData", native.NewTypedFunction(nil, language.Nullable(language.TypeDict), func(ctx native.FnCtx) (language.Object, error) {
		if err := r.ParseMultipartForm(10 << 20); err != nil { // Max 10MB
			if err := r.ParseForm(); err != nil {
				return nil, err
			}
		}

		var data = make(map[any]any)

		// Multipart form
		if r.MultipartForm != nil && r.MultipartForm.Value != nil {
			for k, vals := range r.MultipartForm.Value {
				if len(vals) > 0 {
					data[k] = vals[0]
				}
			}
		} else {
			// Urlencoded form
			for k, vals := range r.Form {
				if len(vals) > 0 {
					data[k] = vals[0]
				}
			}
		}

		return n.Dict(data)
	}))

	proto.SetObject("file", native.NewTypedFunction(native.OneArg("name", language.TypeString), language.Nullable(language.TypeStructInstance), func(ctx native.FnCtx) (language.Object, error) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return language.Nil, nil
		}

		nameObj, _ := ctx.Get("name")
		name := nameObj.String()

		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			fhs, ok := r.MultipartForm.File[name]
			if !ok || len(fhs) == 0 {
				return language.Nil, nil
			}

			fileHeader := fhs[0]

			file, err := fileHeader.Open()
			if err != nil {
				return nil, err
			}
			fileObj := stdio.NewIOStream(file)
			proto := fileObj.GetPrototype()
			proto.SetObject("filename", language.NewString(fileHeader.Filename, nil))
			proto.SetObject("size", language.NewInt(fileHeader.Size, nil))
			proto.SetObject("header", language.NewString(fileHeader.Header.Get("Content-Type"), nil))

			return fileObj, nil
		}

		return language.Nil, nil
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
