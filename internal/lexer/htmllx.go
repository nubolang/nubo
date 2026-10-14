package lexer

import (
	"strings"
	"unicode"

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

	lx.tokens = append(lx.tokens, newToken(t, val, val, m, lx.file, lx.line, lx.col))
}

func (lx *HtmlLexer) addAt(t TokenType, val string, start, line, col int, ma ...map[string]any) {
	var m map[string]any
	if len(ma) > 0 {
		m = ma[0]
	}
	source := string(lx.input[start:lx.pos])
	lx.tokens = append(lx.tokens, newToken(t, val, source, m, lx.file, line, col))
}

func (lx *HtmlLexer) Lex() ([]*Token, error) {
	for lx.curr() != 0 {
		ch := lx.curr()
		if ch == '<' {
			lx.lexTag()
		} else if ch == '!' && lx.peek(1) == '{' {
			if err := lx.lexUnescapedBrace(); err != nil {
				return nil, err
			}
		} else if ch == '{' {
			if err := lx.lexBrace(); err != nil {
				return nil, err
			}
		} else {
			lx.lexText()
		}
	}
	return lx.tokens, nil
}

func (lx *HtmlLexer) lexTag() {
	if lx.peek(1) == '/' {
		lx.add(TokenClosingStartTag, "</")
		lx.advance()
		lx.advance()
	} else {
		lx.add(TokenLessThan, "<")
		lx.advance()
	}

	// tag name (letters, numbers, _, .)
	readIdent := func() {
		start, startLine, startCol := lx.pos, lx.line, lx.col
		for unicode.IsLetter(lx.curr()) || unicode.IsDigit(lx.curr()) {
			lx.advance()
		}
		if start != lx.pos {
			lx.addAt(TokenIdentifier, string(lx.input[start:lx.pos]), start, startLine, startCol)
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
				start, startLine, startCol := lx.pos, lx.line, lx.col
				for unicode.IsLetter(lx.curr()) || unicode.IsDigit(lx.curr()) || lx.curr() == '_' || lx.curr() == '-' {
					lx.advance()
				}
				lx.addAt(TokenIdentifier, string(lx.input[start:lx.pos]), start, startLine, startCol)
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
	tokenStart, tokenLine, tokenCol := lx.pos, lx.line, lx.col
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
	lx.addAt(TokenString, val, tokenStart, tokenLine, tokenCol, map[string]any{
		"quote": string(quote),
	})
}

func (lx *HtmlLexer) lexUnescapedBrace() error {
	// consume '!{'
	lx.add(TokenUnescapedBrace, "!{")
	lx.advance()
	lx.advance()

	// consume until matching '}'
	depth := 1
	start, startLine, startCol := lx.pos, lx.line, lx.col
	for lx.curr() != 0 && depth > 0 {
		if lx.curr() == '{' {
			depth++
		} else if lx.curr() == '}' {
			depth--
			if depth == 0 {
				break
			}
		}
		lx.advance()
	}
	if depth != 0 {
		return newErr(ErrSyntaxError, "unclosed dynamic text", &debug.Debug{Line: lx.line, Column: lx.col, ColumnEnd: lx.col, File: lx.file})
	}
	content := string(lx.input[start:lx.pos])
	if err := lx.htmlContent(content, startLine, startCol); err != nil {
		return err
	}
	lx.add(TokenCloseBrace, "}")
	lx.advance()
	return nil
}

func (lx *HtmlLexer) lexBrace() error {
	// consume '{'
	lx.add(TokenOpenBrace, "{")
	lx.advance()

	// consume until matching '}'
	depth := 1
	start, startLine, startCol := lx.pos, lx.line, lx.col
	for lx.curr() != 0 && depth > 0 {
		if lx.curr() == '{' {
			depth++
		} else if lx.curr() == '}' {
			depth--
			if depth == 0 {
				break
			}
		}
		lx.advance()
	}
	if depth != 0 {
		return newErr(ErrSyntaxError, "unclosed dynamic text", &debug.Debug{Line: lx.line, Column: lx.col, ColumnEnd: lx.col, File: lx.file})
	}
	content := string(lx.input[start:lx.pos])
	if err := lx.htmlContent(content, startLine, startCol); err != nil {
		return err
	}
	lx.add(TokenCloseBrace, "}")
	lx.advance()
	return nil
}

func (lx *HtmlLexer) lexText() {
	start, startLine, startCol := lx.pos, lx.line, lx.col
	for lx.curr() != 0 && lx.curr() != '<' && !(lx.curr() == '!' && lx.peek(1) == '{') && lx.curr() != '{' {
		lx.advance()
	}
	if start != lx.pos {
		lx.addAt(TokenHtmlText, string(lx.input[start:lx.pos]), start, startLine, startCol)
	}
}

func (lx *HtmlLexer) htmlContent(content string, line, col int) error {
	nuboLx, err := New(strings.NewReader(content), lx.debug.File)
	if err != nil {
		return err
	}

	nuboLx.SetDebugLineColData(line, col)
	tokens, err := nuboLx.Parse()
	if err != nil {
		return err
	}

	lx.tokens = append(lx.tokens, tokens...)
	return nil
}
