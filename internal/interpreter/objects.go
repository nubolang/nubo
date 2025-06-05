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
	typ     language.ObjectComplexType
	mutable bool
	next    *entry
}

func hashKey(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

func (i *Interpreter) BindObject(name string, value language.Object, mutable bool, declare ...bool) error {
	isDeclare := len(declare) > 0 && declare[0]

	if strings.Contains(name, ".") {
		return i.bindNested(name, value, mutable, isDeclare)
	}

	if isDeclare {
		return i.declareInCurrentScope(name, value, mutable)
	}

	// Try to assign in current scope
	if i.assignInCurrentScope(name, value) == nil {
		return nil
	}

	// If not declared here, try to assign in parent
	if i.scope == ScopeBlock && i.parent != nil {
		if i.isConstInParent(name) {
			return newErr(ErrImmutableVariable, fmt.Sprintf("Cannot reassign to constant %s", name), value.Debug())
		}
		if err := i.parent.BindObject(name, value, mutable, false); err == nil {
			return nil
		}
	}

	// Variable does not exist anywhere
	return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), value.Debug())
}

// Handles dot notation like obj.prop.nested
func (i *Interpreter) bindNested(name string, value language.Object, mutable, isDeclare bool) error {
	parts := strings.Split(name, ".")

	i.mu.RLock()
	for j, part := range parts {
		if imp, ok := i.imports[part]; ok {
			i.mu.RUnlock()
			return imp.BindObject(strings.Join(parts[j:], "."), value, mutable, isDeclare)
		}
	}
	obj, ok := i.objects[hashKey(parts[0])]
	i.mu.RUnlock()
	if !ok {
		return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), value.Debug())
	}

	current := obj.value
	for _, part := range parts[:len(parts)-1] {
		prototype := current.GetPrototype()
		if prototype == nil {
			return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), nil)
		}
		var ok bool
		current, ok = prototype.GetObject(part)
		if !ok {
			return newErr(ErrUndefinedVariable, fmt.Sprintf("Undefined variable %s", name), nil)
		}
	}
	return current.GetPrototype().SetObject(parts[len(parts)-1], value)
}

// Assigns to variable in current scope if it exists
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
			e.value = value
			return nil
		}
	}

	return fmt.Errorf("not found in current scope")
}

// Declares a new variable in current scope
func (i *Interpreter) declareInCurrentScope(name string, value language.Object, mutable bool) error {
	key := hashKey(name)

	i.mu.Lock()
	i.objects[key] = &entry{key: name, value: value, mutable: mutable, next: i.objects[key]}
	i.mu.Unlock()

	return nil
}

// Checks if variable exists as const in parent scope
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
		if len(parts) == 0 {
			return i.parentGetObject(name)
		}

		i.mu.RLock()
		imp, ok := i.imports[parts[0]]
		i.mu.RUnlock()
		if ok {
			return imp.GetObject(strings.Join(parts[1:], "."))
		}

		i.mu.RLock()
		obj, ok := i.objects[hashKey(parts[0])]
		i.mu.RUnlock()
		if !ok || obj == nil || obj.value == nil {
			return i.parentGetObject(name)
		}

		if len(parts) == 1 {
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

			currentObj, ok := prototype.GetObject(part)
			if !ok || currentObj == nil {
				return i.parentGetObject(name)
			}
			current = currentObj
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
