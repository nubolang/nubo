package runtime

import (
	"log"
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
}

func New(pubsubProvider events.Provider) *Runtime {
	return &Runtime{
		pubsubProvider: pubsubProvider,
		iid:            0,
		interpreters:   make(map[uint]*interpreter.Interpreter),
		filemap:        make(map[string]uint),
		returnMap:      make(map[uint]language.Object),
		builtins:       builtin.GetBuiltins(),
		packages:       make(map[string]language.Object),
	}
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
		p, err := packer.New(".")
		if err != nil {
			return nil, err
		}
		r.packer = p
	}

	return r.packer, nil
}

func (r *Runtime) GetEventProvider() events.Provider {
	if r.pubsubProvider == nil {
		log.Fatal("Event provider is disabled in nubo's configuration file. Enable it to use event functions.")
	}

	return r.pubsubProvider
}

func (r *Runtime) ProvidePackage(name string, pkg language.Object) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.packages[name] = pkg
}

func (r *Runtime) ImportPackage(name string, dg *debug.Debug) (language.Object, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pkg, ok := r.packages[name]
	if ok {
		return pkg, true
	}

	return packages.ImportPackage(name, dg)
}

func (r *Runtime) Interpret(file string, nodes []*astnode.Node) (language.Object, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(file)
	if err != nil {
		return nil, err
	}

	// check if same file already registered
	r.mu.RLock()
	for path, id := range r.filemap {
		existingInfo, err := os.Stat(path)
		if err == nil && os.SameFile(existingInfo, info) {
			if ret, ok := r.returnMap[id]; ok {
				r.mu.RUnlock()
				return ret, nil
			}
			r.mu.RUnlock()
			return nil, nil
		}
	}
	r.mu.RUnlock()

	interpreter := interpreter.New(file, r, false, wd)

	r.mu.Lock()
	r.interpreters[interpreter.ID] = interpreter
	r.filemap[file] = interpreter.ID
	r.mu.Unlock()

	return interpreter.Run(nodes)
}

func (r *Runtime) NewID() uint {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.iid++
	return r.iid
}

func (r *Runtime) AddInterpreter(file string, interpreter *interpreter.Interpreter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.interpreters[interpreter.ID] = interpreter
	r.filemap[file] = interpreter.ID
}

func (r *Runtime) RemoveInterpreter(id uint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.interpreters, id)
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
			return r.interpreters[id], true
		}
	}
	return nil, false
}
