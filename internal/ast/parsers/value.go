package parsers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func ValueParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	if *inx >= len(tokens) {
		return nil, newErr(ErrUnexpectedToken, "unexpected end of input")
	}

	token := tokens[*inx]
	if token.Type == lexer.TokenLessThan && *inx+1 < len(tokens) && tokens[*inx+1].Type == lexer.TokenIdentifier {
		return HTMLParser(ctx, sn, tokens, inx)
	}

	if token.Type == lexer.TokenOpenBracket {
		return ListParser(ctx, sn, tokens, inx)
	}

	if token.Type == lexer.TokenIdentifier && token.Value == "dict" {
		return DictParser(ctx, sn, tokens, inx)
	}

	if token.Type == lexer.TokenOpenBrace {
		return DictParser(ctx, sn, tokens, inx)
	}

	node := &astnode.Node{
		Type:  astnode.NodeTypeExpression,
		Debug: token.Debug,
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

			if isWhite(token) {
				*inx++
				continue
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
			} else if token.Type != lexer.TokenWhiteSpace {
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

			*inx++
			if *inx >= len(tokens) {
				break loop
			}
		}
	}

	if parenCount != 0 {
		return nil, newErr(ErrSyntaxError, "unbalanced parentheses", token.Debug)
	}

	node.Body = body

	return node, nil
}

func singleValueParser(ctx context.Context, sn Parser_HTML, tokens []*lexer.Token, inx *int, token *lexer.Token) (*astnode.Node, error) {
	switch token.Type {
	default:
		return nil, newErr(ErrUnexpectedToken, fmt.Sprintf("expected value, got '%v'", token.Value), token.Debug)
	case lexer.TokenString:
		s := &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "STRING",
			Value: token.Value,
			Debug: token.Debug,
		}

		if token.Map["quote"] == "`" {
			s.Flags.Append("TEMPLATE")
			return parseTemplateLiteral(ctx, sn, s)
		}

		return s, nil
	case lexer.TokenBool:
		value, err := strconv.ParseBool(token.Value)
		if err != nil {
			return nil, newErr(ErrValueError, fmt.Sprintf("value '%s' is not a valid boolean", token.Value), token.Debug)
		}

		return &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "BOOLEAN",
			Value: value,
			Debug: token.Debug,
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
				Debug: token.Debug,
			}, nil
		}

		value, err := strconv.ParseInt(token.Value, token.Map["base"].(int), 64)
		if err != nil {
			return nil, newErr(ErrValueError, fmt.Sprintf("value '%s' is not a integer", token.Value), token.Debug)
		}

		return &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "INTEGER",
			Value: value,
			Debug: token.Debug,
		}, nil
	case lexer.TokenNil:
		return &astnode.Node{
			Type:  astnode.NodeTypeValue,
			Kind:  "NIL",
			Value: nil,
			Debug: token.Debug,
		}, nil
	case lexer.TokenIdentifier:
		id, err := TypeWholeIDParser(ctx, tokens, inx)
		if err != nil {
			return nil, err
		}

		last := *inx

		if err := inxPP(tokens, inx); err != nil {
			return &astnode.Node{
				Type:        astnode.NodeTypeValue,
				Kind:        "IDENTIFIER",
				Value:       id,
				IsReference: true,
				Debug:       token.Debug,
			}, nil
		}

		token := tokens[*inx]

		if token.Type == lexer.TokenOpenParen || token.Type == lexer.TokenOpenBracket {
			if token.Type == lexer.TokenOpenParen {
				n, err := fnCallParser(ctx, sn, id, tokens, inx)
				if err != nil {
					return nil, err
				}
				*inx--
				return n, nil
			} else {
				n, err := arrayKeyParser(ctx, sn, id, tokens, inx)
				if err != nil {
					return nil, err
				}
				*inx--
				return n, nil
			}
		}

		*inx = last

		return &astnode.Node{
			Type:        astnode.NodeTypeValue,
			Kind:        "IDENTIFIER",
			Value:       id,
			IsReference: true,
			Debug:       token.Debug,
		}, nil
	case lexer.TokenFn:
		n, err := FnParser(ctx, sn, tokens, inx, sn, true)
		if err != nil {
			return nil, err
		}
		*inx--
		return n, nil
	case lexer.TokenHtmlBlock:
		return HTMLBlockParser(ctx, sn, token)
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

func arrayKeyParser(ctx context.Context, sn Parser_HTML, id string, tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	node := &astnode.Node{
		Type:        astnode.NodeTypeValue,
		Kind:        "IDENTIFIER",
		Value:       id,
		IsReference: true,
		Debug:       tokens[*inx].Debug,
	}

loop:
	for *inx < len(tokens) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		tok := tokens[*inx]

		switch tok.Type {
		case lexer.TokenOpenBracket:
			start := *inx
			bracketCount := 1
			*inx++

			for *inx < len(tokens) && bracketCount > 0 {
				switch tokens[*inx].Type {
				case lexer.TokenOpenBracket:
					bracketCount++
				case lexer.TokenCloseBracket:
					bracketCount--
				}
				*inx++
			}

			if bracketCount != 0 {
				return nil, fmt.Errorf("unclosed [ in array access")
			}

			exprTokens := tokens[start+1 : *inx-1]
			newInx := 0
			valueNode, err := ValueParser(ctx, sn, exprTokens, &newInx)
			if err != nil {
				return nil, err
			}

			node.ArrayAccess = append(node.ArrayAccess, valueNode)

		case lexer.TokenDot:
			*inx++
			if *inx >= len(tokens) || tokens[*inx].Type != lexer.TokenIdentifier {
				return nil, fmt.Errorf("expected identifier after dot")
			}
			prop := tokens[*inx].Value
			*inx++
			propNode := &astnode.Node{
				Type:  astnode.NodeTypeValue,
				Kind:  "IDENTIFIER",
				Value: prop,
				Debug: tokens[*inx-1].Debug,
			}
			node.ArrayAccess = append(node.ArrayAccess, propNode)

		default:
			break loop
		}
	}

	if *inx < len(tokens) {
		tok := tokens[*inx]
		if tok.Type == lexer.TokenWhiteSpace {
			if err := inxPP(tokens, inx); err != nil {
				return nil, err
			}
		}
	}

	return node, nil
}
