package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func EventParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: astnode.NodeTypeEvent,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
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
		return nil, newErr(ErrUnexpectedToken, "expected open parenthesis", token.Debug)
	}

	args := make([]*astnode.Node, 0)

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			arg, last, err := eventArgumentParser(ctx, tokens, inx)
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

	return skipSemi(tokens, inx, node), nil
}

func eventArgumentParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, bool, error) {
	if err := inxPP(tokens, inx); err != nil {
		return nil, false, err
	}

	token := tokens[*inx]
	if token.Type == lexer.TokenCloseParen {
		return nil, true, nil
	}

	if token.Type != lexer.TokenIdentifier {
		return nil, false, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
	}
	node := &astnode.Node{
		Type:    astnode.NodeTypeEventArgument,
		Content: token.Value,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, false, err
	}

	token = tokens[*inx]
	if token.Type != lexer.TokenColon {
		return nil, false, newErr(ErrUnexpectedToken, "expected colon", token.Debug)
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, false, err
	}

	typ, err := TypeParser(ctx, tokens, inx)
	if err != nil {
		return nil, false, err
	}
	node.ValueType = typ

	token = tokens[*inx]
	if token.Type == lexer.TokenComma {
		if err := inxPP(tokens, inx); err != nil {
			return nil, false, err
		}
		token = tokens[*inx]
		if token.Type == lexer.TokenCloseParen {
			return nil, false, newErr(ErrUnexpectedToken, "expected identifier but got close parenthesis", token.Debug)
		}

		*inx--
		return node, false, nil
	}

	if token.Type != lexer.TokenCloseParen {
		return nil, false, newErr(ErrUnexpectedToken, fmt.Sprintf("expected close parenthesis, got %s", token.Type), token.Debug)
	}

	return node, true, nil
}
