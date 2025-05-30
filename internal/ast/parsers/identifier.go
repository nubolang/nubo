package parsers

import (
	"context"
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func IdentifierParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	firstInx := *inx

	id, err := TypeWholeIDParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]

	switch token.Type {
	case lexer.TokenIncrement, lexer.TokenDecrement:
		node := &astnode.Node{
			Value: id,
		}
		if token.Type == lexer.TokenIncrement {
			node.Type = astnode.NodeTypeIncrement
		} else {
			node.Type = astnode.NodeTypeDecrement
		}

		return skipSemi(tokens, inx, node), nil
	case lexer.TokenAssign:
		node := &astnode.Node{
			Type:    astnode.NodeTypeAssign,
			Content: id,
		}

		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}

		expr, err := ValueParser(ctx, sn, tokens, inx)
		if err != nil {
			return nil, err
		}

		node.Value = expr
		node.Flags.Append("NODEVALUE")

		return skipSemi(tokens, inx, node), nil
	case lexer.TokenOpenParen:
		*inx = firstInx
		return fnCallParser(ctx, sn, tokens, inx)
	}

	return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("unexpected token %s", token.Value), token.Debug)
}

func fnCallParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	id, err := TypeWholeIDParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	node := &astnode.Node{
		Type:    astnode.NodeTypeFunctionCall,
		Content: id,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
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
			arg, last, err := fnCallArgumentParser(ctx, sn, tokens, inx)
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

	_ = inxPP(tokens, inx)

	return skipSemi(tokens, inx, node), nil
}

func fnCallArgumentParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, bool, error) {
	if err := inxPP(tokens, inx); err != nil {
		return nil, false, err
	}

	token := tokens[*inx]
	if token.Type == lexer.TokenCloseParen {
		*inx++
		return nil, true, nil
	}

	value, err := ValueParser(ctx, sn, tokens, inx)
	if err != nil {
		return nil, false, err
	}

	node := &astnode.Node{
		Type:  astnode.NodeTypeFunctionArgument,
		Value: value,
	}
	node.Flags.Append("NODEVALUE")

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
