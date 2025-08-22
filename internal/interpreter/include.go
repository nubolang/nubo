package interpreter

import (
	"path/filepath"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/native"
)

func (ir *Interpreter) handleInclude(node *astnode.Node) error {
	value, err := ir.eval(node.Value.(*astnode.Node))
	if err != nil {
		return err
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
		return newErr(err, ErrImportError.Error(), node.Debug)
	}

	_, err = ir.Run(nodes)
	return err
}
