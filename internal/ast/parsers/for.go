package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func ForParser(ctx context.Context, p Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type:  astnode.NodeTypeFor,
		Debug: tokens[*inx].Debug,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	forValue := &astnode.ForValue{}

	token := tokens[*inx]
	if token.Type != lexer.TokenIdentifier {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
	}

	forValue.Value = &astnode.Node{
		Type:  astnode.NodeTypeValue,
		Kind:  "IDENTIFIER",
		Value: token.Value,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, newErr(ErrUnexpectedEOF, fmt.Sprintf("expected ',' or 'in', got EOF"), token.Debug)
	}

	token = tokens[*inx]
	if token.Type == lexer.TokenComma {
		if err := inxPP(tokens, inx); err != nil {
			return nil, newErr(ErrUnexpectedEOF, fmt.Sprintf("expected identifier, got EOF"), token.Debug)
		}
		token = tokens[*inx]

		if token.Type != lexer.TokenIdentifier {
			return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
		}

		forValue.Iterator = forValue.Value
		forValue.Value = &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "IDENTIFIER",
			Value: token.Value,
		}

		if err := inxPP(tokens, inx); err != nil {
			return nil, newErr(ErrUnexpectedEOF, fmt.Sprintf("expected 'in', got EOF"), token.Debug)
		}

		token = tokens[*inx]
	}

	if token.Type != lexer.TokenIn {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected 'in', got %s", token.Type), token.Debug)
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
	token = tokens[*inx]

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

	node.Value = forValue
	node.Body = bodyNodes

	return node, nil
}
