package parsers

import (
	"context"
	"strings"

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
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if *inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedToken, "unexpected end of tokens", token.Debug)
			}

			tok := tokens[*inx]

			tok = tokens[*inx]

			if tok.Type == lexer.TokenClosingStartTag && *inx+2 < len(tokens) && tokens[*inx+1].Value == tag && tokens[*inx+2].Type == lexer.TokenGreaterThan {
				*inx += 3
				return node, nil
			}

			if tok.Type == lexer.TokenLessThan {
				child, err := HTMLParser(ctx, tokens, inx)
				if err != nil {
					return nil, err
				}
				node.Children = append(node.Children, child)
			} else {
				var content strings.Builder
			textloop:
				for *inx < len(tokens) {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					default:
						tok := tokens[*inx]

						if tok.Type == lexer.TokenOpenBrace {
							var braceCount = 0
							for *inx < len(tokens) {
								select {
								case <-ctx.Done():
									return nil, ctx.Err()
								default:
									tok := tokens[*inx]

									content.WriteString(tok.Value)
									if tok.Type == lexer.TokenOpenBrace {
										braceCount++
									} else if tok.Type == lexer.TokenCloseBrace {
										braceCount--
										if braceCount == 0 {
											*inx++
											continue textloop
										}
									}

									*inx++
								}
							}
						}

						if tok.Type == lexer.TokenLessThan || tok.Type == lexer.TokenClosingStartTag {
							break textloop
						}
						content.WriteString(tok.Value)
						*inx++
					}
				}

				if content.Len() > 0 && strings.TrimSpace(content.String()) != "" {
					text := &astnode.Node{
						Type:    astnode.NodeTypeElementRawText,
						Content: content.String(),
					}
					node.Children = append(node.Children, text)
				}
			}
		}
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
