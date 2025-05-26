package parsers

import (
	"context"
	"fmt"

	"github.com/bndrmrtn/tea/internal/ast/astnode"
	"github.com/bndrmrtn/tea/internal/lexer"
)

func TypeParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: lexer.TokenIdentifier,
		Kind: "TYPE",
	}

	if tokens[*inx].Type == lexer.TokenIdentifier {
		node.Content = tokens[*inx].Value

		switch node.Content {
		case "map":
			break
		case "fn":
			break
		case "ref":
			break
		}

		*inx++

		if *inx < len(tokens) && tokens[*inx].Type == lexer.TokenQuestion {
			node.Flags = append(node.Flags, "OPTIONAL")
			*inx++
		}

		return node, nil
	}

	return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier"), tokens[*inx].Debug)
}
