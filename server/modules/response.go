package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

type Response struct {
	inst *language.StructInstance

	body    *bytes.Buffer
	code    int
	headers http.Header
	written bool

	w http.ResponseWriter
	r *http.Request
}

var responseStruct *language.Struct

func NewResponse(w http.ResponseWriter, req *http.Request) (*Response, error) {
	r := &Response{
		w:       w,
		r:       req,
		body:    &bytes.Buffer{},   // Initialize body as a new bytes.Buffer
		code:    http.StatusOK,     // Default HTTP status code to 200 OK
		headers: make(http.Header), // Initialize headers as an empty http.Header map
		written: false,
	}

	// Default content-type
	r.headers.Set("Content-Type", "text/html")

	if responseStruct == nil {
		responseStruct = language.NewStruct("response", nil, nil)
	}

	inst, err := responseStruct.NewInstance()
	if err != nil {
		return nil, err
	}

	if err := r.setupInstance(inst); err != nil {
		return nil, err
	}
	r.inst = inst

	return r, nil
}

func (r *Response) Pkg() language.Object {
	return r.inst
}

func (r *Response) SetStatus(code int) error {
	if code < 100 || code > 599 {
		return fmt.Errorf("status code must be between 100 and 599")
	}
	r.code = code
	return nil
}

func (r *Response) Sync() error {
	if r.written {
		return nil
	}

	r.written = true

	r.applyHeaders()

	if r.code != 0 {
		r.w.WriteHeader(r.code)
	} else {
		r.w.WriteHeader(http.StatusOK)
	}

	_, err := r.w.Write(r.body.Bytes())
	return err
}

func (r *Response) applyHeaders() {
	for key, values := range r.headers {
		for _, value := range values {
			r.w.Header().Add(key, value)
		}
	}
}

func (r *Response) setupInstance(inst *language.StructInstance) error {
	proto, ok := inst.GetPrototype().(*language.StructPrototype)
	if !ok {
		return fmt.Errorf("response prototype has invalid type")
	}
	proto.Unlock()
	defer proto.Lock()

	ctx := r.r.Context()

	proto.SetObject(ctx, "status", native.NewTypedFunction(ctx, native.OneArg("code", language.TypeInt), language.TypeVoid, r.fnStatus))
	proto.SetObject(ctx, "write", native.NewTypedFunction(ctx, native.OneArg("content", language.TypeAny), language.TypeVoid, r.fnWrite))
	proto.SetObject(ctx, "header", native.NewTypedFunction(ctx, []language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "key"},
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "value"},
	}, language.TypeVoid, r.fnHeader))
	proto.SetObject(ctx, "flushbuf", native.NewTypedFunction(ctx, nil, language.TypeVoid, r.fnFlushbuf))
	proto.SetObject(ctx, "json", native.NewTypedFunction(ctx, native.OneArg("data", language.TypeAny), language.TypeVoid, r.fnJSON))
	proto.SetObject(ctx, "setCookie", native.NewTypedFunction(ctx, []language.FnArg{
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "name"},
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "value"},
		&language.BasicFnArg{TypeVal: language.Nullable(language.TypeInt), NameVal: "maxAge", DefaultVal: language.Nil},
		&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "path", DefaultVal: n.String("/")},
	}, language.TypeVoid, r.fnSetCookie))
	proto.SetObject(ctx, "redirect", native.NewTypedFunction(ctx, native.OneArg("url", language.TypeString), language.TypeVoid, r.fnRedirect))
	return nil
}

func (r *Response) fnStatus(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("code")
	if err != nil {
		return nil, err
	}

	return nil, r.SetStatus(int(obj.Value().(int64)))
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
	key, err := ctx.Get("key")
	if err != nil {
		return nil, err
	}
	value, err := ctx.Get("value")
	if err != nil {
		return nil, err
	}

	r.headers.Set(key.String(), value.String())
	return nil, nil
}

func (r *Response) fnFlushbuf(ctx native.FnCtx) (language.Object, error) {
	r.body.Reset()
	return nil, nil
}

func (r *Response) fnJSON(ctx native.FnCtx) (language.Object, error) {
	data, err := ctx.Get("data")
	if err != nil {
		return nil, err
	}

	value, err := language.ToValue(data, true)
	if err != nil {
		return nil, err
	}

	r.headers.Set("Content-Type", "application/json")
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	r.body.Reset()
	_, err = r.body.Write(bytes)
	return nil, err
}

func (r *Response) fnSetCookie(ctx native.FnCtx) (language.Object, error) {
	name, err := ctx.Get("name")
	if err != nil {
		return nil, err
	}
	value, err := ctx.Get("value")
	if err != nil {
		return nil, err
	}
	maxAgeObj, err := ctx.Get("maxAge")
	if err != nil {
		return nil, err
	}
	pathObj, err := ctx.Get("path")
	if err != nil {
		return nil, err
	}

	cookie := &http.Cookie{
		Name:  name.String(),
		Value: value.String(),
		Path:  pathObj.String(),
	}

	if maxAgeObj.Type() != n.TNil {
		cookie.MaxAge = int(maxAgeObj.Value().(int64))
	}

	r.headers.Add("Set-Cookie", cookie.String())
	return nil, nil
}

func (r *Response) fnRedirect(ctx native.FnCtx) (language.Object, error) {
	urlObj, err := ctx.Get("url")
	if err != nil {
		return nil, err
	}
	url := urlObj.String()

	r.written = true
	r.applyHeaders()
	http.Redirect(r.w, r.r, url, http.StatusFound)
	return nil, nil
}
