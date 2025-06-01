package parsers

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

type HTMLAttrValueParser interface {
	ParseHTMLAttrValue(s string) (*astnode.Node, error)
}

func HTMLParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	var tag string
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

	attributes, selfClosing, err := createAttributes(ctx, sn, tokens, inx)
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

				tok = tokens[*inx]
				if tok.Type != lexer.TokenGreaterThan {
					return nil, newErr(ErrUnexpectedToken, "unexpected token", token.Debug)
				}

				*inx++
				return node, nil
			}

			if tok.Type == lexer.TokenLessThan {
				child, err := HTMLParser(ctx, sn, tokens, inx)
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
							if content.Len() > 0 && strings.TrimSpace(content.String()) != "" {
								text := &astnode.Node{
									Type:    astnode.NodeTypeElementRawText,
									Content: content.String(),
								}
								node.Children = append(node.Children, text)
								content.Reset()
							}

							var dynamicText strings.Builder

							var braceCount = 0
							for *inx < len(tokens) {
								select {
								case <-ctx.Done():
									return nil, ctx.Err()
								default:
									tok := tokens[*inx]

									dynamicText.WriteString(tok.Value)
									if tok.Type == lexer.TokenOpenBrace {
										braceCount++
									} else if tok.Type == lexer.TokenCloseBrace {
										braceCount--
										if braceCount == 0 {
											dynamicStr := dynamicText.String()
											dynamicStr = dynamicStr[1 : len(dynamicStr)-1]

											*inx++
											if dynamicText.Len() > 0 && strings.TrimSpace(dynamicStr) != "" {
												dynamicTextNode, err := sn.ParseHTMLAttrValue(dynamicStr)
												if err != nil {
													return nil, err
												}

												text := &astnode.Node{
													Type:  astnode.NodeTypeElementDynamicText,
													Value: dynamicTextNode,
												}
												text.Flags.Append("NODEVALUE")
												node.Children = append(node.Children, text)
											}
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

func createAttributes(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) ([]*astnode.Node, bool, error) {
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

	attributes, err := attrsToNodeParser(ctx, sn, attrs)
	if err != nil {
		return nil, false, err
	}

	return attributes, selfClosing, nil
}

func attrsToNodeParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token) ([]*astnode.Node, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	var (
		inx   = 0
		nodes []*astnode.Node
	)

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if inx >= len(tokens) {
				break loop
			}

			node := &astnode.Node{
				Type: astnode.NodeTypeElementAttribute,
			}

			token := tokens[inx]
			for inx < len(tokens) && slices.Contains(white, token.Type) {
				inx++
				token = tokens[inx]
			}

			if token.Type == lexer.TokenColon {
				node.Kind = "DYNAMIC"
				inx++

				if inx >= len(tokens) {
					return nil, newErr(ErrUnexpectedToken, "expected attribute name after colon", token.Debug)
				}

				token = tokens[inx]
			} else {
				node.Kind = "TEXT"
			}

			if inx >= len(tokens) || token.Type != lexer.TokenIdentifier {
				return nil, newErr(ErrUnexpectedToken, "expected attribute name", token.Debug)
			}

			start := inx
			var parts []string
			expectIdent := true

			for inx < len(tokens) {
				tok := tokens[inx]

				if expectIdent && tok.Type != lexer.TokenIdentifier {
					break
				}
				if !expectIdent && tok.Type != lexer.TokenMinus {
					break
				}

				parts = append(parts, tok.Value)
				expectIdent = !expectIdent
				inx++
			}

			if len(parts) == 0 || expectIdent {
				return nil, newErr(ErrUnexpectedToken, "invalid attribute name", tokens[start].Debug)
			}

			node.Content = strings.Join(parts, "")

			if inx >= len(tokens) {
				nodes = append(nodes, node)
				continue loop
			}

			token = tokens[inx]
			if token.Type == lexer.TokenIdentifier || token.Type == lexer.TokenColon {
				nodes = append(nodes, node)
				continue loop
			}

			if token.Type != lexer.TokenAssign {
				return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '=', got '%s'", token.Value), token.Debug)
			}

			inx++
			if inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedToken, "expected attribute value", token.Debug)
			}

			token = tokens[inx]
			if token.Type != lexer.TokenString {
				return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected string, got '%s'", token.Value), token.Debug)
			}

			if node.Kind == "DYNAMIC" {
				val, err := sn.ParseHTMLAttrValue(token.Value)
				if err != nil {
					return nil, err
				}
				node.Value = val
			} else {
				node.Value = token.Value
			}

			nodes = append(nodes, node)

			inx++
		}
	}

	return nodes, nil
}
