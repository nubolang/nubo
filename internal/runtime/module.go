package runtime

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/interpreter"
	"github.com/nubolang/nubo/internal/packages"
	"github.com/nubolang/nubo/internal/source"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

type module struct {
	name   string
	source *source.Loader
	nodes  []*astnode.Node
	native language.Object
	value  language.Object
	load   *moduleLoad
}

type moduleLoad struct {
	module *module
	done   chan struct{}
	value  language.Object
	err    error
}

type moduleFrameKey struct{}

type moduleFrame struct {
	load   *moduleLoad
	parent *moduleFrame
	active atomic.Bool
}

func (r *Runtime) AddSourcePackage(name string, native language.Object, code string, files fs.FS) error {
	if packages.IsBuiltin(name) {
		return fmt.Errorf("package %q is built-in", name)
	}

	mod, err := newModule(name, native, code, files)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.packages[name]; exists {
		return fmt.Errorf("package %q is already registered", name)
	}
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("package %q is already registered", name)
	}

	r.modules[name] = mod
	return nil
}

func newModule(name string, native language.Object, code string, files fs.FS) (*module, error) {
	if native == nil {
		return nil, fmt.Errorf("package %q has no native bindings", name)
	}

	loader := source.NewModule(name, code, files)
	nodes, err := loader.Nodes("package.nubo")
	if err != nil {
		return nil, fmt.Errorf("cannot register source package %q: %w", name, err)
	}

	proto := native.GetPrototype()
	if proto == nil {
		return nil, fmt.Errorf("package %q has no native namespace", name)
	}
	if _, exists := proto.GetObject(context.Background(), "GoHandle"); exists {
		return nil, fmt.Errorf("native binding GoHandle is reserved")
	}
	if err := proto.SetObject(context.Background(), "GoHandle", language.NewTypeObject(language.TypeGoHandle, nil)); err != nil {
		return nil, err
	}

	return &module{name: name, source: loader, nodes: nodes, native: native}, nil
}

func (r *Runtime) loadModule(ctx context.Context, mod *module) (value language.Object, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	head, _ := ctx.Value(moduleFrameKey{}).(*moduleFrame)
	var owner *moduleFrame
	for frame := head; frame != nil; frame = frame.parent {
		if frame.active.Load() {
			owner = frame
			break
		}
	}

	r.mu.Lock()
	if mod.value != nil {
		r.mu.Unlock()
		return mod.value, nil
	}

	pending := mod.load
	initialize := pending == nil
	if initialize {
		pending = &moduleLoad{module: mod, done: make(chan struct{})}
		mod.load = pending
	}

	if owner != nil && owner.active.Load() {
		chain := r.modulePath(pending, owner.load, make(map[*moduleLoad]bool))
		if len(chain) != 0 {
			r.mu.Unlock()
			chain = append(chain, mod.name)
			return nil, fmt.Errorf("Circular package import: %s", strings.Join(chain, " → "))
		}

		if r.loads[owner.load] == nil {
			r.loads[owner.load] = make(map[*moduleLoad]int)
		}
		r.loads[owner.load][pending]++
		defer r.finishModuleImport(owner.load, pending)
	}

	if !initialize {
		r.mu.Unlock()

		select {
		case <-pending.done:
			return pending.value, pending.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	r.mu.Unlock()

	frame := &moduleFrame{load: pending, parent: head}
	frame.active.Store(true)
	defer func() {
		if recovered := recover(); recovered != nil {
			value = nil
			err = fmt.Errorf("package %q initialization panicked: %v", mod.name, recovered)
		}

		r.mu.Lock()
		frame.active.Store(false)
		pending.value, pending.err = value, err
		if err == nil {
			mod.value = value
		}
		mod.load = nil
		close(pending.done)
		r.mu.Unlock()
	}()

	return r.runModule(context.WithValue(ctx, moduleFrameKey{}, frame), mod)
}

func (r *Runtime) finishModuleImport(owner, target *moduleLoad) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.loads[owner][target]--
	if r.loads[owner][target] == 0 {
		delete(r.loads[owner], target)
	}
	if len(r.loads[owner]) == 0 {
		delete(r.loads, owner)
	}
}

// Called while the registry is locked.
func (r *Runtime) modulePath(from, target *moduleLoad, visited map[*moduleLoad]bool) []string {
	if from == target {
		return []string{from.module.name}
	}
	if visited[from] {
		return nil
	}
	visited[from] = true

	children := make([]*moduleLoad, 0, len(r.loads[from]))
	for child := range r.loads[from] {
		children = append(children, child)
	}
	slices.SortFunc(children, func(a, b *moduleLoad) int { return strings.Compare(a.module.name, b.module.name) })

	for _, child := range children {
		if chain := r.modulePath(child, target, visited); len(chain) != 0 {
			return append([]string{from.module.name}, chain...)
		}
	}

	return nil
}

func (r *Runtime) runModule(ctx context.Context, mod *module) (value language.Object, err error) {
	var instance *interpreter.Interpreter
	defer func() {
		if recovered := recover(); recovered != nil {
			value = nil
			err = fmt.Errorf("package %q initialization panicked: %v", mod.name, recovered)
		}

		if err != nil && instance != nil {
			instance.MustDetach()
		}
	}()

	run := &Runtime{
		pubsubProvider: r.pubsubProvider,
		interpreters:   make(map[uint]*interpreter.Interpreter),
		filemap:        make(map[string]uint),
		returnMap:      make(map[uint]language.Object),
		source:         mod.source,
		host:           r,
		native:         mod.native,
		ctx:            ctx,
	}
	instance = interpreter.New(ctx, "package.nubo", run, true, ".")

	value, err = instance.Run(mod.nodes)
	if err != nil {
		return nil, err
	}
	if value != nil {
		return value, nil
	}

	pkg := n.NewPackage(strings.TrimPrefix(mod.name, "@std/"), nil)
	proto := pkg.GetPrototype()
	for name, object := range instance.Exports() {
		if err := proto.SetObject(ctx, name, object); err != nil {
			return nil, err
		}
	}

	return pkg, nil
}
