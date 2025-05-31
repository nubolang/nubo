package language

import (
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

	sp.SetObject("length", NewFunction(func(o []Object) (Object, error) {
		return NewInt(int64(len(base.Data)), base.debug), nil
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
