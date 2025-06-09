package parsers

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

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
				if node.Kind == "DYNAMIC" && !strings.ContainsAny(node.Content, "-") {
					node.Value = &astnode.Node{
						Type: astnode.NodeTypeExpression,
						Body: []*astnode.Node{
							{
								Type:        astnode.NodeTypeValue,
								Kind:        "IDENTIFIER",
								Value:       node.Content,
								IsReference: true,
							},
						},
					}
				}
				nodes = append(nodes, node)
				continue loop
			}

			token = tokens[inx]
			if token.Type == lexer.TokenIdentifier || token.Type == lexer.TokenColon {
				if node.Kind == "DYNAMIC" && !strings.ContainsAny(node.Content, "-") {
					node.Value = &astnode.Node{
						Type: astnode.NodeTypeExpression,
						Body: []*astnode.Node{
							{
								Type:        astnode.NodeTypeValue,
								Kind:        "IDENTIFIER",
								Value:       node.Content,
								IsReference: true,
							},
						},
					}
				}
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
