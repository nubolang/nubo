package interpreter

import (
	"sync"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/pubsub"
	"github.com/nubolang/nubo/language"
)

type Runtime interface {
	GetBuiltin(name string) (language.Object, bool)
	GetEventProvider() pubsub.Provider
	NewID() uint
	RemoveInterpreter(id uint)
}

type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeFunction
	ScopeBlock
)

type Interpreter struct {
	ID          uint
	currentFile string
	dependent   bool

	scope  Scope
	parent *Interpreter

	runtime Runtime
	objects map[uint32]*entry
	imports map[string]*Interpreter
	unsub   []pubsub.UnsubscribeFunc

	mu sync.RWMutex
}

func New(currentFile string, runtime Runtime, dependent bool) *Interpreter {
	return &Interpreter{
		ID:          runtime.NewID(),
		currentFile: currentFile,
		scope:       ScopeGlobal,
		dependent:   dependent,
		runtime:     runtime,
		objects:     make(map[uint32]*entry),
		imports:     make(map[string]*Interpreter),
		unsub:       make([]pubsub.UnsubscribeFunc, 0),
	}
}

func NewWithParent(parent *Interpreter, scope Scope) *Interpreter {
	return &Interpreter{
		ID:          parent.ID,
		currentFile: parent.currentFile,
		scope:       scope,
		parent:      parent,
		runtime:     parent.runtime,
		objects:     make(map[uint32]*entry),
		imports:     make(map[string]*Interpreter),
	}
}

func (i *Interpreter) Run(nodes []*astnode.Node) (language.Object, error) {
	defer i.Detach()

	for _, node := range nodes {
		obj, err := i.handleNode(node)
		if err != nil {
			return nil, err
		}
		if obj != nil {
			return obj, nil
		}
	}

	return nil, nil
}

func (i *Interpreter) Detach() {
	if i.dependent {
		return
	}
	i.MustDetach()
}

func (i *Interpreter) MustDetach() {
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, unsub := range i.unsub {
		unsub()
	}

	if i.scope == ScopeGlobal {
		i.runtime.RemoveInterpreter(i.ID)
	}

	for _, child := range i.imports {
		child.MustDetach()
	}
}
