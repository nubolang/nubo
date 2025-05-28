package parsers

import (
	"context"
	"fmt"
	"strings"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func TypeParser(ctx context.Context, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: astnode.NodeTypeType,
	}

	if tokens[*inx].Type == lexer.TokenFn {
		return TypeFnParser(ctx, node, tokens, inx)
	}

	if tokens[*inx].Type == lexer.TokenIdentifier {
		id, err := TypeWholeIDParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		node.Content = id

		switch node.Content {
		case "dict":
			return TypeDictParser(ctx, node, tokens, inx)
		case "ref":
			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}
			node, err := TypeParser(ctx, tokens, inx)
			if err != nil {
				return nil, err
			}
			node.Kind = "REF"
			return node, nil
		}

		*inx++

		if *inx < len(tokens) && tokens[*inx].Type == lexer.TokenQuestion {
			node.Flags = append(node.Flags, "OPTIONAL")
			*inx++
		}

		return node, nil
	}

	if tokens[*inx].Type == lexer.TokenOpenBracket && *inx+1 < len(tokens) && tokens[*inx+1].Type == lexer.TokenCloseBracket {
		*inx += 2
		typ, err := TypeParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		node.Body = append(node.Body, typ)
		node.Kind = "LIST"
		return node, nil
	}

	return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", tokens[*inx].Type), tokens[*inx].Debug)
}

func TypeDictParser(ctx context.Context, node *astnode.Node, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node.Kind = "DICT"

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]
	if token.Type != lexer.TokenOpenBracket {
		return node, nil
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]

	type1, err := TypeParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.Body = append(node.Body, type1)

	if err := inxPPIf(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]

	if token.Type != lexer.TokenComma {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ',', got: %v", token.Value), token.Debug)
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	type2, err := TypeParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.Body = append(node.Body, type2)

	if err := inxPPIf(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]

	if token.Type != lexer.TokenCloseBracket {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ']', got: %v", token.Value), token.Debug)
	}

	safeIncr(tokens, inx)

	return node, nil
}

func TypeWholeIDParser(ctx context.Context, tokens []*lexer.Token, inx *int) (string, error) {
	var res strings.Builder

loop:
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			if *inx >= len(tokens) {
				break loop
			}

			token := tokens[*inx]

			if token.Type != lexer.TokenIdentifier {
				return "", newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got: %v", token.Value), token.Debug)
			}

			res.WriteString(token.Value)

			if err := inxPP(tokens, inx); err != nil {
				return "", err
			}

			token = tokens[*inx]

			if token.Type != lexer.TokenDot {
				*inx--
				break loop
			}

			res.WriteString(".")
			*inx++
		}
	}

	return res.String(), nil
}

func TypeFnParser(ctx context.Context, node *astnode.Node, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node.Kind = "FUNCTION"

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token := tokens[*inx]

	if token.Type != lexer.TokenOpenParen {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected '(', got: %v", token.Value), token.Debug)
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]

	if token.Type != lexer.TokenCloseParen {
		if err := typeFnArgParser(ctx, node, tokens, inx); err != nil {
			return nil, err
		}
	} else {
		*inx--
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]

	if token.Type != lexer.TokenCloseParen {
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected ')', got: %v", token.Value), token.Debug)
	}

	if err := inxPP(tokens, inx); err != nil {
		return node, nil
	}

	token = tokens[*inx]

	if token.Type != lexer.TokenFnReturnArrow {
		return node, nil
	}

	if err := inxPP(tokens, inx); err != nil {
		return node, err
	}

	token = tokens[*inx]
	ret, err := TypeParser(ctx, tokens, inx)
	if err != nil {
		return nil, err
	}

	node.ValueType = ret

	return node, nil
}

func typeFnArgParser(ctx context.Context, node *astnode.Node, tokens []*lexer.Token, inx *int) error {
loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if *inx >= len(tokens) {
				break loop
			}

			typ, err := TypeParser(ctx, tokens, inx)
			if err != nil {
				return err
			}

			node.Args = append(node.Args, typ)

			token := tokens[*inx]

			if token.Type != lexer.TokenComma {
				*inx--
				break loop
			}

			if err := inxPP(tokens, inx); err != nil {
				return err
			}
		}
	}

	return nil
}
