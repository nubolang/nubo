package parsers

import (
	"context"
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func ListParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{Type: astnode.NodeTypeList}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
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
