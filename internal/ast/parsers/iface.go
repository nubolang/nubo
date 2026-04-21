package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/lexer"
)

func IfaceParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	node := &astnode.Node{
		Type:  astnode.NodeTypeType,
		Kind:  "IFACE",
		Debug: token.Debug,
	}

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

	bodyNodes, err := ifaceBodyParser(ctx, body)
	if err != nil {
		return nil, err
	}

	node.Body = bodyNodes

	return node, nil
}

func ifaceBodyParser(ctx context.Context, tokens []*lexer.Token) ([]*astnode.Node, error) {
	var (
		inx   int
		nodes []*astnode.Node
	)

	for inx < len(tokens) {
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			node, err := ifaceBodyPartParser(ctx, tokens, &inx)
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

func ifaceBodyPartParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	switch token.Type {
	case lexer.TokenPrivate:
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}
		node, err := ifaceFnParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		node.Flags.Append("PRIVATE")
		return node, nil
	case lexer.TokenIdentifier:
		node, err := ifaceFnParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	}

	if token.Type == lexer.TokenWhiteSpace || token.Type == lexer.TokenNewLine || token.Type == lexer.TokenSingleLineComment || token.Type == lexer.TokenMultiLineComment {
		*inx++
		return nil, nil
	}

	return nil, exception.Create("unhandled node: %s: '%s'", token.Type, token.Value).WithDebug(token.Debug).WithLevel(exception.LevelSemantic)
}

func ifaceFnParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type:  astnode.NodeTypeFunction,
		Debug: tokens[*inx].Debug,
	}

	token := tokens[*inx]
	if token.Type != lexer.TokenIdentifier {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
	}

	node.Content = token.Value

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenOpenParen {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '(', got %s", token.Type), token.Debug)
	}

	args := make([]*astnode.Node, 0)

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			arg, last, err := fnArgumentParser(ctx, nil, tokens, inx, true)
			if err != nil {
				return nil, err
			}
			if arg != nil {
				args = append(args, arg)
			}
			if last {
				break loop
			}
		}
	}

	node.Args = args
	ifaceReturnTypeErr := newErr(ErrUnexpectedEOF, "iface expects a valid return type for declarations")

	switch tokens[*inx].Type {
	case lexer.TokenNewLine:
		return nil, ifaceReturnTypeErr
	default:
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}
		if tokens[*inx].Type == lexer.TokenNewLine {
			return nil, ifaceReturnTypeErr
		}
	}

	token = tokens[*inx]

	if token.Type == lexer.TokenArrow || token.Type == lexer.TokenOpenBrace {
		return nil, newErr(ErrUnexpectedToken, "iface expects no fn body, just type declaration")
	}

	retType, err := TypeParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}
	node.ValueType = retType
	if *inx < len(tokens) && tokens[*inx].Type == lexer.TokenWhiteSpace {
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}
	}

	return node, nil
}
