package interpreter

import (
	"path/filepath"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
)

func (ir *Interpreter) handleInclude(node *astnode.Node) error {
	_, err := ir.includeValue(node)
	return err
}

func (ir *Interpreter) includeValue(node *astnode.Node) (language.Object, error) {
	value, err := ir.eval(node.Value.(*astnode.Node))
	if err != nil {
		return nil, err
	}

	fileName := value.String()

	var path string
	if filepath.IsAbs(fileName) {
		path = filepath.Clean(fileName)
	} else {
		dir := filepath.Dir(ir.currentFile)
		path = filepath.Join(dir, fileName)
		path = filepath.Clean(path)
	}

	if filepath.Ext(path) == "" {
		path += ".nubo"
	}

	nodes, err := native.NodesFromFile(path, path)
	if err != nil {
		return nil, newErr(err, ErrImportError.Error(), node.Debug)
	}

	inc := newInclude(ir, path, ir.runtime, ir.dependent, ir.workdir)

	ir.mu.Lock()
	ir.includes = append(ir.includes, inc)
	ir.mu.Unlock()

	return inc.Run(nodes)
}

func newInclude(parent *Interpreter, file string, runtime Runtime, dependent bool, wd string) *Interpreter {
	ir := &Interpreter{
		name:        "include",
		parent:      parent,
		ID:          runtime.NewID(),
		currentFile: filepath.Clean(file),
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

	ir.Declare("__id__", language.NewInt(int64(ir.ID), nil), language.TypeInt, false)
	ir.Declare("__entry__", language.NewBool(ir.ID == 1, nil), language.TypeBool, false)
	ir.Declare("__dir__", language.NewString(filepath.Join(wd, filepath.Dir(ir.currentFile)), nil), language.TypeString, false)
	ir.Declare("__file__", language.NewString(filepath.Join(wd, ir.currentFile), nil), language.TypeString, false)

	return ir
}
