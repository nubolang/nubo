package parsers

import (
	"context"
	"fmt"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func VariableParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]
	isConst := token.Type == lexer.TokenConst

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	node := &astnode.Node{
		Type: astnode.NodeTypeVariableDecl,
	}

	if isConst {
		node.Kind = "CONST"
	} else {
		node.Kind = "LET"
	}

	token = tokens[*inx]

	if token.Type != lexer.TokenIdentifier {
		return nil, newErr(ErrSyntaxError, fmt.Sprintf("expected identifier, got %s", token.Value))
	}

	node.Content = token.Value

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]

	if token.Type == lexer.TokenColon {
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}

		typ, err := TypeParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}

		node.ValueType = typ
	}

	token = tokens[*inx]

	if token.Type == lexer.TokenAssign {
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}

		value, err := ValueParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}

		node.Value = value
		node.Flags.Append("NODEVALUE")
	}

	return skipSemi(tokens, inx, node), nil
}
