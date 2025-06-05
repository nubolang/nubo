package dotfolder

import (
	"context"
	"encoding/gob"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func PrepareFiles(dir string) error {
	preparedPath := filepath.Join(dir, RootFolderName, PreparedFolderName)
	if err := os.MkdirAll(preparedPath, 0755); err != nil {
		return err
	}

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".nubo" {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		newPath := strings.TrimSuffix(relPath, ".nubo") + ".nuboc"
		dest := filepath.Join(preparedPath, newPath)

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		lx := lexer.New(path)
		tokens, err := lx.Parse(file)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		parser := ast.New(ctx)
		nodes, err := parser.Parse(tokens)
		if err != nil {
			return err
		}

		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer out.Close()

		return gob.NewEncoder(out).Encode(nodes)
	})
}

func HasPrepared(file string) ([]*astnode.Node, bool) {
	dir := filepath.Dir(file)
	name := strings.TrimSuffix(filepath.Base(file), ".nubo") + ".nuboc"
	preparedPath := filepath.Join(dir, RootFolderName, PreparedFolderName, name)

	f, err := os.Open(preparedPath)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var nodes []*astnode.Node
	if err := gob.NewDecoder(f).Decode(&nodes); err != nil {
		return nil, false
	}

	return nodes, true
}
