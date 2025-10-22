package io

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

func NewIOStream(r io.Reader) language.Object {
	instance, err := streamStruct.NewInstance()
	if err != nil {
		return nil
	}

	proto := instance.GetPrototype().(*language.StructPrototype)
	proto.Unlock()
	defer proto.Lock()

	ctx := context.Background()
	proto.SetObject(ctx, "read", native.NewTypedFunction(ctx, nil, language.TypeString, streamReadFn(r)))
	proto.SetObject(ctx, "readAll", native.NewTypedFunction(ctx, nil, language.TypeString, streamReadAllFn(r)))
	proto.SetObject(ctx, "readByte", native.NewTypedFunction(ctx, nil, language.TypeInt, streamReadByteFn(r)))
	proto.SetObject(ctx, "readLine", native.NewTypedFunction(ctx, nil, language.TypeString, streamReadLineFn(r)))
	proto.SetObject(ctx, "readLines", native.NewTypedFunction(ctx, nil, n.TTList(n.TString), streamReadLinesFn(r)))
	if rc, ok := r.(io.Closer); ok {
		proto.SetObject(ctx, "close", native.NewTypedFunction(ctx, nil, language.TypeVoid, streamCloseFn(rc)))
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

func streamReadLinesFn(r io.Reader) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		reader := bufio.NewReader(r)

		var lines []language.Object

		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				if len(line) > 0 {
					lines = append(lines, language.NewString(line, nil))
				}
				break
			}
			if err != nil {
				return nil, err
			}
			lines = append(lines, language.NewString(line, nil))
		}

		return language.NewList(lines, n.TString, nil), nil
	}
}

func streamCloseFn(rc io.Closer) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		return nil, rc.Close()
	}
}
