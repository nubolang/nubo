package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/config"
	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/dotfolder"
	"github.com/nubolang/nubo/internal/lexer"
	"gopkg.in/yaml.v3"
)

func parseFile(filePath string) ([]*astnode.Node, error) {
	dev := os.Getenv("NUBO_DEV") == "true"

	var syntaxTree []*astnode.Node
	if prepared, ok := dotfolder.HasPrepared(filePath); ok {
		if dev {
			fmt.Println(color.New(color.FgBlue).Sprint("Using a prepared file 🚀"))
		}
		syntaxTree = prepared
	} else {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		lx, err := lexer.New(file, filePath)
		if err != nil {
			return nil, err
		}
		tokens, err := lx.Parse()
		if err != nil {
			return nil, err
		}

		if dev {
			if err := writeDebug(filePath, tokens, true); err != nil {
				return nil, err
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(config.Current.Syntax.Tokenizer.Context.Deadline))
		defer cancel()

		builder := ast.New(ctx, time.Millisecond*time.Duration(config.Current.Syntax.Tokenizer.Context.Deadline))
		syntaxTree, err = builder.Parse(tokens)
		if err != nil {
			return nil, err
		}

		if dev {
			if err := writeDebug(filePath, syntaxTree, false); err != nil {
				return nil, err
			}
		}
	}

	return syntaxTree, nil
}

func writeDebug(filename string, data any, lx bool) error {
	var path string
	if lx {
		path = config.Current.Syntax.Lexer.Debug.File
	} else {
		path = config.Current.Syntax.Tokenizer.Debug.File
	}

	if path == "{nubo_dir_full_file}" {
		path = filepath.Join(config.Nubo, "debug")
		if lx {
			path = filepath.Join(path, filename+".lx.yaml")
		} else {
			path = filepath.Join(path, filename+".node.yaml")
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return yaml.NewEncoder(file).Encode(data)
}
