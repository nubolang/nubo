package parsers

import (
	"context"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func DeferParser(ctx context.Context, p Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: astnode.NodeTypeDefer,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	value, err := ValueParser(ctx, p, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.Children = append(node.Children, value)

	return node, nil
}
