package language

import (
	"slices"
	"sync"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/stoewer/go-strcase"
)

type ElementPrototype struct {
	base *Element
	data map[string]Object
	mu   sync.RWMutex
}

func NewElementPrototype(base *Element) *ElementPrototype {
	ep := &ElementPrototype{
		base: base,
		data: make(map[string]Object),
	}

	setAttr := NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "name"}, &BasicFnArg{TypeVal: TypeAny, NameVal: "value"}},
		TypeVoid,
		func(o []Object) (Object, error) {
			ep.mu.Lock()
			defer ep.mu.Unlock()

			val := o[1]

			attr := Attribute{
				Name: strcase.KebabCase(o[0].String()),
			}

			if val.Type() == TypeString {
				attr.Kind = "TEXT"
				attr.Value = NewString(val.String(), val.Debug())
			} else {
				attr.Kind = "DYNAMIC"
				attr.Value = val.Clone()
			}

			// overwrite if exists
			found := false
			for i, a := range base.Data.Args {
				if a.Name == attr.Name {
					base.Data.Args[i] = attr
					found = true
					break
				}
			}
			if !found {
				base.Data.Args = append(base.Data.Args, attr)
			}

			return nil, nil
		}, base.debug)
	ep.SetObject("setAttribute", setAttr)
	ep.SetObject("__set__", setAttr)

	ep.SetObject("removeAttribute", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeString, NameVal: "name"},
		},
		TypeVoid,
		func(o []Object) (Object, error) {
			ep.mu.Lock()
			defer ep.mu.Unlock()

			name := strcase.KebabCase(o[0].(*String).Data)

			for i, a := range base.Data.Args {
				if a.Name == name {
					base.Data.Args = slices.Delete(base.Data.Args, i, i+1)
					break
				}
			}

			return nil, nil
		},
		base.debug,
	))

	getAttr := NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeString, NameVal: "name"},
		},
		TypeAny,
		func(o []Object) (Object, error) {
			ep.mu.RLock()
			defer ep.mu.RUnlock()

			name := strcase.KebabCase(o[0].(*String).Data)
			for _, a := range base.Data.Args {
				if a.Name == name {
					return a.Value, nil
				}
			}
			return Nil, nil
		},
		base.debug,
	)
	ep.SetObject("getAttribute", getAttr)
	ep.SetObject("__get__", getAttr)

	ep.SetObject("hasAttribute", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeString, NameVal: "name"},
		},
		TypeBool,
		func(o []Object) (Object, error) {
			ep.mu.RLock()
			defer ep.mu.RUnlock()

			name := strcase.KebabCase(o[0].(*String).Data)
			for _, a := range base.Data.Args {
				if a.Name == name {
					return NewBool(true, o[0].Debug()), nil
				}
			}
			return NewBool(false, o[0].Debug()), nil
		},
		base.debug,
	))

	ep.SetObject("children", NewTypedFunction(nil, NewListType(NewUnionType(TypeHtml, TypeString)), func(o []Object) (Object, error) {
		ep.mu.RLock()
		defer ep.mu.RUnlock()

		var objs = make([]Object, len(base.Data.Children))
		for i, child := range base.Data.Children {
			if child.Type == astnode.NodeTypeElementRawText {
				objs[i] = NewString(child.Content, base.debug)
			} else {
				objs[i] = child.Value
			}
		}

		return NewList(objs, NewUnionType(TypeHtml, TypeString), base.debug), nil
	}, ep.base.debug))

	ep.SetObject("setChildren", NewTypedFunction([]FnArg{
		&BasicFnArg{
			NameVal: "children",
			TypeVal: NewListType(NewUnionType(TypeHtml, TypeString)),
		},
	}, TypeVoid, func(o []Object) (Object, error) {
		ep.mu.Lock()
		defer ep.mu.Unlock()

		data := o[0].(*List).Data
		base.Data.Children = make([]ElementChild, len(data))

		for i, child := range data {
			if child.Type().Compare(TypeString) {
				base.Data.Children[i] = ElementChild{
					Type:    astnode.NodeTypeElementRawText,
					Content: child.String(),
				}
			} else {
				base.Data.Children[i] = ElementChild{
					Type:  astnode.NodeTypeElement,
					Value: child.Value().(*Element),
				}
			}
		}

		return nil, nil
	}, ep.base.debug))

	return ep
}

func (e *ElementPrototype) GetObject(name string) (Object, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	obj, ok := e.data[name]
	return obj, ok
}

func (e *ElementPrototype) SetObject(name string, value Object) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.data[name] = value
	return nil
}

func (e *ElementPrototype) Objects() map[string]Object {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.data
}
