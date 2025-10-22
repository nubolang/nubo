package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var respStruct *language.Struct

func getResp(dg *debug.Debug) *language.Struct {
	if respStruct == nil {
		respStruct = language.NewStruct("response", []language.StructField{
			{Name: "url", Type: n.TString},
			{Name: "status", Type: n.TInt},
			{Name: "headers", Type: n.NewDictType(n.TString, n.TUnion(n.TString, n.TTList(n.TString)))},
		}, dg)
	}
	return respStruct
}

func createResp(data struct {
	Url     string
	Status  int
	Headers http.Header
	Body    []byte
}, dg *debug.Debug) (language.Object, error) {
	res := getResp(dg)
	response, err := res.NewInstance()
	if err != nil {
		return nil, err
	}

	proto := response.GetPrototype().(*language.StructPrototype)
	proto.Unlock()
	defer proto.Lock()
	ctx := context.Background()

	proto.SetObject(ctx, "url", n.String(data.Url, dg))
	proto.SetObject(ctx, "status", n.Int(data.Status, dg))

	m := make(map[any]any)
	for k, v := range data.Headers {
		if len(v) == 1 {
			m[k] = v[0]
		} else {
			m[k] = v
		}
	}

	headers, err := n.Dict(m, dg)
	if err != nil {
		return nil, err
	}

	proto.SetObject(ctx, "headers", headers)
	proto.SetObject(ctx, "body", n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
		return n.String(string(data.Body), dg), nil
	}))
	proto.SetObject(ctx, "json", n.Function(n.Describe().Returns(n.TAny), func(a *n.Args) (any, error) {
		var d any
		err = json.Unmarshal(data.Body, &d)
		if err != nil {
			return nil, err
		}

		obj, err := language.FromValue(d, false, dg)
		if err != nil {
			return nil, err
		}

		return obj, nil
	}))

	return response, nil
}
