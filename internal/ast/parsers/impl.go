package parsers

import (
	"context"
	"errors"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/lexer"
)

func ImplParser(ctx context.Context, a Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	node := &astnode.Node{
		Type:  astnode.NodeTypeImpl,
		Debug: token.Debug,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected identifier, got %s", token.Type)
	}
	node.Content = token.Value

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}
	token = tokens[*inx]

	if token.Type != lexer.TokenOpenBrace {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '{', got %s", token.Type), token.Debug)
	}

	var (
		body       []*lexer.Token
		braceCount = 1
	)

bodyloop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			*inx++

			if *inx >= len(tokens) {
				return nil, newErr(ErrUnexpectedToken, "unexpected end of input", token.Debug)
			}

			token = tokens[*inx]

			if token.Type == lexer.TokenCloseBrace {
				braceCount--
				if braceCount == 0 {
					*inx++
					break bodyloop
				}
			} else if token.Type == lexer.TokenOpenBrace || token.Type == lexer.TokenUnescapedBrace {
				braceCount++
			}

			body = append(body, token)
		}
	}

	if braceCount != 0 {
		return nil, newErr(ErrUnexpectedToken, "unbalanced braces", token.Debug)
	}

	bodyNodes, err := implBodyParser(ctx, a, body)
	if err != nil {
		return nil, err
	}

	node.Body = bodyNodes

	return node, nil
}

func implBodyParser(ctx context.Context, a Parser_HTML, tokens []*lexer.Token) ([]*astnode.Node, error) {
	var (
		inx   int
		nodes []*astnode.Node
	)

	for inx < len(tokens) {
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			node, err := implBodyPartParser(ctx, a, tokens, &inx)
			if err != nil {
				return nil, err
			}

			if node != nil {
				nodes = append(nodes, node)
			}
		}
	}

	return nodes, nil
}

func implBodyPartParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	switch token.Type {
	case lexer.TokenFn:
		node, err := FnParser(ctx, sn, tokens, inx, sn, false)
		if err != nil {
			return nil, err
		}
		return node, nil
	}

	if token.Type == lexer.TokenWhiteSpace || token.Type == lexer.TokenNewLine || token.Type == lexer.TokenSingleLineComment || token.Type == lexer.TokenMultiLineComment {
		*inx++
		return nil, nil
	}

	return nil, debug.NewError(errors.New("Ast error"), fmt.Sprintf("Unhandled node: %s", token.Type), token.Debug)
}
