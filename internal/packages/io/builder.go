package io

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

// builder owns its buffer. String snapshots copy so Reset can retain capacity.
// Like the surrounding mutable Nubo collections, builders require external synchronization.
type builder struct{ data []byte }

func newBuilder(a *n.Args) (any, error) {
	capacity := a.Name("capacity").Value().(int64)

	if capacity < 0 || int64(int(capacity)) != capacity {
		return nil, fmt.Errorf("capacity must be a nonnegative native integer")
	}

	b := &builder{
		data: make([]byte, 0, int(capacity)),
	}
	obj := n.NewPackage("StringBuilder", nil)

	p := obj.GetPrototype()
	ctx := context.Background()

	add := func(name string, desc *n.FnDescriber, fn func(*n.Args) (any, error)) {
		p.SetObject(ctx, name, n.Function(desc, fn))
	}

	add("append", n.Describe(n.Arg("text", n.TString)), func(a *n.Args) (any, error) { b.data = append(b.data, a.Name("text").String()...); return nil, nil })

	add("appendByte", n.Describe(n.Arg("value", n.TInt)), func(a *n.Args) (any, error) {
		v := a.Name("value").Value().(int64)
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("byte must be between 0 and 255")
		}
		b.data = append(b.data, byte(v))
		return nil, nil
	})

	add("appendChar", n.Describe(n.Arg("value", language.TypeChar)), func(a *n.Args) (any, error) {
		b.data = utf8.AppendRune(b.data, a.Name("value").Value().(rune))
		return nil, nil
	})

	add("string", n.Describe().Returns(n.TString), func(a *n.Args) (any, error) { return string(b.data), nil })

	add("length", n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) { return len(b.data), nil })

	add("capacity", n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) { return cap(b.data), nil })

	add("reset", n.Describe(), func(a *n.Args) (any, error) { b.data = b.data[:0]; return nil, nil })

	add("grow", n.Describe(n.Arg("additional", n.TInt)), func(a *n.Args) (any, error) {
		v := a.Name("additional").Value().(int64)
		if v < 0 || v > int64(int(^uint(0)>>1)-len(b.data)) {
			return nil, fmt.Errorf("invalid growth")
		}

		required := len(b.data) + int(v)
		if required > cap(b.data) {
			data := make([]byte, len(b.data), required)
			copy(data, b.data)
			b.data = data
		}

		return nil, nil
	})

	return obj, nil
}
