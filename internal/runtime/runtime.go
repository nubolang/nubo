package runtime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/builtin"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/interpreter"
	"github.com/nubolang/nubo/internal/packages"
	"github.com/nubolang/nubo/internal/source"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/packer"
	"go.uber.org/zap"
)

// Runtime represents the runtime environment for executing Nubo code.
type Runtime struct {
	pubsubProvider events.Provider

	mu sync.RWMutex

	// iid is the next unique identifier for an interpreter instance.
	iid uint
	// interpreters is a map of interpreter instances.
	interpreters map[uint]*interpreter.Interpreter
	// filemap is a map of file paths to unique interpreter identifiers.
	filemap map[string]uint
	// returnMap is a map of interpreter identifiers to their computed return values if any
	returnMap map[uint]language.Object

	builtins map[string]language.Object
	packages map[string]language.Object
	modules  map[string]*module
	loads    map[*moduleLoad]map[*moduleLoad]int
	packer   *packer.Packer
	source   *source.Loader
	host     *Runtime
	native   language.Object

	ctx context.Context
}

func New(pubsubProvider events.Provider) *Runtime {
	return NewWithFS(pubsubProvider, nil)
}

func NewWithFS(pubsubProvider events.Provider, files fs.FS) *Runtime {
	rt := &Runtime{
		pubsubProvider: pubsubProvider,
		iid:            0,
		interpreters:   make(map[uint]*interpreter.Interpreter),
		filemap:        make(map[string]uint),
		returnMap:      make(map[uint]language.Object),
		builtins:       builtin.GetBuiltins(),
		packages:       make(map[string]language.Object),
		modules:        make(map[string]*module),
		loads:          make(map[*moduleLoad]map[*moduleLoad]int),
		ctx:            context.Background(),
		source:         source.New(files),
	}
	zap.L().Info("runtime.new", zap.Bool("eventsEnabled", pubsubProvider != nil))
	return rt
}

func (r *Runtime) Source() *source.Loader {
	return r.source
}

func (r *Runtime) Context() context.Context {
	return r.ctx
}

func (r *Runtime) WithContext(ctx context.Context) *Runtime {
	r.ctx = ctx
	zap.L().Debug("runtime.context.set")
	return r
}

func (r *Runtime) GetBuiltin(name string) (language.Object, bool) {
	if r.host != nil {
		return r.host.GetBuiltin(name)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	obj, ok := r.builtins[name]
	return obj, ok
}

func (r *Runtime) AddFunction(name string, fn *language.Function) error {
	if fn == nil {
		return fmt.Errorf("function %q is nil", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	switch name {
	case "ref", "_id", "_callsite", "__id__", "__entry__", "__dir__", "__file__", "__concurrent__":
		return fmt.Errorf("function name %q is reserved", name)
	}

	if _, exists := r.builtins[name]; exists {
		return fmt.Errorf("function %q is already registered", name)
	}

	r.builtins[name] = fn
	return nil
}

func (r *Runtime) GetPacker() (*packer.Packer, error) {
	if !r.source.Disk() {
		return nil, fmt.Errorf("the disk package registry is unavailable with a custom source filesystem")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.packer == nil {
		zap.L().Info("runtime.packer.init")
		p, err := packer.New(".")
		if err != nil {
			return nil, err
		}
		r.packer = p
	} else {
		zap.L().Debug("runtime.packer.cached")
	}

	return r.packer, nil
}

func (r *Runtime) GetEventProvider() events.Provider {
	if r.pubsubProvider == nil {
		zap.L().Fatal("runtime.events.disabled")
	}

	return r.pubsubProvider
}

func (r *Runtime) ProvidePackage(name string, pkg language.Object) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.packages[name] = pkg
	zap.L().Debug("runtime.package.provide", zap.String("name", name))
}

func (r *Runtime) AddPackage(name string, pkg language.Object) error {
	if pkg == nil {
		return fmt.Errorf("package %q is nil", name)
	}

	if packages.IsBuiltin(name) {
		return fmt.Errorf("package %q is built-in", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.packages[name]; exists {
		return fmt.Errorf("package %q is already registered", name)
	}
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("package %q is already registered", name)
	}

	r.packages[name] = pkg
	return nil
}

func (r *Runtime) ImportPackage(ctx context.Context, name string, dg *debug.Debug) (language.Object, bool, error) {
	if name == "@native" {
		if r.native == nil {
			return nil, true, fmt.Errorf("@native is only available inside a registered source package")
		}

		return r.native, true, nil
	}

	if r.host != nil {
		return r.host.ImportPackage(ctx, name, dg)
	}

	r.mu.RLock()
	pkg, ok := r.packages[name]
	module := r.modules[name]
	r.mu.RUnlock()

	if ok {
		zap.L().Debug("runtime.package.cached", zap.String("name", name))
		return pkg, true, nil
	}

	if name == "@std" {
		pkg, err := r.importStandard(ctx, dg)
		return pkg, true, err
	}

	if packages.IsBuiltin(name) && !packages.Allowed(strings.TrimPrefix(name, packages.BuiltInModulePrefix)) {
		return nil, false, nil
	}

	if module != nil {
		pkg, err := r.loadModule(ctx, module)
		return pkg, true, err
	}

	if definition, ok := packages.SourcePackage(name); ok {
		pkg, err := r.importSourceBuiltin(ctx, name, definition)
		return pkg, true, err
	}

	zap.L().Debug("runtime.package.import", zap.String("name", name))
	pkg, ok = packages.ImportPackage(name, dg)
	return pkg, ok, nil
}

func (r *Runtime) Interpret(file string, nodes []*astnode.Node) (language.Object, error) {
	zap.L().Info("runtime.interpret.start", zap.String("file", file), zap.Int("nodeCount", len(nodes)))
	wd, err := r.source.Workdir()
	if err != nil {
		zap.L().Error("runtime.interpret.cwd", zap.String("file", file), zap.Error(err))
		return nil, err
	}

	var hasFileInfo bool

	_, err = r.source.Stat(file)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.L().Error("runtime.interpret.stat", zap.String("file", file), zap.Error(err))
			return nil, err
		}
		zap.L().Debug("runtime.interpret.virtualFile", zap.String("file", file))
	} else {
		hasFileInfo = true
	}

	// check if same file already registered
	if hasFileInfo {
		r.mu.RLock()
		for path, id := range r.filemap {
			if r.source.SameFile(path, file) {
				if ret, ok := r.returnMap[id]; ok {
					r.mu.RUnlock()
					zap.L().Info("runtime.interpret.cachedReturn", zap.Uint("id", id), zap.String("file", file))
					return ret, nil
				}
				r.mu.RUnlock()
				zap.L().Debug("runtime.interpret.skip", zap.Uint("id", id), zap.String("file", file))
				return nil, nil
			}
		}
		r.mu.RUnlock()
	}

	interpreter := interpreter.New(r.ctx, file, r, false, wd)
	zap.L().Info("runtime.interpret.spawn", zap.Uint("id", interpreter.ID), zap.String("file", file))

	r.mu.Lock()
	r.interpreters[interpreter.ID] = interpreter
	if hasFileInfo {
		r.filemap[file] = interpreter.ID
	}
	r.mu.Unlock()

	result, runErr := interpreter.Run(nodes)
	if runErr != nil {
		zap.L().Error("runtime.interpret.error", zap.Uint("id", interpreter.ID), zap.String("file", file), zap.Error(runErr))
		return nil, runErr
	}
	if result != nil {
		r.SetInterpreterReturn(interpreter.ID, result)
	}
	zap.L().Info("runtime.interpret.success", zap.Uint("id", interpreter.ID), zap.String("file", file))
	return result, nil
}

func (r *Runtime) NewID() uint {
	if r.host != nil {
		return r.host.NewID()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.iid++
	zap.L().Debug("runtime.interpret.nextID", zap.Uint("next", r.iid))
	return r.iid
}

func (r *Runtime) AddInterpreter(file string, interpreter *interpreter.Interpreter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.interpreters[interpreter.ID] = interpreter
	r.filemap[file] = interpreter.ID
	zap.L().Debug("runtime.interpreter.add", zap.Uint("id", interpreter.ID), zap.String("file", file))
}

func (r *Runtime) RemoveInterpreter(id uint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.interpreters, id)
	delete(r.returnMap, id)

	for file, interpreterID := range r.filemap {
		if interpreterID == id {
			delete(r.filemap, file)
		}
	}

	zap.L().Debug("runtime.interpreter.remove", zap.Uint("id", id))
}

func (r *Runtime) FindInterpreter(file string) (*interpreter.Interpreter, bool) {
	_, err := r.source.Stat(file)
	if err != nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for path, id := range r.filemap {
		instance := r.interpreters[id]
		if instance == nil {
			continue
		}

		if r.source.SameFile(path, file) {
			zap.L().Debug("runtime.interpreter.find", zap.Uint("id", id), zap.String("file", file))
			return instance, true
		}
	}
	zap.L().Debug("runtime.interpreter.miss", zap.String("file", file))
	return nil, false
}

func (r *Runtime) SetInterpreterReturn(id uint, value language.Object) {
	if value == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.returnMap[id] = value
}

func (r *Runtime) GetInterpreterReturn(id uint) (language.Object, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	obj, ok := r.returnMap[id]
	return obj, ok
}
