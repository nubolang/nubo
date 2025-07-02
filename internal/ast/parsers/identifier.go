package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func IdentifierParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
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
			Content: id,
			Debug:   token.Debug,
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
		return fnCallParser(ctx, sn, id, tokens, inx)
	case lexer.TokenOpenBracket:
		access, err := arrayKeyParser(ctx, sn, id, tokens, inx)
		if err != nil {
			return nil, err
		}

		if *inx > len(tokens) {
			return nil, newErr(ErrUnexpectedEOF, "unexpected end of input", token.Debug)
		}

		token = tokens[*inx]

		switch token.Type {
		default:
			return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("unexpected token %s", token.Value), token.Debug)
		case lexer.TokenIncrement, lexer.TokenDecrement:
			node := &astnode.Node{
				Content: id,
				Debug:   token.Debug,
			}
			node.ArrayAccess = access.ArrayAccess

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
			node.ArrayAccess = access.ArrayAccess

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
		}
	}

	return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("unexpected token %s", token.Value), token.Debug)
}
