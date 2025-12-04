package interpreter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/config"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/native"
	"go.uber.org/zap"
)

func (ir *Interpreter) handleImport(node *astnode.Node) error {
	zap.L().Debug("interpreter.import.start", zap.Uint("id", ir.ID), zap.String("name", node.Content))

	_, ok := ir.GetObject(node.Content)
	if ok {
		err := runExc("imported module name ('%s') should not be used as an identifier", node.Content).WithDebug(node.Debug)
		zap.L().Error("interpreter.import.nameConflict", zap.Uint("id", ir.ID), zap.String("name", node.Content), zap.Error(err))
		return err
	}

	ir.mu.RLock()
	_, ok = ir.imports[node.Content]
	if ok {
		ir.mu.RUnlock()
		err := runExc("already imported module ('%s')", node.Content).WithDebug(node.Debug)
		zap.L().Error("interpreter.import.duplicate", zap.Uint("id", ir.ID), zap.String("name", node.Content), zap.Error(err))
		return err
	}
	ir.mu.RUnlock()

	fileName := node.Value.(string)
	zap.L().Debug("interpreter.import.source", zap.Uint("id", ir.ID), zap.String("name", node.Content), zap.String("source", fileName))

	if strings.HasPrefix(fileName, "@std") || strings.HasPrefix(fileName, "@server") {
		if err := ir.stdImport(node, fileName); err != nil {
			zap.L().Error("interpreter.import.stdError", zap.Uint("id", ir.ID), zap.String("name", node.Content), zap.Error(err))
			return exception.From(err, node.Debug, "failed to import standard library module")
		}
		zap.L().Debug("interpreter.import.stdSuccess", zap.Uint("id", ir.ID), zap.String("name", node.Content))
		return nil
	}

	if strings.HasPrefix(fileName, "@") {
		zap.L().Debug("interpreter.import.packer.start", zap.Uint("id", ir.ID), zap.String("name", node.Content), zap.String("source", fileName))
		fileName = strings.TrimPrefix(fileName, "@")
		p, err := ir.runtime.GetPacker()
		if err != nil {
			zap.L().Error("interpreter.import.packer.registry", zap.Uint("id", ir.ID), zap.Error(err))
			return exception.From(err, node.Debug, "failed to load packer registry")
		}
		name, err := p.ImportFile(fileName)
		if err != nil {
			zap.L().Error("interpreter.import.packer.file", zap.Uint("id", ir.ID), zap.String("source", fileName), zap.Error(err))
			return exception.From(err, node.Debug, fmt.Sprintf("failed to import file from packer registry: %s", fileName))
		}
		fileName = name
	}

	var path string
	if filepath.IsAbs(fileName) {
		path = filepath.Clean(fileName)
	} else {
		path = fileName
		for oldPrefix, newPrefix := range config.Current.Runtime.Interpreter.Import.Prefix {
			if strings.HasPrefix(fileName, oldPrefix) {
				path = filepath.Join(newPrefix, strings.TrimPrefix(fileName, oldPrefix))
				break // allow only one match (not replacing replaced paths)
			}
		}

		if path == fileName {
			dir := filepath.Dir(ir.currentFile)
			path = filepath.Join(dir, fileName)
			path = filepath.Clean(path)
		}
	}

	if filepath.Ext(path) == "" {
		path += ".nubo"
	}

	zap.L().Debug("interpreter.import.path", zap.Uint("id", ir.ID), zap.String("name", node.Content), zap.String("path", path))

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		perr := runExc("imported file %s does not exists", path).WithDebug(node.Debug)
		zap.L().Error("interpreter.import.missingFile", zap.Uint("id", ir.ID), zap.String("path", path), zap.Error(perr))
		return perr
	}

	imported, ok := ir.runtime.FindInterpreter(path)
	if !ok {
		zap.L().Debug("interpreter.import.load", zap.Uint("id", ir.ID), zap.String("path", path))
		nodes, err := native.NodesFromFile(path, path)
		if err != nil {
			zap.L().Error("interpreter.import.parseError", zap.Uint("id", ir.ID), zap.String("path", path), zap.Error(err))
			return exception.From(err, node.Debug, "failed to parse imported file: @err")
		}

		imported = New(ir.ctx, path, ir.runtime, true, ir.workdir)
		if _, err := imported.Run(nodes); err != nil {
			zap.L().Error("interpreter.import.execError", zap.Uint("id", ir.ID), zap.String("path", path), zap.Error(err))
			return exception.From(err, node.Debug, "failed to execute imported file")
		}
		ir.runtime.AddInterpreter(path, imported)
	}

	if node.Kind == "NONE" {
		zap.L().Debug("interpreter.import.noop", zap.Uint("id", ir.ID), zap.String("name", node.Content))
		return nil
	}

	if node.Kind == "SINGLE" {
		ir.mu.Lock()
		defer ir.mu.Unlock()
		ir.imports[node.Content] = imported
		zap.L().Debug("interpreter.import.single", zap.Uint("id", ir.ID), zap.String("name", node.Content))
	}

	if node.Kind == "MULTIPLE" {
		for _, child := range node.Children {
			name := child.Value.(string)
			if _, ok := ir.GetObject(name); ok {
				err := runExc("variable ('%s') already declared", name).WithDebug(node.Debug)
				zap.L().Error("interpreter.import.multiple.redeclare", zap.Uint("id", ir.ID), zap.String("alias", name), zap.Error(err))
				return err
			}

			value, ok := imported.GetObject(child.Content)
			if !ok {
				err := exception.Create("failed to import ('%s') from %s", name, path).WithDebug(node.Debug)
				zap.L().Error("interpreter.import.multiple.missing", zap.Uint("id", ir.ID), zap.String("alias", name), zap.String("source", child.Content), zap.Error(err))
				return err
			}
			if err := ir.Declare(name, value, value.Type(), false); err != nil {
				zap.L().Error("interpreter.import.multiple.declare", zap.Uint("id", ir.ID), zap.String("alias", name), zap.Error(err))
				return exception.From(err, node.Debug, "failed to declare variable")
			}
			zap.L().Debug("interpreter.import.multiple.entry", zap.Uint("id", ir.ID), zap.String("alias", name), zap.String("source", child.Content))
		}
	}

	zap.L().Debug("interpreter.import.success", zap.Uint("id", ir.ID), zap.String("name", node.Content))
	return nil
}

func (ir *Interpreter) stdImport(node *astnode.Node, fileName string) error {
	zap.L().Debug("interpreter.import.std.start", zap.Uint("id", ir.ID), zap.String("name", node.Content), zap.String("package", fileName))

	obj, ok := ir.runtime.ImportPackage(fileName, node.Debug)
	if ok {
		if node.Kind == "NONE" {
			zap.L().Debug("interpreter.import.std.noop", zap.Uint("id", ir.ID), zap.String("name", node.Content))
			return nil
		}

		if node.Kind == "SINGLE" {
			if err := ir.Declare(node.Content, obj, obj.Type(), false); err != nil {
				zap.L().Error("interpreter.import.std.declare", zap.Uint("id", ir.ID), zap.String("name", node.Content), zap.Error(err))
				return wrapRunExc(err, node.Debug)
			}
			zap.L().Debug("interpreter.import.std.single", zap.Uint("id", ir.ID), zap.String("name", node.Content))
			return nil
		}

		if node.Kind == "MULTIPLE" {
			for _, child := range node.Children {
				name := child.Value.(string)
				if _, ok := ir.GetObject(name); ok {
					err := runExc("cannot redeclare variable %q", name).WithDebug(node.Debug)
					zap.L().Error("interpreter.import.std.redeclare", zap.Uint("id", ir.ID), zap.String("alias", name), zap.Error(err))
					return err
				}

				if obj.GetPrototype() == nil {
					err := runExc("failed to import object from package: %s", child.Value.(string)).WithDebug(node.Debug)
					zap.L().Error("interpreter.import.std.noPrototype", zap.Uint("id", ir.ID), zap.String("alias", name), zap.Error(err))
					return err
				}

				value, ok := obj.GetPrototype().GetObject(ir.ctx, child.Content)
				if !ok {
					err := runExc("failed to import object from package: %s", child.Value.(string)).WithDebug(node.Debug)
					zap.L().Error("interpreter.import.std.missing", zap.Uint("id", ir.ID), zap.String("alias", name), zap.Error(err))
					return err
				}

				if err := ir.Declare(name, value, value.Type(), false); err != nil {
					zap.L().Error("interpreter.import.std.multipleDeclare", zap.Uint("id", ir.ID), zap.String("alias", name), zap.Error(err))
					return wrapRunExc(err, node.Debug)
				}

				zap.L().Debug("interpreter.import.std.multipleEntry", zap.Uint("id", ir.ID), zap.String("alias", name), zap.String("source", child.Content))

				return nil
			}
		}
	}
	err := runExc("failed to import %s from @std", strings.TrimPrefix(fileName, "@std/")).WithDebug(node.Debug)
	zap.L().Error("interpreter.import.std.notFound", zap.Uint("id", ir.ID), zap.String("package", fileName), zap.Error(err))
	return err
}
