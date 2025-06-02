package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
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
			*inx++
			if *inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedEOF, "unexpected end of file", token.Debug)
			}
		}
	}

	if parenCount != 0 {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ')' but got %s", token.Type), token.Debug)
	}

	if *inx < len(tokens) && tokens[*inx].Type == lexer.TokenDot {
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}
		if tokens[*inx].Type == lexer.TokenIdentifier {
			node, err := fnChildParser(ctx, attrParser, tokens, inx)
			if err != nil {
				return nil, err
			}
			fn.Children = append(fn.Children, node)
			return fn, nil
		}
	}

	last := *inx
	if err := inxPP(tokens, inx); err == nil {
		if tokens[*inx].Type == lexer.TokenDot {
			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}
			if tokens[*inx].Type == lexer.TokenIdentifier {
				node, err := fnChildParser(ctx, attrParser, tokens, inx)
				if err != nil {
					return nil, err
				}
				fn.Children = append(fn.Children, node)
				return fn, nil
			}
		}
	}
	*inx = last

	return fn, nil
}

func fnChildParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	debug := tokens[*inx].Debug
	id, err := TypeWholeIDParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	last := *inx
	if err := inxPP(tokens, inx); err != nil {
		*inx = last
		return &astnode.Node{
			Type:    astnode.NodeTypeValue,
			Content: id,
			Kind:    "IDENTIFIER",
			Debug:   debug,
		}, nil
	}

	token := tokens[*inx]

	switch token.Type {
	case lexer.TokenOpenParen:
		return fnCallParser(ctx, sn, id, tokens, inx)
	}

	return &astnode.Node{
		Type:    astnode.NodeTypeValue,
		Content: id,
		Kind:    "IDENTIFIER",
		Debug:   debug,
	}, newErr(ErrUnexpectedToken, fmt.Sprintf("unexpected token %s", token.Value), token.Debug)
}
