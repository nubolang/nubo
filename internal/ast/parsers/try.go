package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func TryParser(ctx context.Context, p Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type:  astnode.NodeTypeTry,
		Debug: tokens[*inx].Debug,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	if token.Type != lexer.TokenIdentifier {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("Expected identifier, got %s", token.Type), tokens[*inx].Debug)
	}

	node.Content = token.Value

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenOpenBrace {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '{', got %s", token.Type), token.Debug)
	}

	var (
		body       []*lexer.Token
		braceCount = 1
	)

bodyloop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			*inx++

			if *inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedToken, "unexpected end of input", token.Debug)
			}

			token = tokens[*inx]

			if token.Type == lexer.TokenCloseBrace {
				braceCount--
				if braceCount == 0 {
					*inx++
					break bodyloop
				}
			} else if token.Type == lexer.TokenOpenBrace || token.Type == lexer.TokenUnescapedBrace {
				braceCount++
			}

			body = append(body, token)
		}
	}

	if braceCount != 0 {
		return nil, newErr(ErrUnexpectedToken, "unbalanced braces", token.Debug)
	}

	bodyNodes, err := p.Parse(body)
	if err != nil {
		return nil, err
	}

	node.Body = bodyNodes

	return node, nil
}
