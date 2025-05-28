package parsers

import (
	"context"
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func HTMLParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	var tag string
	if token.Type == lexer.TokenLessThan && *inx+1 < len(tokens) && tokens[*inx+1].Type == lexer.TokenIdentifier {
		*inx++
		tag = tokens[*inx].Value
	}

	node := &astnode.Node{
		Type:    astnode.NodeTypeElement,
		Content: tag,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	attributes, selfClosing, err := createAttributes(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.Args = attributes
	if selfClosing {
		node.Flags.Append("SELFCLOSING")
		return node, nil
	}

	for *inx < len(tokens) {
		tok := tokens[*inx]

		if tok.Type == lexer.TokenNewLine {
			if err := nl(tokens, inx); err != nil {
				return nil, err
			}
		}

		tok = tokens[*inx]

		if tok.Type == lexer.TokenClosingStartTag {
			if *inx+1 < len(tokens) && tokens[*inx+1].Value == tag {
				if *inx+2 < len(tokens) && tokens[*inx+2].Type == lexer.TokenGreaterThan {
					*inx += 3
					break
				}
			}
		}

		i := *inx
		child, err := HTMLParser(ctx, tokens, inx)
		if err != nil {
			for _, t := range tokens[i:] {
				fmt.Printf("%s ", t.Value)
			}
			return nil, err
		}
		node.Children = append(node.Children, child)
	}

	return node, nil
}

func createAttributes(ctx context.Context, tokens []*lexer.Token, inx *int) ([]*astnode.Node, bool, error) {
	var (
		token       = tokens[*inx]
		attrs       []*lexer.Token
		selfClosing = false
	)

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		default:
			if token.Type == lexer.TokenGreaterThan || token.Type == lexer.TokenSelfClosingTag {
				if token.Type == lexer.TokenSelfClosingTag {
					selfClosing = true
				}
				*inx++
				break loop
			}

			attrs = append(attrs, token)

			if err := inxPP(tokens, inx); err != nil {
				return nil, false, err
			}

			token = tokens[*inx]
		}
	}

	return nil, selfClosing, nil
}
