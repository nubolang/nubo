package parsers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

func ValueParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	if *inx >= len(tokens) {
		return nil, newErr(ErrUnexpectedToken, "unexpected end of input")
	}

	token := tokens[*inx]
	if token.Type == lexer.TokenLessThan && *inx+1 < len(tokens) && tokens[*inx+1].Type == lexer.TokenIdentifier {
		return HTMLParser(ctx, sn, tokens, inx)
	}

	node := &astnode.Node{
		Type: astnode.NodeTypeExpression,
	}

	var (
		body       []*astnode.Node
		parenCount = 0
	)

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if *inx >= len(tokens) {
				break loop
			}

			token = tokens[*inx]
			if token.Type == lexer.TokenNewLine || token.Type == lexer.TokenSemicolon {
				*inx++
				break loop
			}

			var (
				err   error
				value *astnode.Node
			)

			if token.Type == lexer.TokenOpenParen || token.Type == lexer.TokenCloseParen {
				if token.Type == lexer.TokenOpenParen {
					parenCount++
				} else {
					parenCount--
				}
				if parenCount < 0 {
					return nil, newErr(ErrSyntaxError, "unbalanced parentheses", token.Debug)
				}
				body = append(body, &astnode.Node{
					Type: astnode.NodeTypeOperator,
					Kind: token.Value,
				})
			} else if isBinaryOperator(token.Type) {
				value = &astnode.Node{
					Type: astnode.NodeTypeOperator,
					Kind: token.Value,
				}
				body = append(body, value)
			} else {
				value, err = singleValueParser(ctx, sn, tokens, inx, token)
				if err != nil {
					return nil, err
				}
				body = append(body, value)

				last := *inx
				if err := inxPP(tokens, inx); err == nil {
					if !isBinaryOperator(tokens[*inx].Type) && parenCount == 0 {
						break loop
					}
				}
				*inx = last
			}

			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}
		}
	}

	if parenCount != 0 {
		return nil, newErr(ErrSyntaxError, "unbalanced parentheses", token.Debug)
	}

	node.Body = body
	return node, nil
}

func singleValueParser(ctx context.Context, sn HTMLAttrValueParser, tokens []*lexer.Token, inx *int, token *lexer.Token) (*astnode.Node, error) {
	switch token.Type {
	default:
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected value, got '%v'", token.Value), token.Debug)
	case lexer.TokenString:
		return &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "STRING",
			Value: token.Value,
		}, nil
	case lexer.TokenBool:
		value, err := strconv.ParseBool(token.Value)
		if err != nil {
			return nil, newErr(ErrValueError, fmt.Sprintf("value '%s' is not a valid boolean", token.Value), token.Debug)
		}

		return &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "BOOLEAN",
			Value: value,
		}, nil
	case lexer.TokenNumber:
		isFloat, ok := token.Map["isFloat"].(bool)
		if !ok {
			return nil, newErr(ErrUnexpectedToken, "unexpected token", token.Debug)
		}

		if isFloat {
			value, err := strconv.ParseFloat(token.Value, 64)
			if err != nil {
				return nil, newErr(ErrValueError, fmt.Sprintf("value '%s' is not a valid float", token.Value), token.Debug)
			}

			return &astnode.Node{
				Type:  astnode.NodeTypeValue,
				Kind:  "FLOAT",
				Value: value,
			}, nil
		}

		value, err := strconv.Atoi(token.Value)
		if err != nil {
			return nil, newErr(ErrValueError, fmt.Sprintf("value '%s' is not a integer", token.Value), token.Debug)
		}

		return &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "INTEGER",
			Value: value,
		}, nil
	case lexer.TokenNil:
		return &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "NIL",
			Value: nil,
		}, nil
	case lexer.TokenIdentifier:
		firstInx := *inx

		id, err := TypeWholeIDParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}

		if err := inxPP(tokens, inx); err == nil {
			tkn := tokens[*inx]
			if tkn.Type == lexer.TokenOpenParen {
				*inx = firstInx
				return fnCallParser(ctx, sn, tokens, inx)
			}
			*inx--
		}

		return &astnode.Node{
			Type:        astnode.NodeTypeValue,
			Kind:        "IDENTIFIER",
			Value:       id,
			IsReference: true,
		}, nil
	}
}

func isBinaryOperator(typ lexer.TokenType) bool {
	switch typ {
	default:
		return false
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenAsterisk,
		lexer.TokenPower, lexer.TokenSlash, lexer.TokenPercent, lexer.TokenIn,

		lexer.TokenEqual, lexer.TokenNotEqual,
		lexer.TokenLessThan, lexer.TokenLessEqual,
		lexer.TokenGreaterThan, lexer.TokenGreaterEqual,
		lexer.TokenQuestion, lexer.TokenColon,
		lexer.TokenAnd, lexer.TokenOr, lexer.TokenNot:
		return true
	}
}
