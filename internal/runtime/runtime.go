package runtime

import (
	"sync"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/builtin"
	"github.com/nubogo/nubo/internal/interpreter"
	"github.com/nubogo/nubo/internal/pubsub"
	"github.com/nubogo/nubo/language"
)

type Runtime struct {
	pubsubProvider pubsub.Provider

	mu           sync.RWMutex
	interpreters map[string]*interpreter.Interpreter

	builtins map[string]language.Object
}

func New(pubsubProvider pubsub.Provider) *Runtime {
	return &Runtime{
		pubsubProvider: pubsubProvider,
		interpreters:   make(map[string]*interpreter.Interpreter),
		builtins:       builtin.GetBuiltins(),
	}
}

func (r *Runtime) GetBuiltin(name string) (language.Object, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	obj, ok := r.builtins[name]
	return obj, ok
}

func (r *Runtime) Interpret(file string, nodes []*astnode.Node) (language.Object, error) {
	r.mu.RLock()
	if interpreter, ok := r.interpreters[file]; ok {
		r.mu.RUnlock()
		return interpreter.Run(nodes)
	}
	r.mu.RUnlock()

	r.mu.Lock()
	interpreter := interpreter.New(file, r)
	r.interpreters[file] = interpreter
	r.mu.Unlock()

	return interpreter.Run(nodes)
}
