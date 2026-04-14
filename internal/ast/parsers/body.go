package parsers

import (
	"context"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func getParsedBodyNodes(ctx context.Context, tokens []*lexer.Token, inx *int, sn Parser_HTML, p parser) ([]*astnode.Node, error) {
	token := tokens[*inx]
	switch token.Type {
	default:
		return nil, newErr(ErrUnexpectedToken, "unexpected token", token.Debug)
	case lexer.TokenOpenBrace:
		return getBraceBodyParse(ctx, tokens, inx, p)
	case lexer.TokenColon:
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}

		dummyInx := *inx
		_, err := ValueParser(ctx, sn, tokens, &dummyInx)
		if err != nil {
			*inx = dummyInx
			return nil, err
		}

		tokens = tokens[*inx:dummyInx]
		*inx = dummyInx

		return p.Parse(tokens)
	}
}

func getBraceBodyParse(ctx context.Context, tokens []*lexer.Token, inx *int, p parser) ([]*astnode.Node, error) {
	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]

	var (
		braceCount = 1
		body       []*lexer.Token
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

	return p.Parse(body)
}
