package parsers

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func FnParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int, p parser, inline bool) (*astnode.Node, error) {
	node := &astnode.Node{
		Type: astnode.NodeTypeFunction,
	}

	if inline {
		node.Type = astnode.NodeTypeInlineFunction
	}

	if !inline {
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}

		token := tokens[*inx]
		if token.Type != lexer.TokenIdentifier {
			return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
		}

		node.Content = token.Value
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
			arg, last, err := fnArgumentParser(ctx, sn, tokens, inx)
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

	if err := inxPP(tokens, inx); err != nil {
		return nil, err
	}

	token = tokens[*inx]

	if token.Type == lexer.TokenFnReturnArrow {
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}

		retType, err := TypeParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		node.ValueType = retType
	}

	token = tokens[*inx]

	if token.Type == lexer.TokenArrow {
		if err := inxPP(tokens, inx); err != nil {
			return nil, err
		}

		body, err := ValueParser(ctx, sn, tokens, inx)
		if err != nil {
			return nil, err
		}

		node.Body = append(node.Body, &astnode.Node{
			Type:  astnode.NodeTypeReturn,
			Value: body,
			Flags: astnode.AppendFlags{"NODEVALUE"},
		})

		return node, nil
	}

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

	bodyNodes, err := p.Parse(body)
	if err != nil {
		return nil, err
	}

	node.Body = bodyNodes

	return node, nil
}

func fnArgumentParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, bool, error) {
	if err := inxPP(tokens, inx); err != nil {
		return nil, false, err
	}

	token := tokens[*inx]
	if token.Type == lexer.TokenCloseParen {
		*inx++
		return nil, true, nil
	}

	if token.Type != lexer.TokenIdentifier {
		return nil, false, newErr(ErrUnexpectedToken, fmt.Sprintf("expected identifier, got %s", token.Type), token.Debug)
	}
	node := &astnode.Node{
		Type:    astnode.NodeTypeFunctionArgument,
		Content: token.Value,
		Debug:   token.Debug,
	}

	if err := inxPP(tokens, inx); err != nil {
		return nil, false, err
	}

	token = tokens[*inx]
	if token.Type == lexer.TokenColon {
		if err := inxPP(tokens, inx); err != nil {
			return nil, false, err
		}

		typ, err := TypeParser(ctx, tokens, inx)
		if err != nil {
			return nil, false, err
		}
		node.ValueType = typ
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

	if token.Type == lexer.TokenAssign {
		if err := inxPP(tokens, inx); err != nil {
			return nil, false, err
		}
		val, err := ValueParser(ctx, sn, tokens, inx)
		if err != nil {
			return nil, false, err
		}
		*inx--

		node.FallbackValue = val
		if err := inxPP(tokens, inx); err != nil {
			return nil, false, err
		}

		token = tokens[*inx]
		if token.Type == lexer.TokenCloseParen {
			return node, true, nil
		}

		if token.Type == lexer.TokenComma {
			return node, false, nil
		}
	}

	if token.Type != lexer.TokenCloseParen {
		return nil, false, newErr(ErrUnexpectedToken, fmt.Sprintf("expected close parenthesis, got %s", token.Type), token.Debug)
	}

	return node, true, nil
}
