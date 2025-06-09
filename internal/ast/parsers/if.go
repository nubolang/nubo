package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func IfParser(ctx context.Context, p interface {
	HTMLAttrValueParser
	parser
}, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type:  astnode.NodeTypeIf,
		Debug: tokens[*inx].Debug,
	}

	var (
		conditionTokens []*lexer.Token
		braceCount      = 0
	)

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
			if token.Type == lexer.TokenOpenBrace {
				if braceCount == 0 {
					break loop
				}
				braceCount++
			}
			if token.Type == lexer.TokenCloseBrace {
				braceCount--
			}

			conditionTokens = append(conditionTokens, token)
		}
	}

	cinx := 0
	condition, err := ValueParser(ctx, p, conditionTokens, &cinx)
	if err != nil {
		return nil, err
	}

	node.Args = append(node.Args, condition)
	token := tokens[*inx]

	if token.Type != lexer.TokenOpenBrace {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '{', got %s", token.Type), token.Debug)
	}

	var (
		body []*lexer.Token
	)

	braceCount = 1

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

	last := *inx
	if err := inxPP(tokens, inx); err == nil {
		token = tokens[*inx]
		if token.Type == lexer.TokenElse {
			els, err := elseParser(ctx, p, tokens, inx)
			if err != nil {
				return nil, err
			}
			node.Children = els
		}
	} else {
		*inx = last
	}

	return node, nil
}

func elseParser(ctx context.Context, p interface {
	parser
	HTMLAttrValueParser
}, tokens []*lexer.Token, inx *int) ([]*astnode.Node, error) {
	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	if token.Type == lexer.TokenIf {
		ifNode, err := IfParser(ctx, p, tokens, inx)
		if err != nil {
			return nil, err
		}
		return []*astnode.Node{ifNode}, nil
	}

	if token.Type != lexer.TokenOpenBrace {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '{', got %s", token.Type), token.Debug)
	}

	var (
		body []*lexer.Token
	)

	braceCount := 1

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
