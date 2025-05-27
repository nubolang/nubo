package parsers

import (
	"context"
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func StructParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: lexer.TokenStruct,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	if token.Type != lexer.TokenIdentifier {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
	}
	node.Content = token.Value

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenOpenBrace {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '{', got %s", token.Type), token.Debug)
	}

	if err := nl(tokens, inx); err != nil {
		return nil, err
	}

	var body []*astnode.Node

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}

			token = tokens[*inx]
			if token.Type == lexer.TokenCloseBrace {
				*inx++
				break loop
			}

			child := &astnode.Node{
				Type: lexer.TokenIdentifier,
			}

			token = tokens[*inx]
			if token.Type != lexer.TokenIdentifier {
				return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
			}
			child.Content = token.Value

			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}

			token = tokens[*inx]
			if token.Type != lexer.TokenColon {
				return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ':', got %s", token.Type), token.Debug)
			}

			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}

			typ, err := TypeParser(ctx, tokens, inx)
			if err != nil {
				return nil, err
			}
			child.ValueType = typ

			if *inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("unexpected end of input"), token.Debug)
			}

			body = append(body, child)

			token = tokens[*inx]
			if token.Type == lexer.TokenNewLine {
				continue
			}

			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}
		}
	}

	node.Body = body

	return node, nil
}
