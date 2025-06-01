package language

import (
	"sync"
)

type StructPrototype struct {
	base *Struct
	data map[string]Object
	mu   sync.RWMutex
}

func NewStructPrototype(base *Struct) *StructPrototype {
	sp := &StructPrototype{
		base: base,
		data: make(map[string]Object),
	}

	return sp
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
	s.data[name] = value
	return nil
}

func (s *StructPrototype) Objects() map[string]Object {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}
