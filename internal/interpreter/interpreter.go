package interpreter

import (
	"path/filepath"
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
	workdir     string

	scope  Scope
	name   string
	parent *Interpreter

	runtime Runtime
	unsub   []events.UnsubscribeFunc

	imports map[string]*Interpreter
	objects map[uint32]*entry

	deferred [][]*astnode.Node

	mu sync.RWMutex
}

func New(currentFile string, runtime Runtime, dependent bool, wd string) *Interpreter {
	ir := &Interpreter{
		ID:          runtime.NewID(),
		currentFile: filepath.Clean(currentFile),
		scope:       ScopeGlobal,
		dependent:   dependent,
		workdir:     wd,
		runtime:     runtime,
		objects:     make(map[uint32]*entry),
		imports:     make(map[string]*Interpreter),
		unsub:       make([]events.UnsubscribeFunc, 0),
		deferred:    make([][]*astnode.Node, 0),
	}

	ir.Declare("__id__", language.NewInt(int64(ir.ID), nil), language.TypeInt, false)
	ir.Declare("__entry__", language.NewBool(ir.ID == 1, nil), language.TypeBool, false)
	ir.Declare("__dir__", language.NewString(filepath.Join(wd, filepath.Dir(ir.currentFile)), nil), language.TypeString, false)
	ir.Declare("__file__", language.NewString(filepath.Join(wd, ir.currentFile), nil), language.TypeString, false)

	return ir
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
	defer i.runDeferred()
	defer i.Detach()

	for _, node := range nodes {
		obj, err := i.handleNode(node)
		if err != nil {
			return nil, err
		}
		if obj != nil {
			if i.parent != nil && node.Type == astnode.NodeTypeFunctionCall && i.scope == ScopeFunction {
				continue
			}
			if i.parent == nil && obj.Type().Base() == language.ObjectTypeSignal {
				return nil, nil
			}
			return obj, nil
		}
	}

	return nil, nil
}

func (i *Interpreter) runDeferred() {
	for _, deferred := range i.deferred {
		for _, node := range deferred {
			_, _ = i.eval(node)
		}
	}
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
