package parsers

import (
	"context"
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func fnCallParser(ctx context.Context, attrParser HTMLAttrValueParser, id string, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]
	if token.Type != lexer.TokenOpenParen {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '(' but got %s", token.Type), token.Debug)
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	fn := &astnode.Node{
		Type:    astnode.NodeTypeFunctionCall,
		Content: id,
	}

	var (
		parenCount    = 1
		currentTokens []*lexer.Token
	)

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if *inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedEOF, "unexpected end of file", token.Debug)
			}

			var token = tokens[*inx]

			if token.Type == lexer.TokenCloseParen {
				parenCount--
				if parenCount == 0 {
					if len(currentTokens) != 0 {
						tinx := 0

						node, err := ValueParser(ctx, attrParser, currentTokens, &tinx)
						if err != nil {
							return nil, err
						}
						fn.Args = append(fn.Args, node)
						currentTokens = nil
					}
					*inx++
					break loop
				}
			}

			if token.Type == lexer.TokenOpenParen {
				parenCount++
			}

			if token.Type == lexer.TokenComma && parenCount == 1 {
				tinx := 0
				node, err := ValueParser(ctx, attrParser, currentTokens, &tinx)
				if err != nil {
					return nil, err
				}
				fn.Args = append(fn.Args, node)
				currentTokens = nil

				if err := inxPP(tokens, inx); err != nil {
					return nil, err
				}
				continue loop
			}

			currentTokens = append(currentTokens, token)
			if err := inxPP(tokens, inx); err != nil {
				break loop
			}
		}
	}

	if parenCount != 0 {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ')' but got %s", token.Type), token.Debug)
	}

	return fn, nil
}
