package interpreter

import (
	"fmt"
	"hash/fnv"

	"github.com/nubogo/nubo/language"
)

type entry struct {
	key     string
	value   language.Object
	mutable bool
	next    *entry
}

func hashKey(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

func (i *Interpreter) BindObject(name string, value language.Object, mutable bool) error {
	key := hashKey(name)
	i.mu.RLock()
	head := i.objects[key]
	i.mu.RUnlock()

	// Loop through the chain if it exists
	for e := head; e != nil; e = e.next {
		if e.key == name {
			if !e.mutable {
				return newErr(ErrImmutableVariable, fmt.Sprintf("Cannot assign to immutable variable %s", name), e.value.Debug())
			}
			e.value = value
			return nil
		}
	}

	// Add new entry to the beginning of the chain
	i.mu.Lock()
	i.objects[key] = &entry{key: name, value: value, mutable: mutable, next: head}
	i.mu.Unlock()

	return nil
}

func (i *Interpreter) GetObject(name string) (language.Object, bool) {
	key := hashKey(name)
	i.mu.RLock()
	head := i.objects[key]
	i.mu.RUnlock()

	for e := head; e != nil; e = e.next {
		if e.key == name {
			return e.value, true
		}
	}

	return nil, false
}
