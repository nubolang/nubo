package runtime

import (
	"sync"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/builtin"
	"github.com/nubolang/nubo/internal/interpreter"
	"github.com/nubolang/nubo/internal/packages"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/packer"
)

type Runtime struct {
	pubsubProvider events.Provider

	mu           sync.RWMutex
	interpreters map[uint]*interpreter.Interpreter
	iid          uint

	builtins map[string]language.Object
	packages map[string]language.Object
	packer   *packer.Packer
}

func New(pubsubProvider events.Provider) *Runtime {
	return &Runtime{
		pubsubProvider: pubsubProvider,
		interpreters:   make(map[uint]*interpreter.Interpreter),
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
	return r.pubsubProvider
}

func (r *Runtime) ProvidePackage(name string, pkg language.Object) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.packages[name] = pkg
}

func (r *Runtime) ImportPackage(name string) (language.Object, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pkg, ok := r.packages[name]
	if ok {
		return pkg, true
	}

	return packages.ImportPackage(name)
}

func (r *Runtime) Interpret(file string, nodes []*astnode.Node) (language.Object, error) {
	interpreter := interpreter.New(file, r, false)

	r.mu.Lock()
	r.interpreters[interpreter.ID] = interpreter
	r.mu.Unlock()

	return interpreter.Run(nodes)
}

func (r *Runtime) NewID() uint {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.iid++
	return r.iid
}

func (r *Runtime) RemoveInterpreter(id uint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.interpreters, id)
}
