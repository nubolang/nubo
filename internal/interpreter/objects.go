package interpreter

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

const maxScopeDepth = 64

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
	zap.L().Debug("interpreter.objects.declare", zap.Uint("id", i.ID), zap.String("name", name), zap.Bool("mutable", mutable))

	if strings.Contains(name, ".") {
		return runExc("cannot declare nested variables").WithDebug(value.Debug())
	}
	return i.declareInCurrentScope(name, value, typ, mutable)
}

func (i *Interpreter) Assign(name string, value language.Object) error {
	zap.L().Debug("interpreter.objects.assign", zap.Uint("id", i.ID), zap.String("name", name))

	if strings.Contains(name, ".") {
		return i.assignNested(name, value, 0)
	}

	assignErr := i.assignInCurrentScope(name, value)
	if assignErr == nil {
		return nil
	}

	if isTypeErr(assignErr) {
		return assignErr
	}

	if (i.scope == ScopeBlock || i.scope == ScopeFunction) && i.parent != nil {
		if i.isConstInParent(name) {
			return runExc("cannot reassign constant %q", name).WithDebug(value.Debug())
		}
		return i.assignViaParent(name, value, 0)
	}

	return wrapRunExc(assignErr, value.Debug())
}

// assignViaParent walks the parent chain with a depth guard.
func (i *Interpreter) assignViaParent(name string, value language.Object, depth int) error {
	if depth > maxScopeDepth {
		return runExc("scope depth limit exceeded while assigning %q", name).WithDebug(value.Debug())
	}
	if i.parent == nil {
		return runExc("assignment: %q not found in any parent scope", name).WithDebug(value.Debug())
	}

	err := i.parent.assignInCurrentScope(name, value)
	if err == nil {
		return nil
	}
	if isTypeErr(err) {
		return err
	}

	if (i.parent.scope == ScopeBlock || i.parent.scope == ScopeFunction) && i.parent.parent != nil {
		if i.parent.isConstInParent(name) {
			return runExc("cannot reassign constant %q", name).WithDebug(value.Debug())
		}
		return i.parent.assignViaParent(name, value, depth+1)
	}

	return wrapRunExc(err, value.Debug())
}

func (i *Interpreter) assignNested(name string, value language.Object, depth int) error {
	if depth > maxScopeDepth {
		return runExc("scope depth limit exceeded while assigning nested %q", name).WithDebug(value.Debug())
	}
	zap.L().Debug("interpreter.objects.assignNested", zap.Uint("id", i.ID), zap.String("name", name))

	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return runExc("invalid nested name %q", name).WithDebug(value.Debug())
	}

	i.mu.RLock()
	obj, ok := i.objects[hashKey(parts[0])]
	i.mu.RUnlock()
	if !ok || obj.value == nil {
		if i.parent != nil {
			return i.parent.assignNested(name, value, depth+1)
		}
		return runExc("undefined variable %q", parts[0]).WithDebug(value.Debug())
	}

	current := obj.value
	for _, part := range parts[1 : len(parts)-1] {
		proto := current.GetPrototype()
		if proto == nil {
			return runExc("undefined property %q", part).WithDebug(value.Debug())
		}
		var ok bool
		current, ok = proto.GetObject(i.ctx, part)
		if !ok || current == nil {
			return runExc("undefined property %q", part).WithDebug(value.Debug())
		}
	}

	lastKey := parts[len(parts)-1]
	proto := current.GetPrototype()
	if proto == nil {
		return runExc("no prototype for %q", name).WithDebug(value.Debug())
	}

	if setFn, ok := proto.GetObject(i.ctx, "__set__"); ok {
		if callErr := i.callSetFunction(setFn, lastKey, value); callErr == nil {
			zap.L().Debug("interpreter.objects.assignNested.prototype", zap.Uint("id", i.ID), zap.String("name", name))
			return nil
		}
	}

	if err := proto.SetObject(i.ctx, lastKey, value); err == nil {
		zap.L().Debug("interpreter.objects.assignNested.prototype", zap.Uint("id", i.ID), zap.String("name", name))
		return nil
	}

	return runExc("failed to assign %q", name).WithDebug(value.Debug())
}

func (i *Interpreter) assignInCurrentScope(name string, value language.Object) error {
	zap.L().Debug("interpreter.objects.assignInScope", zap.Uint("id", i.ID), zap.String("name", name))

	key := hashKey(name)

	i.mu.RLock()
	head := i.objects[key]
	i.mu.RUnlock()

	for e := head; e != nil; e = e.next {
		if e.key == name {
			if !e.mutable {
				return runExc("cannot assign to immutable variable %q", name).WithDebug(value.Debug())
			}
			if e.typ != nil && !e.typ.Compare(value.Type()) {
				return typeError("variable %q type is expected to be %s, got %s", name, e.typ, value.Type()).WithDebug(value.Debug())
			}
			e.value = value
			return nil
		}
	}

	return runExc("assigment: not found in current scope").WithDebug(value.Debug())
}

func (i *Interpreter) declareInCurrentScope(name string, value language.Object, typ *language.Type, mutable bool) error {
	zap.L().Debug("interpreter.objects.declareInScope", zap.Uint("id", i.ID), zap.String("name", name))

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
	return i.getObject(name, 0)
}

func (i *Interpreter) getObject(name string, depth int) (language.Object, bool) {
	if depth > maxScopeDepth {
		return nil, false
	}

	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")

		i.mu.RLock()
		imp, ok := i.imports[parts[0]]
		i.mu.RUnlock()
		if ok {
			if ob, ok := imp.getObject(strings.Join(parts[1:], "."), depth+1); ok {
				return ob, true
			}
		}

		i.mu.RLock()
		obj, ok := i.objects[hashKey(parts[0])]
		i.mu.RUnlock()
		if !ok || obj == nil || obj.value == nil {
			return i.parentGetObject(name, depth+1)
		}

		current := obj.value
		for _, part := range parts[1 : len(parts)-1] {
			if current == nil {
				return i.parentGetObject(name, depth+1)
			}
			proto := current.GetPrototype()
			if proto == nil {
				return i.parentGetObject(name, depth+1)
			}
			var ok bool
			current, ok = proto.GetObject(i.ctx, part)
			if !ok {
				return i.parentGetObject(name, depth+1)
			}
		}

		last := parts[len(parts)-1]
		if current == nil {
			return i.parentGetObject(name, depth+1)
		}

		proto := current.GetPrototype()
		if proto == nil {
			return i.parentGetObject(name, depth+1)
		}

		val, ok := proto.GetObject(i.ctx, last)
		if ok {
			return val, true
		}

		if getFn, ok := proto.GetObject(i.ctx, "__get__"); ok {
			if res, err := i.callGetFunction(getFn, last); err == nil {
				return res, true
			}
		}

		return i.parentGetObject(name, depth+1)
	}

	// normal (non-nested) lookup
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

	return i.parentGetObject(name, depth+1)
}

func (i *Interpreter) parentGetObject(name string, depth int) (language.Object, bool) {
	if depth > maxScopeDepth {
		return nil, false
	}

	if i.parent == nil {
		for _, inc := range i.includes {
			if obj, ok := inc.getObject(name, depth+1); ok {
				return obj, true
			}
		}
		return nil, false
	}

	return i.parent.getObject(name, depth+1)
}

func (i *Interpreter) callGetFunction(fn language.Object, key string) (language.Object, error) {
	callable, ok := fn.(*language.Function)
	if !ok {
		return nil, fmt.Errorf("not callable")
	}
	args := []language.Object{language.NewString(key, fn.Debug())}
	return callable.Data(i.ctx, args)
}

func (i *Interpreter) callSetFunction(fn language.Object, key string, value language.Object) error {
	callable, ok := fn.(*language.Function)
	if !ok {
		return fmt.Errorf("not callable")
	}
	args := []language.Object{language.NewString(key, fn.Debug()), value}
	_, err := callable.Data(i.ctx, args)
	return err
}
