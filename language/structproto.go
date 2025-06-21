package language

import (
	"fmt"
	"sync"
)

type StructPrototype struct {
	base     *Struct
	instance *StructInstance
	data     map[string]Object
	mu       sync.RWMutex
}

func NewStructPrototype(base *Struct) *StructPrototype {
	sp := &StructPrototype{
		base: base,
		data: make(map[string]Object),
	}

	return sp
}

func (sp *StructPrototype) IsBase() bool {
	return sp.instance == nil
}

func (sp *StructPrototype) NewInstance(instance *StructInstance) (*StructPrototype, error) {
	cloned := sp.Clone()
	cloned.instance = instance

	for _, field := range sp.base.Data {
		if err := cloned.SetObject(field.Name, DefaultValue(field.Type)); err != nil {
			return nil, err
		}
	}

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

	for _, field := range s.base.Data {
		if field.Name == name {
			if !field.Type.Compare(value.Type()) {
				return fmt.Errorf("Type mismatch, expected %s, got %s", field.Type, value.Type())
			}
			s.data[name] = value
			return nil
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

	return cloned
}
