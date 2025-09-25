package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func ListParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{Type: astnode.NodeTypeList, Debug: tokens[*inx].Debug}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	node.Debug = token.Debug
	if token.Type == lexer.TokenCloseBracket {
		return node, nil
	}

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			value, err := ValueParser(ctx, sn, tokens, inx)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, value)

			if *inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedEOF, "unexpected end of file", node.Debug)
			}

			token := tokens[*inx]
			if token.Type == lexer.TokenComma {
				if err := inxPP(tokens, inx); err != nil {
					return nil, err
				}
				continue loop
			}

			if token.Type == lexer.TokenCloseBracket {
				_ = inxPP(tokens, inx)
				break loop
			}

			return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ',' or ']', got %s", token.Type), token.Debug)
		}
	}

	return node, nil
}
