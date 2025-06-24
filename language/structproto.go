package language

import (
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
	mu   sync.RWMutex
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

	for _, field := range sp.base.Data {
		if err := cloned.SetObject(field.Name, DefaultValue(field.Type)); err != nil {
			return nil, err
		}
	}

	for name, set := range sp.setters {
		if err := cloned.SetObject(name, set); err != nil {
			return nil, err
		}
	}

	cloned.Lock()
	return cloned, nil
}

func (s *StructPrototype) GetObject(name string) (Object, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, ok := s.data[name]
	return obj, ok
}

func (s *StructPrototype) SetObject(name string, value Object) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.instance == nil {
		s.setters[name] = value
		return nil
	}

	for _, field := range s.base.Data {
		if field.Name == name {
			if !field.Type.Compare(value.Type()) {
				return fmt.Errorf("Type mismatch, expected %s, got %s", field.Type, value.Type())
			}
			s.data[name] = value
			return nil
		}
	}

	if s.locked && s.instance != nil {
		return fmt.Errorf("Cannot use struct prototype directly, add a field or implement it via the impl block")
	}

	if value.Type().Base() == ObjectTypeFunction {
		fn, ok := value.(*Function)
		if !ok {
			return fmt.Errorf("Expected function, got %s", value.Type())
		}

		if len(fn.ArgTypes) > 0 {
			if fn.ArgTypes[0].Type().Base() != ObjectTypeAny && fn.ArgTypes[0].Type().Compare(s.base.structType) {
				newFn := NewTypedFunction(fn.ArgTypes[1:], fn.ReturnType, func(o []Object) (Object, error) {
					objs := make([]Object, 0, len(o)+1)
					objs = append(objs, s.instance)
					for _, obj := range o {
						objs = append(objs, obj)
					}
					return fn.Data(objs)
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
