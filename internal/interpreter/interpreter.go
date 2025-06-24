package interpreter

import (
	"sync"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/packer"
)

type Runtime interface {
	GetBuiltin(name string) (language.Object, bool)
	GetEventProvider() events.Provider
	NewID() uint
	RemoveInterpreter(id uint)
	ImportPackage(name string, dg *debug.Debug) (language.Object, bool)
	GetPacker() (*packer.Packer, error)
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
	name   string
	parent *Interpreter

	runtime Runtime
	unsub   []events.UnsubscribeFunc

	imports map[string]*Interpreter
	objects map[uint32]*entry

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
		unsub:       make([]events.UnsubscribeFunc, 0),
	}
}

func NewWithParent(parent *Interpreter, scope Scope, name ...string) *Interpreter {
	var n string
	if len(name) > 0 {
		n = name[0]
	}

	return &Interpreter{
		ID:          parent.ID,
		currentFile: parent.currentFile,
		scope:       scope,
		name:        n,
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

func (i *Interpreter) isChildOf(scope Scope, name string) bool {
	current := i
	for {
		if current.scope == scope && current.name == name {
			return current.scope == scope && current.name == name
		}

		if current.parent == nil {
			return false
		}
		current = current.parent
	}
}
