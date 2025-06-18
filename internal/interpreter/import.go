package interpreter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/native"
)

func (ir *Interpreter) handleImport(node *astnode.Node) error {
	_, ok := ir.GetObject(node.Content)
	if ok {
		return newErr(ErrImportError, fmt.Sprintf("imported module name should not be used as an identifier"), node.Debug)
	}

	ir.mu.RLock()
	_, ok = ir.imports[node.Content]
	if ok {
		ir.mu.RUnlock()
		return newErr(ErrImportError, fmt.Sprintf("module %s already imported", node.Content), node.Debug)
	}
	ir.mu.RUnlock()

	fileName := node.Value.(string)

	if strings.HasPrefix(fileName, "pkg:") {
		fileName = strings.TrimPrefix(fileName, "pkg:")
		p, err := ir.runtime.GetPacker()
		if err != nil {
			return err
		}
		name, err := p.ImportFile(fileName)
		if err != nil {
			return err
		}
		fileName = name
	}

	obj, ok := ir.runtime.ImportPackage(fileName)
	if ok {
		return ir.Declare(node.Content, obj, obj.Type(), false)
	}

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

	nodes, err := native.NodesFromFile(path)
	if err != nil {
		return err
	}

	imported := New(path, ir.runtime, true)
	if _, err := imported.Run(nodes); err != nil {
		return err
	}

	ir.mu.Lock()
	defer ir.mu.Unlock()
	ir.imports[node.Content] = imported

	return nil
}
