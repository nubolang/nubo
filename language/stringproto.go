package language

import (
	"fmt"
	"strings"
	"sync"
)

type StringPrototype struct {
	base *String
	data map[string]Object
	mu   sync.RWMutex
}

func NewStringPrototype(base *String) *StringPrototype {
	sp := &StringPrototype{
		base: base,
		data: make(map[string]Object),
	}

	sp.SetObject("length", NewTypedFunction(nil, TypeInt, func(o []Object) (Object, error) {
		sp.mu.RLock()
		defer sp.mu.RUnlock()

		return NewInt(int64(len(base.Data)), base.debug), nil
	}, nil))

	sp.SetObject("includes", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "substr"}},
		TypeBool,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			substr := o[0].(*String).Data
			return NewBool(strings.Contains(base.Data, substr), base.debug), nil
		}, nil))

	sp.SetObject("indexOf", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "substr"}},
		TypeInt,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			substr := o[0].(*String).Data
			return NewInt(int64(strings.Index(base.Data, substr)), base.debug), nil
		}, nil))

	sp.SetObject("lastIndexOf", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "substr"}},
		TypeInt,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			substr := o[0].(*String).Data
			return NewInt(int64(strings.LastIndex(base.Data, substr)), base.debug), nil
		}, nil))

	sp.SetObject("startsWith", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "prefix"}},
		TypeBool,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			prefix := o[0].(*String).Data
			return NewBool(strings.HasPrefix(base.Data, prefix), base.debug), nil
		}, nil))

	sp.SetObject("endsWith", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "suffix"}},
		TypeBool,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			suffix := o[0].(*String).Data
			return NewBool(strings.HasSuffix(base.Data, suffix), base.debug), nil
		}, nil))

	sp.SetObject("toUpperCase", NewFunction(func(o []Object) (Object, error) {
		sp.mu.RLock()
		defer sp.mu.RUnlock()

		return NewString(strings.ToUpper(base.Data), base.debug), nil
	}, nil))

	sp.SetObject("toLowerCase", NewFunction(func(o []Object) (Object, error) {
		sp.mu.RLock()
		defer sp.mu.RUnlock()

		return NewString(strings.ToLower(base.Data), base.debug), nil
	}, nil))

	sp.SetObject("trim", NewFunction(func(o []Object) (Object, error) {
		sp.mu.RLock()
		defer sp.mu.RUnlock()

		return NewString(strings.TrimSpace(base.Data), base.debug), nil
	}, nil))

	sp.SetObject("trimPrefix", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "prefix"}},
		TypeString,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			prefix := o[0].(*String).Data
			return NewString(strings.TrimPrefix(base.Data, prefix), base.debug), nil
		}, nil))

	sp.SetObject("trimSuffix", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "suffix"}},
		TypeString,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			suffix := o[0].(*String).Data
			return NewString(strings.TrimSuffix(base.Data, suffix), base.debug), nil
		}, nil))

	sp.SetObject("replace", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeString, NameVal: "old"},
			&BasicFnArg{TypeVal: TypeString, NameVal: "new"},
			&BasicFnArg{TypeVal: TypeInt, NameVal: "n"},
		},
		TypeString,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			old := o[0].(*String).Data
			newStr := o[1].(*String).Data
			n := int(o[2].(*Int).Data)
			return NewString(strings.Replace(base.Data, old, newStr, n), base.debug), nil
		}, nil))

	sp.SetObject("split", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeString, NameVal: "sep", DefaultVal: NewString(" ", nil)}},
		TypeList,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			sep := o[0].(*String).Data
			parts := strings.Split(base.Data, sep)
			data := make([]Object, len(parts))
			for i, p := range parts {
				data[i] = NewString(p, base.debug)
			}
			return NewList(data, TypeString, nil), nil
		}, nil))

	sp.SetObject("substring", NewTypedFunction(
		[]FnArg{
			&BasicFnArg{TypeVal: TypeInt, NameVal: "start"},
			&BasicFnArg{TypeVal: TypeInt, NameVal: "end"},
		},
		TypeString,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			start := int(o[0].(*Int).Data)
			end := int(o[1].(*Int).Data)
			if start < 0 || end > len(base.Data) || start > end {
				return nil, fmt.Errorf("invalid substring range")
			}
			return NewString(base.Data[start:end], base.debug), nil
		}, nil))

	sp.SetObject("charAt", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeInt, NameVal: "index"}},
		TypeChar,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			idx := int(o[0].(*Int).Data)
			runes := []rune(base.Data)
			if idx < 0 || idx >= len(runes) {
				return nil, fmt.Errorf("index out of range")
			}
			return NewChar(runes[idx], base.debug), nil
		}, nil))

	sp.SetObject("codePointAt", NewTypedFunction(
		[]FnArg{&BasicFnArg{TypeVal: TypeInt, NameVal: "index"}},
		TypeInt,
		func(o []Object) (Object, error) {
			sp.mu.RLock()
			defer sp.mu.RUnlock()

			idx := int(o[0].(*Int).Data)
			runes := []rune(base.Data)
			if idx < 0 || idx >= len(runes) {
				return nil, fmt.Errorf("index out of range")
			}
			return NewInt(int64(runes[idx]), base.debug), nil
		}, nil))

	return sp
}

func (s *StringPrototype) GetObject(name string) (Object, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, ok := s.data[name]
	return obj, ok
}

func (s *StringPrototype) SetObject(name string, value Object) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[name] = value
	return nil
}

func (s *StringPrototype) Objects() map[string]Object {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}
