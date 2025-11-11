package interpreter

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeFunction
	ScopeBlock
)

type Interpreter struct {
	ctx context.Context

	ID          uint
	currentFile string
	dependent   bool
	workdir     string

	scope  Scope
	name   string
	parent *Interpreter

	runtime Runtime
	unsub   []events.UnsubscribeFunc

	imports  map[string]*Interpreter
	includes []*Interpreter
	objects  map[uint32]*entry

	deferred [][]*astnode.Node

	mu sync.RWMutex
}

func New(ctx context.Context, currentFile string, runtime Runtime, dependent bool, wd string) *Interpreter {
	ir := &Interpreter{
		ctx:         ctx,
		ID:          runtime.NewID(),
		currentFile: filepath.Clean(currentFile),
		scope:       ScopeGlobal,
		dependent:   dependent,
		workdir:     wd,
		runtime:     runtime,
		objects:     make(map[uint32]*entry),
		imports:     make(map[string]*Interpreter),
		includes:    make([]*Interpreter, 0),
		unsub:       make([]events.UnsubscribeFunc, 0),
		deferred:    make([][]*astnode.Node, 0),
	}

	zap.L().Info("[interpreter] new interpreter", zap.Uint("id", ir.ID), zap.String("file", ir.currentFile))

	ir.Declare("__id__", language.NewInt(int64(ir.ID), nil), language.TypeInt, false)
	ir.Declare("__entry__", language.NewBool(ir.ID == 1, nil), language.TypeBool, false)
	ir.Declare("__dir__", language.NewString(filepath.Join(wd, filepath.Dir(ir.currentFile)), nil), language.TypeString, false)
	ir.Declare("__file__", language.NewString(filepath.Join(wd, ir.currentFile), nil), language.TypeString, false)
	ir.Declare("__concurrent__", language.NewBool(false, nil), language.TypeBool, false)

	return ir
}

func NewWithParent(parent *Interpreter, scope Scope, name ...string) *Interpreter {
	var n string
	if len(name) > 0 {
		n = name[0]
	}

	zap.L().Info("[interpreter] new parent interpreter", zap.Uint("for_id", parent.ID), zap.String("file", parent.currentFile), zap.String("name", n))

	return &Interpreter{
		ctx:         parent.ctx,
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
	defer func() {
		i.runDeferred()
		i.Detach()
	}()

	zap.L().Info("[interpreter] running nodes", zap.Uint("id", i.ID), zap.Int("count", len(nodes)))

	for _, node := range nodes {
		obj, err := i.handleNode(node)
		if err != nil {
			zap.L().Info("[interpreter] handleNode exception", zap.Uint("id", i.ID), zap.Error(err))
			return nil, exception.From(err, node.Debug, "failed to handle node: @err")
		}
		if obj != nil {
			if i.parent != nil && node.Type == astnode.NodeTypeFunctionCall && i.scope == ScopeFunction {
				zap.L().Info("[interpreter] handleNode continue", zap.Uint("id", i.ID))
				continue
			}

			if i.parent == nil && obj.Type().Base() == language.ObjectTypeSignal {
				zap.L().Info("[interpreter] empty return signal", zap.Uint("id", i.ID))
				return nil, nil
			}

			zap.L().Info("[interpreter] object return signal", zap.Uint("id", i.ID))
			return obj, nil
		}
	}

	return nil, nil
}

func (i *Interpreter) runDeferred() {
	zap.L().Info("[interpreter] running deferred statements", zap.Uint("id", i.ID), zap.Int("count", len(i.deferred)))

	for _, deferred := range i.deferred {
		for _, node := range deferred {
			_, _ = i.eval(node)
		}
	}
}

func (i *Interpreter) Detach() {
	zap.L().Info("[interpreter] detaching interpreter", zap.Uint("id", i.ID), zap.Bool("dependent", i.dependent))

	if i.dependent {
		return
	}
	i.MustDetach()
}

func (i *Interpreter) MustDetach() {
	zap.L().Info("[interpreter] must detaching interpreter", zap.Uint("id", i.ID))

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
