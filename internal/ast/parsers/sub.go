package parsers

import (
	"context"
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func SubParser(ctx context.Context, p interface {
	HTMLAttrValueParser
	parser
}, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
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

	node, err := fnCallParser(ctx, p, id, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.Type = astnode.NodeTypeSubscribe // make it a subscribe node

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]

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
			} else if token.Type == lexer.TokenOpenBrace {
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
