package interpreter

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sync"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/execution"
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

	zap.L().Debug("interpreter.new", zap.Uint("id", ir.ID), zap.String("file", ir.currentFile))

	file := ir.currentFile
	if !filepath.IsAbs(file) {
		file = filepath.Join(wd, file)
	}

	ir.Declare("__id__", language.NewInt(int64(ir.ID), nil), language.TypeInt, false)
	ir.Declare("__entry__", language.NewBool(ir.ID == 1 && !dependent, nil), language.TypeBool, false)
	ir.Declare("__dir__", language.NewString(filepath.Dir(file), nil), language.TypeString, false)
	ir.Declare("__file__", language.NewString(file, nil), language.TypeString, false)
	ir.Declare("__concurrent__", language.NewBool(false, nil), language.TypeBool, false)

	return ir
}

func NewWithParent(parent *Interpreter, scope Scope, name ...string) *Interpreter {
	var n string
	if len(name) > 0 {
		n = name[0]
	}

	zap.L().Debug("interpreter.new.parent", zap.Uint("for_id", parent.ID), zap.String("file", parent.currentFile), zap.String("name", n))

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

func NewWithCustomFileParent(parent *Interpreter, scope Scope, file string, name ...string) *Interpreter {
	var n string
	if len(name) > 0 {
		n = name[0]
	}

	zap.L().Debug("interpreter.new.parent", zap.Uint("for_id", parent.ID), zap.String("file", parent.currentFile), zap.String("name", n))

	ir := &Interpreter{
		ctx:         parent.ctx,
		ID:          parent.ID,
		currentFile: file,
		scope:       scope,
		name:        n,
		parent:      parent,
		runtime:     parent.runtime,
		objects:     make(map[uint32]*entry),
		imports:     make(map[string]*Interpreter),
	}

	ir.Declare("__id__", language.NewInt(int64(ir.ID), nil), language.TypeInt, false)
	ir.Declare("__entry__", language.NewBool(ir.ID == 1, nil), language.TypeBool, false)
	ir.Declare("__dir__", language.NewString(filepath.Dir(ir.currentFile), nil), language.TypeString, false)
	ir.Declare("__file__", language.NewString(ir.currentFile, nil), language.TypeString, false)

	return ir
}

func (i *Interpreter) Run(nodes []*astnode.Node) (obj language.Object, err error) {
	if err := execution.Check(i.ctx); err != nil {
		return nil, err
	}
	base := len(i.deferred)
	defer i.Detach()
	defer func() {
		err = errors.Join(err, i.runDeferred(base))
	}()

	zap.L().Debug("interpreter.run.start", zap.Uint("id", i.ID), zap.Int("count", len(nodes)))

	for _, node := range nodes {
		if err := execution.Check(i.ctx); err != nil {
			return nil, err
		}
		obj, err := i.handleNode(node)
		if err != nil {
			zap.L().Error("interpreter.run.handleNode", zap.Uint("id", i.ID), zap.Error(err))
			return nil, exception.From(err, node.Debug, "failed to handle node: @err")
		}

		if obj != nil {
			if i.parent != nil && node.Type == astnode.NodeTypeFunctionCall && i.scope == ScopeFunction {
				zap.L().Debug("interpreter.run.continue", zap.Uint("id", i.ID))
				continue
			}

			if i.parent == nil && obj.Type().Base() == language.ObjectTypeSignal {
				zap.L().Debug("interpreter.run.emptyReturn", zap.Uint("id", i.ID))
				return nil, nil
			}

			zap.L().Debug("interpreter.run.return", zap.Uint("id", i.ID))
			return obj, nil
		}
	}

	return nil, nil
}

func (i *Interpreter) runDeferred(base int) (err error) {
	if len(i.deferred) == base {
		return nil
	}
	cleanup := NewWithParent(i, ScopeBlock, "defer")
	cleanup.ctx = execution.Cleanup(i.ctx)
	pending := i.deferred[base:]
	i.deferred = i.deferred[:base]
	
	for _, deferred := range slices.Backward(pending) {
		for _, node := range slices.Backward(deferred) {
			func() {
				defer func() {
					if v := recover(); v != nil {
						err = errors.Join(err, fmt.Errorf("deferred function panicked: %v", v))
					}
				}()
				_, e := cleanup.eval(node)
				err = errors.Join(err, e)
			}()
		}
	}
	return err
}

func (i *Interpreter) Detach() {
	zap.L().Debug("interpreter.detach", zap.Uint("id", i.ID), zap.Bool("dependent", i.dependent))

	if i.dependent {
		return
	}
	i.MustDetach()
}

func (i *Interpreter) MustDetach() {
	i.mustDetach(make(map[*Interpreter]bool))
}

func (i *Interpreter) mustDetach(visited map[*Interpreter]bool) {
	if i == nil || visited[i] {
		return
	}

	visited[i] = true
	zap.L().Debug("interpreter.detach.must", zap.Uint("id", i.ID))

	// Release the lock before recursive cleanup.
	i.mu.RLock()
	unsubscribers := append([]events.UnsubscribeFunc(nil), i.unsub...)
	children := make([]*Interpreter, 0, len(i.imports))

	for _, child := range i.imports {
		children = append(children, child)
	}

	i.mu.RUnlock()

	for _, unsub := range unsubscribers {
		unsub()
	}

	if i.scope == ScopeGlobal {
		i.runtime.RemoveInterpreter(i.ID)
	}

	for _, child := range children {
		child.mustDetach(visited)
	}
}

func (i *Interpreter) isChildOf(scope Scope, name string) bool {
	current := i
	for {
		if current.scope == scope && current.name == name {
			return current.scope == scope && current.name == name
		}

		if current.scope == ScopeFunction || current.parent == nil {
			return false
		}
		current = current.parent
	}
}
