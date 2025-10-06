package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nubolang/nubo/internal/debug"
)

type HtmlLexer struct {
	input  []rune
	pos    int
	tokens []*Token
	debug  *debug.Debug

	file string
	line int
	col  int
}

func NewHtmlLexer(input string, dg *debug.Debug) *HtmlLexer {
	return &HtmlLexer{input: []rune(input), debug: dg, file: dg.File, line: dg.Line, col: dg.Column}
}

func (lx *HtmlLexer) curr() rune {
	if lx.pos >= len(lx.input) {
		return 0
	}
	return lx.input[lx.pos]
}

func (lx *HtmlLexer) peek(n int) rune {
	idx := lx.pos + n
	if idx >= len(lx.input) || idx < 0 {
		return 0
	}
	return lx.input[idx]
}

func (lx *HtmlLexer) advance() {
	if lx.pos < len(lx.input) {
		if lx.curr() == '\n' {
			lx.line++
			lx.col = 1
		} else {
			lx.col++
		}
		lx.pos++
	}
}

func (lx *HtmlLexer) add(t TokenType, val string, ma ...map[string]any) {
	var m map[string]any
	if len(ma) > 0 {
		m = ma[0]
	}

	end := lx.col + utf8.RuneCountInString(val) - 1
	lx.tokens = append(lx.tokens, &Token{Type: t, Value: val, Debug: &debug.Debug{
		File:      lx.file,
		Line:      lx.line,
		Column:    lx.col,
		ColumnEnd: end,
	}, Map: m})
}

func (lx *HtmlLexer) Lex() ([]*Token, error) {
	for lx.curr() != 0 {
		ch := lx.curr()
		if ch == '<' {
			lx.lexTag()
		} else if ch == '!' && lx.peek(1) == '{' {
			lx.lexUnescapedBrace()
		} else if ch == '{' {
			lx.lexBrace()
		} else {
			lx.lexText()
		}
	}
	return lx.tokens, nil
}

func (lx *HtmlLexer) lexTag() {
	lx.advance()

	if lx.curr() == '/' {
		lx.add(TokenClosingStartTag, "</")
		lx.advance()
	} else {
		lx.add(TokenLessThan, "<")
	}

	// tag name (letters, numbers, _, .)
	readIdent := func() {
		start := lx.pos
		for unicode.IsLetter(lx.curr()) || unicode.IsDigit(lx.curr()) {
			lx.advance()
		}
		if start != lx.pos {
			lx.add(TokenIdentifier, string(lx.input[start:lx.pos]))
		}
	}

	readIdent()

	// dot‑separated parts, e.g. Nubo.Dashboard
	for lx.curr() == '.' {
		lx.add(TokenDot, ".")
		lx.advance()
		readIdent()
	}

	// inside tag: attributes, spaces, punctuation
	for lx.curr() != 0 && lx.curr() != '>' && !(lx.curr() == '/' && lx.peek(1) == '>') {
		ch := lx.curr()
		switch {
		case unicode.IsSpace(ch):
			lx.add(TokenWhiteSpace, string(ch))
			lx.advance()
		case ch == '=':
			lx.add(TokenAssign, "=")
			lx.advance()
		case ch == ':':
			lx.add(TokenColon, string(ch))
			lx.advance()
		case ch == '-':
			lx.add(TokenMinus, string(ch))
			lx.advance()
		case ch == '"' || ch == '\'':
			lx.lexString()
		default:
			if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
				start := lx.pos
				for unicode.IsLetter(lx.curr()) || unicode.IsDigit(lx.curr()) || lx.curr() == '_' || lx.curr() == '-' {
					lx.advance()
				}
				lx.add(TokenIdentifier, string(lx.input[start:lx.pos]))
			} else {
				lx.add(TokenUnknown, string(ch))
				lx.advance()
			}
		}
	}

	// handle '/>' self-closing
	if lx.curr() == '/' && lx.peek(1) == '>' {
		lx.add(TokenSelfClosingTag, "/>")
		lx.advance()
		lx.advance()
		return
	}

	// handle '>'
	if lx.curr() == '>' {
		lx.add(TokenGreaterThan, ">")
		lx.advance()
	}
}

func (lx *HtmlLexer) lexString() {
	quote := lx.curr()
	lx.advance()
	start := lx.pos
	for lx.curr() != 0 && lx.curr() != quote {
		lx.advance()
	}
	val := string(lx.input[start:lx.pos])
	if lx.curr() == quote {
		lx.advance()
	}
	lx.add(TokenString, val, map[string]any{
		"quote": string(quote),
	})
}

func (lx *HtmlLexer) lexUnescapedBrace() {
	// consume '!{'
	lx.advance()
	lx.advance()
	lx.add(TokenUnescapedBrace, "!{")

	// consume until matching '}'
	depth := 1
	start := lx.pos
	for lx.curr() != 0 && depth > 0 {
		if lx.curr() == '{' {
			depth++
		} else if lx.curr() == '}' {
			depth--
		}
		lx.advance()
	}
	content := string(lx.input[start : lx.pos-1])
	lx.htmlContent(content)
	lx.add(TokenCloseBrace, "}")
}

func (lx *HtmlLexer) lexBrace() {
	// consume '{'
	lx.advance()
	lx.add(TokenOpenBrace, "{")

	// consume until matching '}'
	depth := 1
	start := lx.pos
	for lx.curr() != 0 && depth > 0 {
		if lx.curr() == '{' {
			depth++
		} else if lx.curr() == '}' {
			depth--
		}
		lx.advance()
	}
	content := string(lx.input[start : lx.pos-1])
	lx.htmlContent(content)
	lx.add(TokenCloseBrace, "}")
}

func (lx *HtmlLexer) lexText() {
	start := lx.pos
	for lx.curr() != 0 && lx.curr() != '<' && !(lx.curr() == '!' && lx.peek(1) == '{') && lx.curr() != '{' {
		lx.advance()
	}
	if start != lx.pos {
		lx.add(TokenHtmlText, string(lx.input[start:lx.pos]))
	}
}

func (lx *HtmlLexer) htmlContent(content string) error {
	nuboLx, err := New(strings.NewReader(content), lx.debug.File)
	if err != nil {
		return err
	}

	tokens, err := nuboLx.Parse()
	if err != nil {
		return err
	}

	lx.tokens = append(lx.tokens, tokens...)
	return nil
}
