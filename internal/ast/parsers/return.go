package parsers

import (
	"context"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func ReturnParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: astnode.NodeTypeReturn,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	value, err := ValueParser(ctx, sn, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.Value = value
	node.Flags.Append("NODEVALUE")

	return node, nil
}
