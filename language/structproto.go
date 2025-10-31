package language

import (
	"context"
	"fmt"
	"sync"
)

type StructPrototype struct {
	base     *Struct
	instance *StructInstance

	setters     map[string]Object
	implemented bool
	locked      bool

	data map[string]Object

	mu sync.RWMutex
}

func NewStructPrototype(base *Struct) *StructPrototype {
	sp := &StructPrototype{
		base:    base,
		setters: make(map[string]Object),
		data:    make(map[string]Object),
		locked:  true,
	}

	return sp
}

func (sp *StructPrototype) IsBase() bool {
	return sp.instance == nil
}

func (sp *StructPrototype) NewInstance(instance *StructInstance) (*StructPrototype, error) {
	cloned := sp.Clone()
	cloned.instance = instance
	cloned.Unlock()

	ctx := StructAllowPrivateCtx(context.Background())
	for _, field := range sp.base.Data {
		if err := cloned.SetObject(ctx, field.Name, DefaultValue(field.Type)); err != nil {
			return nil, err
		}
	}

	for name, set := range sp.setters {
		if err := cloned.SetObject(ctx, name, set); err != nil {
			return nil, err
		}
	}

	err := cloned.SetObject(ctx, "$convout", NewTypedFunction([]FnArg{
		&BasicFnArg{NameVal: "self", TypeVal: sp.base.Type()},
	}, TypeAny, func(ctx context.Context, o []Object) (Object, error) {
		keys := make([]Object, 0, len(sp.base.Data)-len(sp.base.privateMap))
		values := make([]Object, 0, len(sp.base.Data)-len(sp.base.privateMap))

		for _, field := range sp.base.Data {
			if field.Private {
				continue
			}
			value, ok := cloned.GetObject(context.Background(), field.Name)
			keys = append(keys, NewString(field.Name, value.Debug()))
			if !ok {
				return nil, fmt.Errorf("value not found for key: %s", field.Name)
			}
			values = append(values, value)
		}

		return NewDict(keys, values, TypeString, TypeAny, sp.base.Debug())
	}, instance.Debug()))
	if err != nil {
		return nil, err
	}

	cloned.Lock()
	return cloned, nil
}

func (s *StructPrototype) GetObject(ctx context.Context, name string) (Object, bool) {
	if _, ok := s.base.privateMap[name]; ok {
		if val := ctx.Value("unlock_struct_private"); val == nil || val != true {
			return nil, false
		}
	}

	obj, ok := s.data[name]
	return obj, ok
}

func (s *StructPrototype) SetObject(ctx context.Context, name string, value Object) error {
	if _, ok := s.base.privateMap[name]; ok {
		if val := ctx.Value("unlock_struct_private"); val == nil || val != true {
			return fmt.Errorf("cannot modify private field outside implementation")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.instance == nil {
		if val := ctx.Value("set_private"); val != nil && val == true {
			s.base.privateMap[name] = struct{}{}
		}
		s.setters[name] = value
		return nil
	}

	for _, field := range s.base.Data {
		if field.Name == name {
			if !field.Type.Compare(value.Type()) {
				return fmt.Errorf("type mismatch, expected %s, got %s", field.Type, value.Type())
			}
			s.data[name] = value
			return nil
		}
	}

	if s.locked && s.instance != nil {
		return fmt.Errorf("cannot use struct prototype directly, add a field or implement it via the impl block")
	}

	if value.Type().Base() == ObjectTypeFunction {
		fn, ok := value.(*Function)
		if !ok {
			return fmt.Errorf("expected function, got %s", value.Type())
		}

		if len(fn.ArgTypes) > 0 {
			if fn.ArgTypes[0].Type().Base() != ObjectTypeAny && fn.ArgTypes[0].Type().Compare(s.base.structType) {
				newFn := NewTypedFunction(fn.ArgTypes[1:], fn.ReturnType, func(ctx context.Context, o []Object) (Object, error) {
					objs := make([]Object, 0, len(o)+1)
					objs = append(objs, s.instance)
					for _, obj := range o {
						objs = append(objs, obj)
					}
					return fn.Data(StructAllowPrivateCtx(ctx), objs)
				}, fn.Debug())

				s.data[name] = newFn
				return nil
			}
		}
	}

	s.data[name] = value
	return nil
}

func (s *StructPrototype) Objects() map[string]Object {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *StructPrototype) Clone() *StructPrototype {
	// Struct instances are not cloneable
	if s.instance != nil {
		return s
	}

	cloned := &StructPrototype{
		base: s.base,
	}

	data := make(map[string]Object)
	for k, v := range s.data {
		data[k] = v.Clone()
	}
	cloned.data = data
	cloned.locked = true

	return cloned
}

func (s *StructPrototype) Implement() {
	s.implemented = true
}

func (s *StructPrototype) Implemented() bool {
	return s.implemented
}

func (s *StructPrototype) Lock() {
	s.locked = true
}

func (s *StructPrototype) Unlock() {
	s.locked = false
}

func StructAllowPrivateCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, "unlock_struct_private", true)
}

func StructSetPrivate(ctx context.Context) context.Context {
	return context.WithValue(ctx, "set_private", true)
}
