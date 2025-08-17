package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func ImportParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type:  astnode.NodeTypeImport,
		Debug: tokens[*inx].Debug,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	if token.Type != lexer.TokenOpenBrace && token.Type != lexer.TokenIdentifier && token.Type != lexer.TokenFrom {
		return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected identifier, `from` or open brace, got %s", token.Type), token.Debug)
	}

	if token.Type == lexer.TokenIdentifier {
		node.Kind = "SINGLE"
		node.Content = token.Value

		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}
	} else if token.Type == lexer.TokenOpenBrace {
		node.Kind = "MULTIPLE"
		newNode, err := multiImportParser(ctx, node, tokens, inx)
		if err != nil {
			return nil, err
		}
		node = newNode
	} else {
		node.Kind = "NONE"
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenFrom {
		return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected keyword 'from', got %s", token.Type), token.Debug)
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenString {
		return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected string, got %s", token.Type), token.Debug)
	}

	node.Value = token.Value

	return skipSemi(tokens, inx, node), nil
}

func multiImportParser(ctx context.Context, node *astnode.Node, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
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
			if token.Type != lexer.TokenIdentifier {
				return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
			}

			name := token.Value
			value := token.Value

			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}

			token = tokens[*inx]
			if token.Type == lexer.TokenColon {
				if err := inxPP(tokens, inx); err != nil {
					return nil, err
				}

				token = tokens[*inx]

				if token.Type != lexer.TokenIdentifier {
					return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
				}

				value = token.Value

				if err := inxPP(tokens, inx); err != nil {
					return nil, err
				}
			}

			node.Children = append(node.Children, &astnode.Node{
				Type:    astnode.NodeTypeImport,
				Kind:    "CHILD",
				Content: name,
				Value:   value,
			})

			token = tokens[*inx]
			if token.Type == lexer.TokenComma {
				continue loop
			}

			if token.Type == lexer.TokenCloseBrace {
				if err := inxPP(tokens, inx); err != nil {
					return nil, err
				}
				return node, nil
			}
		}
	}
}
