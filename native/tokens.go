package native

import (
	"context"
	"os"
	"time"

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

	lx, err := lexer.New(file, lxPath)
	if err != nil {
		return nil, err
	}
	tokens, err := lx.Parse()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	parser := ast.New(ctx, time.Second*5)

	nodes, err := parser.Parse(tokens)
	if err != nil {
		return nil, err
	}

	_ = dotfolder.SavePrepared(path, nodes)

	return nodes, nil
}
