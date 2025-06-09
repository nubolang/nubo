package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func DictParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	node := &astnode.Node{
		Type:  astnode.NodeTypeDict,
		Debug: token.Debug,
	}

	node.Flags.Append("NODEVALUE")

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenOpenBrace {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '{', got '%s'", token.Value), node.Debug)
	}

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}

			token := tokens[*inx]
			if token.Type == lexer.TokenNewLine {
				continue loop
			}

			if token.Type == lexer.TokenCloseBrace {
				break loop
			}

			if token.Type != lexer.TokenString && token.Type != lexer.TokenIdentifier && token.Type != lexer.TokenNumber {
				return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("dict key must be string, identifier or number, got '%s'", token.Value), node.Debug)
			}

			key, err := singleValueParser(ctx, sn, tokens, inx, token)
			if err != nil {
				return nil, err
			}

			if key.Kind == "IDENTIFIER" {
				key.Kind = "STRING"
				key.IsReference = false
			}

			dataset := &astnode.Node{
				Type: astnode.NodeTypeDictField,
				Value: &astnode.Node{
					Type:  astnode.NodeTypeExpression,
					Body:  []*astnode.Node{key},
					Debug: key.Debug,
				},
			}

			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}

			token = tokens[*inx]
			if token.Type != lexer.TokenColon {
				return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ':', got '%s'", token.Value), node.Debug)
			}

			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}

			value, err := ValueParser(ctx, sn, tokens, inx)
			if err != nil {
				return nil, err
			}

			dataset.Children = append(dataset.Children, value)

			node.Children = append(node.Children, dataset)

			if *inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("unexpected end of input"), node.Debug)
			}

			token = tokens[*inx]
			if token.Type == lexer.TokenComma {
				if err := inxPP(tokens, inx); err != nil {
					return nil, err
				}

				continue loop
			}

			if token.Type == lexer.TokenNewLine {
				if err := inxPP(tokens, inx); err != nil {
					return nil, err
				}

				if tokens[*inx].Type == lexer.TokenCloseBrace {
					break loop
				}
			}

			if token.Type == lexer.TokenCloseBrace {
				break loop
			}

			return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ',' or newline, got '%s'", token.Value), node.Debug)
		}
	}

	last := *inx
	if err := inxPP(tokens, inx); err != nil {
		*inx = last
	}

	return node, nil
}
