package parsers

import (
	"context"
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func PubParser(ctx context.Context, attrParser Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	debug := tokens[*inx].Debug

	id, err := TypeWholeIDParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	if strings.Count(id, ".") > 1 {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected 1 dot, got %d", strings.Count(id, ".")), tokens[*inx].Debug)
	}

	node, err := fnCallParser(ctx, attrParser, id, tokens, inx)
	if err != nil {
		return nil, err
	}
	node.Debug = debug

	node.Type = astnode.NodeTypePublish // make it a publish node

	return node, nil
}
