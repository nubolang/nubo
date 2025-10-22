package codehighlight

import (
	"io"
	"slices"
	"strings"

	"github.com/nubolang/nubo/internal/lexer"
)

type Highlight struct {
	tokens []*lexer.Token
}

func NewHighlight(code io.Reader) (*Highlight, error) {
	lx, err := lexer.New(code, "<codehighlight>")
	if err != nil {
		return nil, err
	}

	tokens, err := lx.Parse()
	if err != nil {
		return nil, err
	}

	return &Highlight{
		tokens: tokens,
	}, nil
}

func (h *Highlight) HighlightConsole() (string, error) {
	return h.highlight(ModeConsole)
}

func (h *Highlight) HighlightHTML() (string, error) {
	return h.highlight(ModeHTML)
}

func (h *Highlight) highlight(mode Mode) (string, error) {
	var sb strings.Builder

	for i, token := range h.tokens {
		s, err := h.highlightToken(mode, i, token)
		if err != nil {
			return "", err
		}
		sb.WriteString(s)
	}

	return sb.String(), nil
}

func (h *Highlight) highlightToken(mode Mode, i int, token *lexer.Token) (string, error) {
	switch token.Type {
	default:
		if slices.Contains(lexer.Operators(), token.Type) {
			return highlightOperator(mode, token.Value), nil
		}
		return highlightUnknown(mode, token.Value), nil
	case lexer.TokenLet, lexer.TokenConst, lexer.TokenFn, lexer.TokenReturn, lexer.TokenStruct,
		lexer.TokenImpl, lexer.TokenFor, lexer.TokenWhile, lexer.TokenImport, lexer.TokenFrom,
		lexer.TokenIf, lexer.TokenElse, lexer.TokenIn, lexer.TokenDefer, lexer.TokenContinue,
		lexer.TokenBreak, lexer.TokenInclude, lexer.TokenEvent, lexer.TokenPub, lexer.TokenSub, lexer.TokenPrivate:
		return highlightKeyword(mode, token.Value), nil
	case lexer.TokenOpenBrace, lexer.TokenCloseBrace, lexer.TokenOpenParen, lexer.TokenCloseParen,
		lexer.TokenOpenBracket, lexer.TokenCloseBracket:
		return highlightBracket(mode, token.Value), nil
	case lexer.TokenIdentifier:
		next := h.nextToken(i)
		if next.Type == lexer.TokenOpenParen {
			return highlightFunction(mode, token.Value), nil
		}
		return highlightIdentifier(mode, token.Value), nil
	case lexer.TokenNumber:
		return highlightNumber(mode, token.Value, token.Map["base"].(int)), nil
	case lexer.TokenNil, lexer.TokenBool:
		return highlightNumeric(mode, token.Value), nil
	case lexer.TokenString:
		return highlightString(mode, token), nil
	case lexer.TokenHtmlBlock:
		html, err := lexer.NewHtmlLexer(token.Value, token.Debug).Lex()
		if err != nil {
			return "", err
		}
		return h.highlightHtmlCode(mode, html)
	}
}

func (h *Highlight) nextToken(i int) *lexer.Token {
	var next *lexer.Token

	if i+1 < len(h.tokens) {
		next = h.tokens[i+1]
		if next.Type == lexer.TokenWhiteSpace || next.Type == lexer.TokenNewLine || next.Type == lexer.TokenSingleLineComment || next.Type == lexer.TokenMultiLineComment {
			next = h.nextToken(i + 1)
		}
	}

	return next
}
