package modules

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
)

type Response struct {
	inst *language.Struct

	body    *bytes.Buffer
	code    int
	headers http.Header
	written bool

	w http.ResponseWriter
}

func NewResponse(w http.ResponseWriter) *Response {
	r := &Response{
		w:       w,
		body:    &bytes.Buffer{},   // Initialize body as a new bytes.Buffer
		code:    http.StatusOK,     // Default HTTP status code to 200 OK
		headers: make(http.Header), // Initialize headers as an empty http.Header map
		written: false,
	}

	// Default content-type
	r.headers.Set("Content-Type", "text/html")

	inst := language.NewStruct("@server/response", nil, nil)
	r.setupInstance(inst)
	r.inst = inst

	return r
}

func (r *Response) Pkg() language.Object {
	return r.inst
}

func (r *Response) Sync() {
	if r.written {
		return
	}

	r.written = true

	// 1. Copy all the headers
	for key, values := range r.headers {
		for _, value := range values {
			r.w.Header().Add(key, value)
		}
	}

	// 2. Set the status code
	if r.code != 0 {
		r.w.WriteHeader(r.code)
	} else {
		r.w.WriteHeader(http.StatusOK)
	}

	// 3. Write the content
	r.w.Write(r.body.Bytes())
}

func (r *Response) setupInstance(inst *language.Struct) {
	proto := inst.GetPrototype()

	proto.SetObject("status", native.NewTypedFunction(native.OneArg("code", language.TypeInt), language.TypeVoid, r.fnStatus))
	proto.SetObject("write", native.NewTypedFunction(native.OneArg("content", language.TypeAny), language.TypeVoid, r.fnWrite))
	proto.SetObject("header", native.NewTypedFunction([]language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "key"},
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "value"},
	}, language.TypeVoid, r.fnHeader))
	proto.SetObject("flushbuf", native.NewTypedFunction(nil, language.TypeVoid, r.fnFlushbuf))
}

func (r *Response) fnStatus(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("code")
	if err != nil {
		return nil, err
	}

	code := int(obj.Value().(int64))
	if code < 100 || code > 599 {
		return nil, fmt.Errorf("status code must be between 100 and 599")
	}

	r.code = code
	return nil, err
}

func (r *Response) fnWrite(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("content")
	if err != nil {
		return nil, err
	}

	_, err = r.body.WriteString(obj.String())
	return nil, err
}

func (r *Response) fnHeader(ctx native.FnCtx) (language.Object, error) {
	key, _ := ctx.Get("key")
	value, _ := ctx.Get("value")

	r.headers.Set(key.String(), value.String())
	return nil, nil
}

func (r *Response) fnFlushbuf(ctx native.FnCtx) (language.Object, error) {
	r.body.Reset()
	return nil, nil
}
