package parsers

import (
	"context"

	"github.com/bndrmrtn/tea/internal/ast/astnode"
	"github.com/bndrmrtn/tea/internal/lexer"
)

func EventParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: lexer.TokenEvent,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	if token.Type != lexer.TokenIdentifier {
		return nil, newErr(ErrUnexpectedToken, "expected identifier", token.Debug)
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
			arg, last, err := eventArgumentParser(tokens, inx)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if last {
				break loop
			}
		}
	}

	node.Args = args

	return skipSemi(tokens, inx, node), nil
}

func eventArgumentParser(tokens []*lexer.Token, inx *int) (*astnode.Node, bool, error) {
	if err := inxPP(tokens, inx); err != nil {
		return nil, false, err
	}

	token := tokens[*inx]
	if token.Type != lexer.TokenIdentifier {
		return nil, false, newErr(ErrUnexpectedToken, "expected identifier", token.Debug)
	}
	node := &astnode.Node{
		Type:    lexer.TokenIdentifier,
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

	token = tokens[*inx]
	if token.Type != lexer.TokenIdentifier {
		return nil, false, newErr(ErrUnexpectedToken, "expected identifier", token.Debug)
	}
	node.Value = token.Value

	if err := inxPP(tokens, inx); err != nil {
		return nil, false, err
	}

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
		return nil, false, newErr(ErrUnexpectedToken, "expected close parenthesis", token.Debug)
	}

	return node, true, nil
}
