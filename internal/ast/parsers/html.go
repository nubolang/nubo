package parsers

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/lexer"
)

type HTMLAttrValueParser interface {
	ParseHTMLAttrValue(s string) (*astnode.Node, error)
	ParseHTML(s string, dg *debug.Debug) ([]*lexer.Token, error)
}

func HTMLBlockParser(ctx context.Context, sn HTMLAttrValueParser, token *lexer.Token) (*astnode.Node, error) {
	tokens, err := sn.ParseHTML(token.Value, token.Debug)
	if err != nil {
		return nil, err
	}

	inx := 0
	return HTMLParser(ctx, sn, tokens, &inx)
}

func HTMLParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]
	var tag string

	// Parse opening tag
	if token.Type == lexer.TokenLessThan && *inx+1 < len(tokens) && tokens[*inx+1].Type == lexer.TokenIdentifier {
		*inx++
		id, err := TypeWholeIDParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		tag = id
	}

	node := &astnode.Node{
		Type:    astnode.NodeTypeElement,
		Content: tag,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	attrs, selfClosing, err := createAttributes(ctx, sn, tokens, inx)
	if err != nil {
		return nil, err
	}
	node.Args = attrs

	if selfClosing {
		node.Flags.Append("SELFCLOSING")
		return node, nil
	}

	for *inx < len(tokens) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tok := tokens[*inx]

			// Handle closing tag
			if tok.Type == lexer.TokenClosingStartTag && *inx+2 < len(tokens) {
				*inx++
				id, err := TypeWholeIDParser(ctx, tokens, inx)
				if err != nil {
					return nil, err
				}
				if id != tag {
					return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("invalid closing tag, expected %s, got %s", tag, id), token.Debug)
				}
				if err := inxPP(tokens, inx); err != nil {
					return nil, err
				}
				if tokens[*inx].Type != lexer.TokenGreaterThan {
					return nil, newErr(ErrUnexpectedToken, "unexpected token", token.Debug)
				}
				*inx++
				return node, nil
			}

			// Handle child element
			if tok.Type == lexer.TokenLessThan {
				child, err := HTMLParser(ctx, sn, tokens, inx)
				if err != nil {
					return nil, err
				}
				node.Children = append(node.Children, child)
				continue
			}

			// Handle raw/dynamic text
			children, err := parseTextNodes(ctx, sn, tokens, inx)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, children...)
		}
	}

	return node, nil
}

func parseTextNodes(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) ([]*astnode.Node, error) {
	var children []*astnode.Node
	var content strings.Builder

textloop:
	for *inx < len(tokens) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tok := tokens[*inx]

			switch tok.Type {
			case lexer.TokenOpenBrace, lexer.TokenUnescapedBrace:
				isUnescaped := tok.Type == lexer.TokenUnescapedBrace
				if content.Len() > 0 && strings.TrimSpace(content.String()) != "" {
					children = append(children, &astnode.Node{
						Type:    astnode.NodeTypeElementRawText,
						Content: strings.TrimLeftFunc(content.String(), unicode.IsSpace),
					})
					content.Reset()
				}

				dynamicNode, err := parseDynamicText(ctx, sn, tokens, inx, isUnescaped)
				if err != nil {
					return nil, err
				}
				children = append(children, dynamicNode)
				continue textloop

			case lexer.TokenLessThan, lexer.TokenClosingStartTag:
				break textloop

			default:
				content.WriteString(tok.Value)
				*inx++
			}
		}
	}

	if content.Len() > 0 && strings.TrimSpace(content.String()) != "" {
		children = append(children, &astnode.Node{
			Type:    astnode.NodeTypeElementRawText,
			Content: strings.TrimSpace(content.String()),
		})
	}
	return children, nil
}

func parseDynamicText(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int, isUnescaped bool) (*astnode.Node, error) {
	var (
		b     strings.Builder
		debug *debug.Debug
	)
	braceCount := 0

	for *inx < len(tokens) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tok := tokens[*inx]
			debug = tok.Debug
			b.WriteString(tok.Text())

			if tok.Type == lexer.TokenOpenBrace || tok.Type == lexer.TokenUnescapedBrace {
				braceCount++
			} else if tok.Type == lexer.TokenCloseBrace {
				braceCount--
				if braceCount == 0 {
					text := b.String()
					text = strings.TrimSuffix(text, "}")
					if isUnescaped {
						text = strings.TrimPrefix(text, "@{")
					} else {
						text = strings.TrimPrefix(text, "{")
					}
					*inx++
					if strings.TrimSpace(text) == "" {
						return nil, nil
					}
					value, err := sn.ParseHTMLAttrValue(text)
					if err != nil {
						return nil, err
					}
					node := &astnode.Node{
						Type:  astnode.NodeTypeElementDynamicText,
						Value: value,
					}
					node.Flags.Append("NODEVALUE")
					if isUnescaped {
						node.Flags.Append("UNESCAPED")
					}
					return node, nil
				}
			}
			*inx++
		}
	}
	return nil, newErr(ErrUnexpectedToken, "unclosed dynamic text", debug)
}
