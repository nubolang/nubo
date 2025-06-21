package interpreter

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/nubolang/nubo/language"
)

type entry struct {
	key     string
	value   language.Object
	typ     *language.Type
	mutable bool
	next    *entry
}

func hashKey(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

func (i *Interpreter) Declare(name string, value language.Object, typ *language.Type, mutable bool) error {
	if strings.Contains(name, ".") {
		return newErr(ErrImmutableVariable, "Cannot declare nested variables", nil)
	}
	return i.declareInCurrentScope(name, value, typ, mutable)
}

func (i *Interpreter) Assign(name string, value language.Object) error {
	if strings.Contains(name, ".") {
		return i.assignNested(name, value)
	}

	assignErr := i.assignInCurrentScope(name, value)
	if assignErr == nil {
		return nil
	}

	if isTypeErr(assignErr) {
		return assignErr
	}

	if i.scope == ScopeBlock && i.parent != nil {
		if i.isConstInParent(name) {
			return newErr(ErrImmutableVariable, fmt.Sprintf("Cannot reassign to constant %s", name), value.Debug())
		}
		return i.parent.Assign(name, value)
	}

	return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), value.Debug())
}

func (i *Interpreter) assignNested(name string, value language.Object) error {
	parts := strings.Split(name, ".")

	i.mu.RLock()
	obj, ok := i.objects[hashKey(parts[0])]
	i.mu.RUnlock()
	if !ok {
		return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), value.Debug())
	}
	parts = parts[1:]

	current := obj.value
	for _, part := range parts[:len(parts)-1] {
		prototype := current.GetPrototype()
		if prototype == nil {
			return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined property %s", part), nil)
		}
		var ok bool
		current, ok = prototype.GetObject(part)
		if !ok {
			return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined property '%s'", part), nil)
		}
	}
	return current.GetPrototype().SetObject(parts[len(parts)-1], value)
}

func (i *Interpreter) assignInCurrentScope(name string, value language.Object) error {
	key := hashKey(name)

	i.mu.RLock()
	head := i.objects[key]
	i.mu.RUnlock()

	for e := head; e != nil; e = e.next {
		if e.key == name {
			if !e.mutable {
				return newErr(ErrImmutableVariable, fmt.Sprintf("Cannot assign to immutable variable %s", name), e.value.Debug())
			}
			if e.typ != nil && !e.typ.Compare(value.Type()) {
				return newErr(ErrTypeMismatch, fmt.Sprintf("Variable \"%s\" type is expected to be %s, got %s", name, e.typ, value.Type()), value.Debug())
			}
			e.value = value
			return nil
		}
	}

	return fmt.Errorf("not found in current scope")
}

func (i *Interpreter) declareInCurrentScope(name string, value language.Object, typ *language.Type, mutable bool) error {
	key := hashKey(name)

	i.mu.Lock()
	i.objects[key] = &entry{key: name, value: value, typ: typ, mutable: mutable, next: i.objects[key]}
	i.mu.Unlock()

	return nil
}

func (i *Interpreter) isConstInParent(name string) bool {
	key := hashKey(name)

	i.parent.mu.RLock()
	defer i.parent.mu.RUnlock()

	head := i.parent.objects[key]
	for e := head; e != nil; e = e.next {
		if e.key == name && !e.mutable {
			return true
		}
	}
	return false
}

func (i *Interpreter) GetObject(name string) (language.Object, bool) {
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")

		i.mu.RLock()
		imp, ok := i.imports[parts[0]]
		i.mu.RUnlock()

		if ok {
			if ob, ok := imp.GetObject(strings.Join(parts[1:], ".")); ok {
				return ob, true
			}
		}

		i.mu.RLock()
		obj, ok := i.objects[hashKey(parts[0])]
		i.mu.RUnlock()
		if !ok || obj == nil || obj.value == nil {
			return i.parentGetObject(name)
		}

		current := obj.value
		for _, part := range parts[1 : len(parts)-1] {
			if current == nil {
				return i.parentGetObject(name)
			}
			prototype := current.GetPrototype()
			if prototype == nil {
				return i.parentGetObject(name)
			}
			var ok bool
			current, ok = prototype.GetObject(part)
			if !ok {
				return i.parentGetObject(name)
			}
		}

		last := parts[len(parts)-1]
		if current == nil || current.GetPrototype() == nil {
			return i.parentGetObject(name)
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

	return i.parentGetObject(name)
}

func (i *Interpreter) parentGetObject(name string) (language.Object, bool) {
	if i.parent == nil {
		return nil, false
	}
	return i.parent.GetObject(name)
}
