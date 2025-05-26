package parsers

import (
	"context"
	"fmt"

	"github.com/bndrmrtn/tea/internal/ast/astnode"
	"github.com/bndrmrtn/tea/internal/lexer"
)

func ImportParser(_ context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: lexer.TokenImport,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	if token.Type != lexer.TokenOpenBrace && token.Type != lexer.TokenIdentifier {
		return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected identifier or open brace, got %s", token.Type), token.Debug)
	}

	if token.Type == lexer.TokenIdentifier {
		node.Kind = "SINGLE"
		node.Content = token.Value
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenFrom {
		return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected keyword 'from', got %s", token.Type), token.Debug)
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenString {
		return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected string, got %s", token.Type), token.Debug)
	}

	node.Content = token.Value

	return skipSemi(tokens, inx, node), nil
}
