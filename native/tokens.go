package native

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/nubolang/nubo/config"
	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/dotfolder"
	"github.com/nubolang/nubo/internal/lexer"
)

func NodesFromFile(path, lxPath string) ([]*astnode.Node, error) {
	if prepared, ok := dotfolder.HasPrepared(path); ok {
		return prepared, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	nodes, err := NodesFromReader(file, lxPath)
	if err != nil {
		return nil, err
	}

	_ = dotfolder.SavePrepared(path, nodes)

	return nodes, nil
}

func NodesFromReader(reader io.Reader, file string) ([]*astnode.Node, error) {
	config.Verify()

	lx, err := lexer.New(reader, file)
	if err != nil {
		return nil, err
	}
	tokens, err := lx.Parse()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	parser := ast.New(ctx, time.Millisecond*time.Duration(config.Current.Syntax.Tokenizer.Context.Deadline))

	nodes, err := parser.Parse(tokens)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}
