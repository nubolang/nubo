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

type multiCloser struct {
	closers []io.Closer
}

func (mc *multiCloser) Close() error {
	var firstErr error
	for _, c := range mc.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func NewIOStream(r io.Reader, optWriter ...io.Writer) language.Object {
	var w io.Writer
	if len(optWriter) > 0 {
		w = optWriter[0]
	}

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

	mc := &multiCloser{}
	if rc, ok := r.(io.Closer); ok {
		mc.closers = append(mc.closers, rc)
	}

	if w != nil {
		proto.SetObject(ctx, "write", native.NewTypedFunction(ctx, []language.FnArg{
			&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "content"},
		}, language.TypeInt, streamWriteStringFn(w)))
		proto.SetObject(ctx, "writeByte", native.NewTypedFunction(ctx, []language.FnArg{
			&language.BasicFnArg{TypeVal: language.TypeByte, NameVal: "content"},
		}, language.TypeInt, streamWriteByteFn(w)))

		if wc, ok := w.(io.Closer); ok {
			mc.closers = append(mc.closers, wc)
		}
	}

	proto.SetObject(ctx, "close", native.NewTypedFunction(ctx, nil, language.TypeVoid, streamCloseFn(mc)))

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

func streamWriteStringFn(w io.Writer) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		content, err := ctx.Get("content")
		if err != nil {
			return nil, err
		}
		str := content.String()
		n, err := w.Write([]byte(str))
		if err != nil {
			return nil, err
		}
		return language.NewInt(int64(n), nil), nil
	}
}

func streamWriteByteFn(w io.Writer) native.FunctionWrapper {
	return func(ctx native.FnCtx) (language.Object, error) {
		content, err := ctx.Get("content")
		if err != nil {
			return nil, err
		}

		b := content.Value().(byte)
		n, err := w.Write([]byte{b})
		if err != nil {
			return nil, err
		}
		return language.NewInt(int64(n), nil), nil
	}
}
