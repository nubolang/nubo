package interpreter

import (
	"path/filepath"

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

	return ir.Run(nodes)
}
