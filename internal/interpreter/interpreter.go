package interpreter

import (
	"sync"

	"github.com/google/uuid"
	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/pubsub"
	"github.com/nubogo/nubo/language"
)

type Runtime interface {
	GetBuiltin(name string) (language.Object, bool)
	GetEventProvider() pubsub.Provider
}

type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeFunction
	ScopeBlock
)

type Interpreter struct {
	ID          string
	currentFile string

	scope  Scope
	parent *Interpreter

	runtime Runtime
	objects map[uint32]*entry
	imports map[string]*Interpreter
	unsub   []pubsub.UnsubscribeFunc

	mu sync.RWMutex
}

func New(currentFile string, runtime Runtime) *Interpreter {
	return &Interpreter{
		ID:          uuid.NewString(),
		currentFile: currentFile,
		scope:       ScopeGlobal,
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
	defer func() {
		for _, unsub := range i.unsub {
			unsub()
		}
	}()

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
