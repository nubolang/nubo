package interpreter

import (
	"fmt"
	"hash/fnv"
	"strings"

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
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		i.mu.RLock()

		for j, part := range parts {
			if imp, ok := i.imports[part]; ok {
				i.mu.RUnlock()
				return imp.BindObject(strings.Join(parts[j:], "."), value, mutable)
			}
		}

		obj, ok := i.objects[hashKey(parts[0])]
		i.mu.RUnlock()

		if ok {
			current := obj.value
			for _, part := range parts[:len(parts)-1] {
				prototype := current.GetPrototype()
				if prototype == nil {
					return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), nil)
				}

				current, ok = prototype.GetObject(part)
				if !ok {
					return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), nil)
				}
			}
			return current.GetPrototype().SetObject(parts[len(parts)], value)
		}

		return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), value.Debug())
	}

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
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) == 0 {
			return nil, false
		}

		i.mu.RLock()
		for j, part := range parts {
			if imp, ok := i.imports[part]; ok {
				i.mu.RUnlock()
				return imp.GetObject(strings.Join(parts[j:], "."))
			}
		}

		obj, ok := i.objects[hashKey(parts[0])]
		i.mu.RUnlock()
		if !ok || obj == nil || obj.value == nil {
			return nil, false
		}

		if len(parts) == 1 {
			return obj.value, true
		}

		current := obj.value
		for _, part := range parts[1 : len(parts)-1] {
			if current == nil {
				return nil, false
			}
			prototype := current.GetPrototype()
			if prototype == nil {
				return nil, false
			}

			currentObj, ok := prototype.GetObject(part)
			if !ok || currentObj == nil {
				return nil, false
			}
			current = currentObj
		}

		last := parts[len(parts)-1]
		if current == nil || current.GetPrototype() == nil {
			return nil, false
		}
		return current.GetPrototype().GetObject(last)
	}

	if obj, ok := i.runtime.GetBuiltin(name); ok {
		return obj, true
	}

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
