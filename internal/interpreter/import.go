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
)

func (ir *Interpreter) handleImport(node *astnode.Node) error {
	_, ok := ir.GetObject(node.Content)
	if ok {
		return runExc("imported module name ('%s') should not be used as an identifier", node.Content).WithDebug(node.Debug)
	}

	ir.mu.RLock()
	_, ok = ir.imports[node.Content]
	if ok {
		ir.mu.RUnlock()
		return runExc("already imported module ('%s')", node.Content).WithDebug(node.Debug)
	}
	ir.mu.RUnlock()

	fileName := node.Value.(string)

	if strings.HasPrefix(fileName, "@std") || strings.HasPrefix(fileName, "@server") {
		if err := ir.stdImport(node, fileName); err != nil {
			return exception.From(err, node.Debug, "failed to import standard library module")
		}
		return nil
	}

	if strings.HasPrefix(fileName, "@") {
		fileName = strings.TrimPrefix(fileName, "@")
		p, err := ir.runtime.GetPacker()
		if err != nil {
			return exception.From(err, node.Debug, "failed to load packer registry")
		}
		name, err := p.ImportFile(fileName)
		if err != nil {
			return exception.From(err, node.Debug, fmt.Sprintf("failed to import file from packer registry: %s", fileName))
		}
		fileName = name
	}

	var path string
	if filepath.IsAbs(fileName) {
		path = filepath.Clean(fileName)
	} else {
		var oldFileName string
		for oldPrefix, newPrefix := range config.Current.Runtime.Interpreter.Import.Prefix {
			if strings.HasPrefix(fileName, oldPrefix) {
				fileName = filepath.Join(newPrefix, strings.TrimPrefix(fileName, oldPrefix))
				break // allow only one match (not replacing replaced paths)
			}
		}

		if oldFileName == fileName {
			dir := filepath.Dir(ir.currentFile)
			path = filepath.Join(dir, fileName)
			path = filepath.Clean(path)
		}
	}

	if filepath.Ext(path) == "" {
		path += ".nubo"
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return runExc("imported file %s does not exists", path).WithDebug(node.Debug)
	}

	imported, ok := ir.runtime.FindInterpreter(path)
	if !ok {
		nodes, err := native.NodesFromFile(path, path)
		if err != nil {
			return exception.From(err, node.Debug, "failed to parse imported file: @err")
		}

		imported = New(path, ir.runtime, true, ir.workdir)
		if _, err := imported.Run(nodes); err != nil {
			return exception.From(err, node.Debug, "failed to execute imported file")
		}
		ir.runtime.AddInterpreter(path, imported)
	}

	if node.Kind == "NONE" {
		return nil
	}

	if node.Kind == "SINGLE" {
		ir.mu.Lock()
		defer ir.mu.Unlock()
		ir.imports[node.Content] = imported
	}

	if node.Kind == "MULTIPLE" {
		for _, child := range node.Children {
			name := child.Value.(string)
			if _, ok := ir.GetObject(name); ok {
				return runExc("variable ('%s') already declared", name).WithDebug(node.Debug)
			}

			value, ok := imported.GetObject(child.Content)
			if !ok {
				return exception.Create("failed to import ('%s') from %s", name, path).WithDebug(node.Debug)
			}
			if err := ir.Declare(name, value, value.Type(), false); err != nil {
				return exception.From(err, node.Debug, "failed to declare variable")
			}
		}
	}

	return nil
}

func (ir *Interpreter) stdImport(node *astnode.Node, fileName string) error {
	obj, ok := ir.runtime.ImportPackage(fileName, node.Debug)
	if ok {
		if node.Kind == "NONE" {
			return nil
		}

		if node.Kind == "SINGLE" {
			if err := ir.Declare(node.Content, obj, obj.Type(), false); err != nil {
				return wrapRunExc(err, node.Debug)
			}
			return nil
		}

		if node.Kind == "MULTIPLE" {
			for _, child := range node.Children {
				name := child.Value.(string)
				if _, ok := ir.GetObject(name); ok {
					return runExc("cannot redeclare variable %q", name).WithDebug(node.Debug)
				}

				if obj.GetPrototype() == nil {
					return runExc("failed to import object from package: %s", child.Value.(string)).WithDebug(node.Debug)
				}

				value, ok := obj.GetPrototype().GetObject(child.Content)
				if !ok {
					return runExc("failed to import object from package: %s", child.Value.(string)).WithDebug(node.Debug)
				}

				if err := ir.Declare(name, value, value.Type(), false); err != nil {
					return wrapRunExc(err, node.Debug)
				}

				return nil
			}
		}
	}
	return runExc("failed to import %s from @std", strings.TrimPrefix(fileName, "@std/")).WithDebug(node.Debug)
}
