package language

import (
	"context"
	"sync"
)

type FunctionPrototype struct {
	base *Function
	data map[string]Object
	mu   sync.RWMutex
}

func NewFunctionPrototype(base *Function) *FunctionPrototype {
	fp := &FunctionPrototype{
		base: base,
		data: make(map[string]Object),
	}

	argObjs := make([]Object, len(base.ArgTypes))
	for i, argType := range base.ArgTypes {
		var fallback Object = Nil
		if argType.Default() != nil {
			fallback = argType.Default().Clone()
		}

		typ := argType.Type()
		typs := make([]Object, 0)
		for {
			typs = append(typs, NewString(typ.String(), base.debug))
			if typ.Next == nil {
				break
			}
			typ = typ.Next
		}

		var realTyp Object
		if len(typs) == 1 {
			realTyp = typs[0]
		} else {
			realTyp = NewList(typs, TypeString, base.debug)
		}

		dict, _ := NewDict([]Object{
			NewString("name", base.debug),
			NewString("type", base.debug),
			NewString("default", base.debug),
		}, []Object{
			NewString(argType.Name(), base.debug),
			realTyp,
			fallback,
		}, TypeString, TypeAny, base.debug)
		argObjs[i] = dict
	}

	ctx := context.Background()

	fp.SetObject(ctx, "__args__", NewList(argObjs, NewDictType(TypeString, TypeAny), base.debug))

	typ := base.ReturnType
	typs := make([]Object, 0)
	for {
		typs = append(typs, NewString(typ.String(), base.debug))
		if typ.Next == nil {
			break
		}
		typ = typ.Next
	}

	var realTyp Object
	if len(typs) == 1 {
		realTyp = typs[0]
	} else {
		realTyp = NewList(typs, TypeString, base.debug)
	}
	fp.SetObject(ctx, "__returns__", realTyp)

	fp.SetObject(ctx, "init", NewTypedFunction(base.ArgTypes, NewFunctionType(base.ReturnType), func(ctx context.Context, args []Object) (Object, error) {
		return NewTypedFunction(nil, base.ReturnType, func(ctx context.Context, _ []Object) (Object, error) {
			return base.Data(ctx, args)
		}, base.debug), nil
	}, base.debug))

	fp.SetObject(ctx, "call", NewTypedFunction(base.ArgTypes, NewFunctionType(base.ReturnType), func(ctx context.Context, args []Object) (Object, error) {
		return base.Data(ctx, args)
	}, base.debug))

	return fp
}

func (fp *FunctionPrototype) GetObject(ctx context.Context, name string) (Object, bool) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	obj, ok := fp.data[name]
	return obj, ok
}

func (fp *FunctionPrototype) SetObject(ctx context.Context, name string, value Object) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.data[name] = value
	return nil
}

func (fp *FunctionPrototype) Objects() map[string]Object {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	return fp.data
}
