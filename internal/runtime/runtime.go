package runtime

import (
	"sync"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/interpreter"
	"github.com/nubogo/nubo/internal/pubsub"
	"github.com/nubogo/nubo/language"
)

type Runtime struct {
	pubsubProvider pubsub.Provider

	mu           sync.RWMutex
	interpreters map[string]*interpreter.Interpreter
}

func New(pubsubProvider pubsub.Provider) *Runtime {
	return &Runtime{
		pubsubProvider: pubsubProvider,
		interpreters:   make(map[string]*interpreter.Interpreter),
	}
}

func (r *Runtime) Interpret(file string, nodes []*astnode.Node) (language.Object, error) {
	r.mu.RLock()
	if interpreter, ok := r.interpreters[file]; ok {
		r.mu.RUnlock()
		return interpreter.Run(nodes)
	}
	r.mu.RUnlock()

	r.mu.Lock()
	interpreter := interpreter.New(file)
	r.interpreters[file] = interpreter
	r.mu.Unlock()

	return interpreter.Run(nodes)
}
