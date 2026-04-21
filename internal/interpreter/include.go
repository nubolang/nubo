package interpreter

import (
	"fmt"
	"path/filepath"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"go.uber.org/zap"
)

func (ir *Interpreter) handleInclude(node *astnode.Node) error {
	zap.L().Debug("interpreter.include.start", zap.Uint("id", ir.ID))

	if _, err := ir.includeValue(node); err != nil {
		zap.L().Error("interpreter.include.error", zap.Uint("id", ir.ID), zap.Error(err))
		return wrapRunExc(err, node.Debug)
	}
	zap.L().Debug("interpreter.include.success", zap.Uint("id", ir.ID))
	return nil
}

func (ir *Interpreter) includeValue(node *astnode.Node) (language.Object, error) {
	value, err := ir.eval(node.Value.(*astnode.Node))
	if err != nil {
		zap.L().Error("interpreter.include.evalError", zap.Uint("id", ir.ID), zap.Error(err))
		return nil, wrapRunExc(err, node.Debug)
	}

	fileName := value.String()
	zap.L().Debug("interpreter.include.file", zap.Uint("id", ir.ID), zap.String("fileName", fileName))

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

	zap.L().Debug("interpreter.include.path", zap.Uint("id", ir.ID), zap.String("path", path))

	nodes, err := native.NodesFromFile(path, path)
	if err != nil {
		zap.L().Error("interpreter.include.parseError", zap.Uint("id", ir.ID), zap.String("path", path), zap.Error(err))
		return nil, wrapRunExc(err, node.Debug, "failed to tokenize file")
	}

	inc := newInclude(ir, resolveIncludePath(ir.currentFile, fileName, ir.workdir))

	var interp = ir
	for interp.parent != nil {
		interp = interp.parent
	}

	interp.mu.Lock()
	interp.includes = append(interp.includes, inc)
	interp.mu.Unlock()

	ob, err := inc.Run(nodes)
	if err != nil {
		zap.L().Error("interpreter.include.runError", zap.Uint("id", ir.ID), zap.Error(err))
		return nil, wrapRunExc(err, node.Debug, "failed to execute included file")
	}
	zap.L().Debug("interpreter.include.return", zap.Uint("id", ir.ID), zap.String("returnType", logObjectType(ob)))
	return ob, nil
}

func newInclude(parent *Interpreter, file string) *Interpreter {
	ir := NewWithCustomFileParent(parent, ScopeGlobal, file, fmt.Sprintf("<include '%s'>", file))
	zap.L().Debug("interpreter.include.new", zap.Uint("id", ir.ID), zap.String("file", ir.currentFile))
	return ir
}

func resolveIncludePath(currentFile, includePath, workdir string) string {
	if filepath.IsAbs(includePath) {
		return filepath.Clean(includePath)
	}
	baseDir := filepath.Dir(currentFile)
	return filepath.Clean(filepath.Join(workdir, baseDir, includePath))
}
