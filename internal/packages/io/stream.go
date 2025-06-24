package io

import (
	"bufio"
	"errors"
	"io"
	"strings"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
)

func NewIOStream(r io.Reader) language.Object {
	instance, err := streamStruct.NewInstance()
	if err != nil {
		return nil
	}

	proto := instance.GetPrototype().(*language.StructPrototype)
	proto.Unlock()
	defer proto.Lock()

	proto.SetObject("read", native.NewTypedFunction(nil, language.TypeString, streamReadFn(r)))
	proto.SetObject("readAll", native.NewTypedFunction(nil, language.TypeString, streamReadAllFn(r)))
	proto.SetObject("readByte", native.NewTypedFunction(nil, language.TypeInt, streamReadByteFn(r)))
	proto.SetObject("readLine", native.NewTypedFunction(nil, language.TypeString, streamReadLineFn(r)))
	if rc, ok := r.(io.Closer); ok {
		proto.SetObject("close", native.NewTypedFunction(nil, language.TypeVoid, streamCloseFn(rc)))
	}

	return instance
}

func streamReadFn(r io.Reader) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			return nil, err
		}
		return language.NewString(string(buf[:n]), nil), nil
	}
}

func streamReadAllFn(r io.Reader) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return language.NewString(string(data), nil), nil
	}
}

func streamReadByteFn(r io.Reader) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		b := make([]byte, 1)
		_, err := r.Read(b)
		if err != nil {
			return nil, err
		}
		return language.NewByte(b[0], nil), nil
	}
}

func streamReadLineFn(r io.Reader) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		bufReader := bufio.NewReader(r)
		line, err := bufReader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		return language.NewString(strings.TrimRight(line, "\r\n"), nil), nil
	}
}

func streamCloseFn(rc io.Closer) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		return nil, rc.Close()
	}
}
