package runtime

import (
	"context"
	"os"
	"sync"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/builtin"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/interpreter"
	"github.com/nubolang/nubo/internal/packages"
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
	packer   *packer.Packer

	ctx context.Context
}

func New(pubsubProvider events.Provider) *Runtime {
	rt := &Runtime{
		pubsubProvider: pubsubProvider,
		iid:            0,
		interpreters:   make(map[uint]*interpreter.Interpreter),
		filemap:        make(map[string]uint),
		returnMap:      make(map[uint]language.Object),
		builtins:       builtin.GetBuiltins(),
		packages:       make(map[string]language.Object),
		ctx:            context.Background(),
	}
	zap.L().Info("runtime.new", zap.Bool("eventsEnabled", pubsubProvider != nil))
	return rt
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	obj, ok := r.builtins[name]
	return obj, ok
}

func (r *Runtime) GetPacker() (*packer.Packer, error) {
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

func (r *Runtime) ImportPackage(name string, dg *debug.Debug) (language.Object, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pkg, ok := r.packages[name]
	if ok {
		zap.L().Debug("runtime.package.cached", zap.String("name", name))
		return pkg, true
	}

	zap.L().Debug("runtime.package.import", zap.String("name", name))
	return packages.ImportPackage(name, dg)
}

func (r *Runtime) Interpret(file string, nodes []*astnode.Node) (language.Object, error) {
	zap.L().Info("runtime.interpret.start", zap.String("file", file), zap.Int("nodeCount", len(nodes)))
	wd, err := os.Getwd()
	if err != nil {
		zap.L().Error("runtime.interpret.cwd", zap.String("file", file), zap.Error(err))
		return nil, err
	}

	info, err := os.Stat(file)
	if err != nil {
		zap.L().Error("runtime.interpret.stat", zap.String("file", file), zap.Error(err))
		return nil, err
	}

	// check if same file already registered
	r.mu.RLock()
	for path, id := range r.filemap {
		existingInfo, err := os.Stat(path)
		if err == nil && os.SameFile(existingInfo, info) {
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

	interpreter := interpreter.New(r.ctx, file, r, false, wd)
	zap.L().Info("runtime.interpret.spawn", zap.Uint("id", interpreter.ID), zap.String("file", file))

	r.mu.Lock()
	r.interpreters[interpreter.ID] = interpreter
	r.filemap[file] = interpreter.ID
	r.mu.Unlock()

	result, runErr := interpreter.Run(nodes)
	if runErr != nil {
		zap.L().Error("runtime.interpret.error", zap.Uint("id", interpreter.ID), zap.String("file", file), zap.Error(runErr))
		return nil, runErr
	}
	zap.L().Info("runtime.interpret.success", zap.Uint("id", interpreter.ID), zap.String("file", file))
	return result, nil
}

func (r *Runtime) NewID() uint {
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
	zap.L().Debug("runtime.interpreter.remove", zap.Uint("id", id))
}

func (r *Runtime) FindInterpreter(file string) (*interpreter.Interpreter, bool) {
	info, err := os.Stat(file)
	if err != nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for path, id := range r.filemap {
		existingInfo, err := os.Stat(path)
		if err == nil && os.SameFile(existingInfo, info) {
			zap.L().Debug("runtime.interpreter.find", zap.Uint("id", id), zap.String("file", file))
			return r.interpreters[id], true
		}
	}
	zap.L().Debug("runtime.interpreter.miss", zap.String("file", file))
	return nil, false
}
