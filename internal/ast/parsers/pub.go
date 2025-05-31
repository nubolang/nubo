package parsers

import (
	"context"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func PubParser(ctx context.Context, attrParser HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	id, err := TypeWholeIDParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	node, err := fnCallParser(ctx, attrParser, id, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.Type = astnode.NodeTypePublish // make it a publish node

	return node, nil
}
