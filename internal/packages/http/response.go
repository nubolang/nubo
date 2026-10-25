package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

const responseBucket = "http.response"

type responseState struct {
	body []byte
}

var (
	responseOnce sync.Once
	responseType *language.Struct
)

func getResp(dg *debug.Debug) *language.Struct {
	responseOnce.Do(func() {
		responseType = language.NewStruct("response", []language.StructField{
			{Name: "url", Type: n.TString},
			{Name: "status", Type: n.TInt},
			{Name: "statusText", Type: n.TString},
			{Name: "ok", Type: n.TBool},
			{Name: "headers", Type: n.NewDictType(n.TString, n.TAny)},
			{Name: "cookies", Type: n.NewDictType(n.TString, n.TString)},
		}, dg)

		proto := responseType.GetPrototype().(*language.StructPrototype)
		proto.Unlock()
		defer func() {
			proto.Lock()
			proto.Implement()
		}()

		ctx := context.Background()
		set := func(name string, fn language.Object) { _ = proto.SetObject(ctx, name, fn) }

		set("body", n.Function(n.Describe(
			n.Arg("self", responseType.Type()),
		).Returns(n.TString), func(args *n.Args) (any, error) {
			body, err := responseBody(args.Name("self"))
			if err != nil {
				return nil, err
			}
			return string(body), nil
		}))

		set("bytes", n.Function(n.Describe(
			n.Arg("self", responseType.Type()),
		).Returns(n.TTList(n.TByte)), func(args *n.Args) (any, error) {
			body, err := responseBody(args.Name("self"))
			if err != nil {
				return nil, err
			}
			items := make([]language.Object, len(body))
			for index, value := range body {
				items[index] = n.Byte(value, args.Name("self").Debug())
			}
			return language.NewList(items, n.TByte, args.Name("self").Debug()), nil
		}))

		set("json", n.Function(n.Describe(
			n.Arg("self", responseType.Type()),
		).Returns(n.TAny), func(args *n.Args) (any, error) {
			self := args.Name("self")
			body, err := responseBody(self)
			if err != nil {
				return nil, err
			}

			decoder := json.NewDecoder(strings.NewReader(string(body)))
			decoder.UseNumber()
			var value any
			if err := decoder.Decode(&value); err != nil {
				return nil, err
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				return nil, fmt.Errorf("response contains multiple JSON values")
			}
			return parseJSONValue(value, self.Debug())
		}))
	})

	return responseType
}

func createResp(resp *http.Response, body []byte, dg *debug.Debug) (language.Object, error) {
	instance, err := getResp(dg).NewInstance()
	if err != nil {
		return nil, err
	}
	instance.BucketSet(responseBucket, &responseState{body: body})

	urlString := ""
	if resp.Request != nil && resp.Request.URL != nil {
		urlString = resp.Request.URL.String()
	}

	ctx := context.Background()
	proto := instance.GetPrototype()
	fields := map[string]language.Object{
		"url":        n.String(urlString, dg),
		"status":     n.Int(resp.StatusCode, dg),
		"statusText": n.String(http.StatusText(resp.StatusCode), dg),
		"ok":         n.Bool(resp.StatusCode >= 200 && resp.StatusCode < 300, dg),
	}
	for name, value := range fields {
		if err := proto.SetObject(ctx, name, value); err != nil {
			return nil, err
		}
	}

	keys := make([]language.Object, 0, len(resp.Header))
	values := make([]language.Object, 0, len(resp.Header))
	for name, entries := range resp.Header {
		keys = append(keys, n.String(name, dg))
		if len(entries) == 1 {
			values = append(values, n.String(entries[0], dg))
			continue
		}
		items := make([]language.Object, len(entries))
		for i, entry := range entries {
			items[i] = n.String(entry, dg)
		}
		values = append(values, language.NewList(items, n.TString, dg))
	}
	headerDict, err := language.NewDict(keys, values, n.TString, n.TAny, dg)
	if err != nil {
		return nil, err
	}
	if err := proto.SetObject(ctx, "headers", headerDict); err != nil {
		return nil, err
	}

	cookieKeys := make([]language.Object, 0)
	cookieValues := make([]language.Object, 0)
	for _, cookie := range resp.Cookies() {
		cookieKeys = append(cookieKeys, n.String(cookie.Name, dg))
		cookieValues = append(cookieValues, n.String(cookie.Value, dg))
	}
	cookies, err := language.NewDict(cookieKeys, cookieValues, n.TString, n.TString, dg)
	if err != nil {
		return nil, err
	}
	if err := proto.SetObject(ctx, "cookies", cookies); err != nil {
		return nil, err
	}

	return instance, nil
}

func responseBody(obj language.Object) ([]byte, error) {
	instance, ok := obj.(*language.StructInstance)
	if !ok {
		return nil, fmt.Errorf("expected an HTTP response")
	}
	value, ok := instance.BucketGet(responseBucket)
	if !ok {
		return nil, fmt.Errorf("HTTP response body is not available")
	}
	state, ok := value.(*responseState)
	if !ok {
		return nil, fmt.Errorf("HTTP response state is invalid")
	}
	return state.body, nil
}

func parseJSONValue(value any, dg *debug.Debug) (language.Object, error) {
	switch typed := value.(type) {
	case json.Number:
		if integer, err := strconv.ParseInt(typed.String(), 10, 64); err == nil {
			return n.Int64(integer, dg), nil
		}
		floating, err := strconv.ParseFloat(typed.String(), 64)
		if err != nil {
			return nil, err
		}
		return n.Float(floating, dg), nil

	case map[string]any:
		keys := make([]language.Object, 0, len(typed))
		values := make([]language.Object, 0, len(typed))
		for key, item := range typed {
			converted, err := parseJSONValue(item, dg)
			if err != nil {
				return nil, err
			}
			keys = append(keys, n.String(key, dg))
			values = append(values, converted)
		}
		return language.NewDict(keys, values, n.TString, n.TAny, dg)

	case []any:
		items := make([]language.Object, len(typed))
		for i, item := range typed {
			converted, err := parseJSONValue(item, dg)
			if err != nil {
				return nil, err
			}
			items[i] = converted
		}
		return language.NewList(items, n.TAny, dg), nil

	default:
		return language.FromValue(value, false, dg)
	}
}
