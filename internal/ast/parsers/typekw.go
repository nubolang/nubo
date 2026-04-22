package parsers

import (
	"context"
	"fmt"
	"slices"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func TypeKWParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	node := &astnode.Node{
		Type:  astnode.NodeTypeTypeKW,
		Debug: token.Debug,
	}

	token = tokens[*inx]

	if token.Type != lexer.TokenIdentifier {
		return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected identifier, got %s", token.Value), token.Debug)
	}

	node.Content = token.Value

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenColon {
		return nil, newErr(ErrSyntaxError, "type keyword expects ':'")
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	typ, err := TypeParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.ValueType = typ

	token = tokens[*inx]
	if slices.Contains(white, token.Type) {
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}
		token = tokens[*inx]
	}

	return skipSemi(tokens, inx, node), nil
}
