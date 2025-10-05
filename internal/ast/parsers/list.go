package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func ListParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{Type: astnode.NodeTypeList, Debug: tokens[*inx].Debug}

	if err := inxNlPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	node.Debug = token.Debug
	if token.Type == lexer.TokenCloseBracket {
		return node, nil
	}

	var (
		cleaned     = make([]*lexer.Token, 0, len(tokens))
		bc      int = 1
		brace   int = 0
		paren   int = 0
	)

	for {
		if bc == 0 {
			break
		}

		token := tokens[*inx]
		if token.Type == lexer.TokenOpenBracket {
			bc++
		}

		if token.Type == lexer.TokenCloseBracket {
			bc--
		}

		if token.Type == lexer.TokenOpenBrace {
			brace++
		} else if token.Type == lexer.TokenCloseBrace {
			brace--
		}

		if token.Type == lexer.TokenOpenParen {
			paren++
		} else if token.Type == lexer.TokenCloseParen {
			paren--
		}

		cleaned = append(cleaned, token)

		if brace == 0 && paren == 0 {
			if err := inxNlPP(tokens, inx); err != nil {
				return nil, err
			}
		} else {
			*inx++
		}
	}

	*inx--

	cinx := 0
loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			value, err := ValueParser(ctx, sn, cleaned, &cinx)
			if err != nil {
				return nil, err
			}

			node.Children = append(node.Children, value)
			token := cleaned[cinx]
			if token.Type == lexer.TokenComma {
				cinx++
				continue loop
			}

			if token.Type == lexer.TokenCloseBracket {
				break loop
			}

			return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ',' or ']', got %s", token.Type), token.Debug)
		}
	}

	return node, nil
}
