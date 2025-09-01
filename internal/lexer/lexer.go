package lexer

import (
	"io"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nubolang/nubo/internal/debug"
)

// Lexer performs rune‑oriented lexical analysis.
type Lexer struct {
	input  []rune
	pos    int // absolute rune index
	line   int // 1‑based
	col    int // 1‑based (runes, NOT bytes)
	file   string
	tokens []*Token
}

// New returns a new lexer initialised for s.
func New(r io.Reader, file string) (*Lexer, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return &Lexer{
		input: []rune(strings.ReplaceAll(string(data), "\r", "")),
		file:  filepath.Clean(file),
		line:  1,
		col:   1,
	}, nil
}

// ---------- small helpers ----------

func (lx *Lexer) curr() rune {
	if lx.pos >= len(lx.input) {
		return 0
	}
	return lx.input[lx.pos]
}

func (lx *Lexer) peek(n int) rune {
	idx := lx.pos + n
	if idx >= len(lx.input) || idx < 0 {
		return 0
	}
	return lx.input[idx]
}

func (lx *Lexer) prev() rune { return lx.peek(-1) }

func (lx *Lexer) advance() {
	if lx.pos >= len(lx.input) {
		return
	}
	if lx.curr() == '\n' {
		lx.line++
		lx.col = 1
	} else {
		lx.col++
	}
	lx.pos++
}

func (lx *Lexer) add(t TokenType, v string, extra map[string]any) {
	end := lx.col + utf8.RuneCountInString(v) - 1
	tok := &Token{Type: t, Value: v, Map: extra, Debug: &debug.Debug{Line: lx.line, Column: lx.col, ColumnEnd: end, File: lx.file}}
	lx.tokens = append(lx.tokens, tok)
}

func (lx *Lexer) newErr(base error, err string) error {
	return newErr(base, err, &debug.Debug{Line: lx.line, Column: lx.col, File: lx.file})
}

// ---------- public entry ----------

// Parse performs lexical analysis and returns collected tokens.
func (lx *Lexer) Parse() ([]*Token, error) {
	if lx.curr() != 0 && lx.curr() == '#' && lx.peek(1) == '!' {
		if err := lx.lexSingleLineComment(); err != nil {
			return nil, err
		}
	}

	for lx.curr() != 0 {
		switch lx.curr() {
		case '\n':
			lx.add(TokenNewLine, "\n", nil)
			lx.advance()
		case ' ', '\t':
			lx.add(TokenWhiteSpace, string(lx.curr()), nil)
			lx.advance()
		case '/':
			if lx.peek(1) == '/' {
				if err := lx.lexSingleLineComment(); err != nil {
					return nil, err
				}
			} else if lx.peek(1) == '*' {
				if err := lx.lexMultiLineComment(); err != nil {
					return nil, err
				}
			} else if lx.peek(1) == '>' {
				lx.advance() // consume '/'
				lx.advance() // consume '>'
				lx.add(TokenSelfClosingTag, "/>", nil)
			} else {
				lx.add(TokenSlash, "/", nil)
				lx.advance()
			}
		case '"', '\'', '`':
			if err := lx.lexString(); err != nil {
				return nil, err
			}
		case '?':
			lx.add(TokenQuestion, "?", nil)
			lx.advance()
		case ';', ':', ',', '.', '(', ')', '{', '}', '[', ']':
			ch := lx.curr()
			lx.add(lx.getCharIdent(ch), string(ch), nil)
			lx.advance()
		case '+':
			if lx.peek(1) == '+' {
				lx.add(TokenIncrement, "++", nil)
				lx.advance()
				lx.advance()
			} else {
				lx.add(TokenPlus, "+", nil)
				lx.advance()
			}
		case '-':
			switch lx.peek(1) {
			case '-':
				lx.add(TokenDecrement, "--", nil)
				lx.advance()
				lx.advance()
			case '>':
				lx.add(TokenFnReturnArrow, "->", nil)
				lx.advance()
				lx.advance()
			default:
				lx.add(TokenMinus, "-", nil)
				lx.advance()
			}
		case '*':
			if lx.peek(1) == '*' {
				lx.add(TokenPower, "**", nil)
				lx.advance()
				lx.advance()
			} else {
				lx.add(TokenAsterisk, "*", nil)
				lx.advance()
			}
		case '=':
			switch lx.peek(1) {
			case '=':
				lx.add(TokenEqual, "==", nil)
				lx.advance()
				lx.advance()
			case '>':
				lx.add(TokenArrow, "=>", nil)
				lx.advance()
				lx.advance()
			default:
				lx.add(TokenAssign, "=", nil)
				lx.advance()
			}
		case '!':
			if lx.peek(1) == '=' {
				lx.add(TokenNotEqual, "!=", nil)
				lx.advance()
				lx.advance()
			} else {
				lx.add(TokenNot, "!", nil)
				lx.advance()
			}
		case '&':
			if lx.peek(1) == '&' {
				lx.add(TokenAnd, "&&", nil)
				lx.advance()
				lx.advance()
			} else {
				lx.add(TokenAnd, "&", nil)
				lx.advance()
			}
		case '|':
			if lx.peek(1) == '|' {
				lx.add(TokenOr, "||", nil)
				lx.advance()
				lx.advance()
			} else {
				lx.add(TokenPipe, "|", nil)
				lx.advance()
			}
		case '<':
			if unicode.IsLetter(lx.peek(1)) { // valódi HTML‑kezdés
				if err := lx.lexHtmlBlock(); err != nil {
					return nil, err
				}
			} else {
				switch lx.peek(1) {
				case '=':
					lx.add(TokenLessEqual, "<=", nil)
					lx.advance()
					lx.advance()
				case '/':
					lx.add(TokenClosingStartTag, "</", nil)
					lx.advance()
					lx.advance()
				default:
					lx.add(TokenLessThan, "<", nil)
					lx.advance()
				}
			}
			break
		case '>':
			if lx.peek(1) == '=' {
				lx.add(TokenGreaterEqual, ">=", nil)
				lx.advance()
				lx.advance()
			} else {
				lx.add(TokenGreaterThan, ">", nil)
				lx.advance()
			}
		case '@':
			if lx.peek(1) == '{' {
				lx.add(TokenUnescapedBrace, "@{", nil)
				lx.advance()
				lx.advance()
			} else {
				lx.add(TokenUnknown, "@", nil)
				lx.advance()
			}
		default:
			if unicode.IsLetter(lx.curr()) || lx.curr() == '_' {
				lx.lexIdentifierOrKeyword()
			} else if unicode.IsDigit(lx.curr()) || (lx.curr() == '.' && unicode.IsDigit(lx.peek(1))) {
				if err := lx.lexNumber(); err != nil {
					return nil, err
				}
			} else {
				lx.add(TokenUnknown, string(lx.curr()), nil)
				lx.advance()
			}
		}
	}
	return lx.tokens, nil
}

// ---------- specialised lexers ----------

func (lx *Lexer) lexSingleLineComment() error {
	startCol := lx.col
	lx.advance()
	lx.advance() // skip "//"
	var sb strings.Builder
	sb.WriteString("//")
	for lx.curr() != 0 && lx.curr() != '\n' {
		sb.WriteRune(lx.curr())
		lx.advance()
	}
	lx.add(TokenSingleLineComment, sb.String(), nil)
	// newline token (kept for debug parity with old impl)
	if lx.curr() == '\n' {
		lx.add(TokenNewLine, "\n", nil)
	}
	_ = startCol // col already correct due to advance
	return nil
}

func (lx *Lexer) lexMultiLineComment() error {
	lx.advance()
	lx.advance() // skip "/*"
	var sb strings.Builder
	sb.WriteString("/*")
	for lx.curr() != 0 {
		if lx.curr() == '*' && lx.peek(1) == '/' {
			sb.WriteString("*/")
			lx.advance()
			lx.advance()
			lx.add(TokenMultiLineComment, sb.String(), nil)
			return nil
		}
		sb.WriteRune(lx.curr())
		lx.advance()
	}
	return newErr(ErrSyntaxError, "unterminated multi‑line comment", &debug.Debug{Line: lx.line, Column: lx.col, File: lx.file})
}

func (lx *Lexer) lexString() error {
	quote := lx.curr()
	lx.advance() // skip opening quote
	startLine, startCol := lx.line, lx.col
	var sb strings.Builder
	for lx.curr() != 0 {
		if lx.curr() == '\\' { // escape
			sb.WriteRune(lx.curr())
			lx.advance()
			if lx.curr() == 0 {
				return newErr(ErrSyntaxError, "incomplete escape sequence", &debug.Debug{Line: lx.line, Column: lx.col, File: lx.file})
			}
			sb.WriteRune(lx.curr())
			lx.advance()
			continue
		}
		if lx.curr() == quote {
			lx.advance() // consume closing quote
			strVal, err := escapeString(sb.String(), quote)
			if err != nil {
				return newErr(ErrSyntaxError, err.Error(), &debug.Debug{Line: startLine, Column: startCol, File: lx.file})
			}
			lx.add(TokenString, strVal, map[string]any{"quote": string(quote)})
			return nil
		}
		sb.WriteRune(lx.curr())
		lx.advance()
	}
	return newErr(ErrSyntaxError, "missing closing quote", &debug.Debug{Line: startLine, Column: startCol, File: lx.file})
}

func (lx *Lexer) lexIdentifierOrKeyword() {
	startPos := lx.pos
	for unicode.IsLetter(lx.curr()) || unicode.IsDigit(lx.curr()) || lx.curr() == '_' {
		lx.advance()
	}
	value := string(lx.input[startPos:lx.pos])
	lx.add(lx.getIdentType(value), value, nil)
}

func (lx *Lexer) lexNumber() error {
	startPos := lx.pos
	isFloat := false

	if lx.curr() == '0' && slices.Contains([]rune{'b', 'B', 'o', 'O', 'x', 'X'}, lx.peek(1)) {
		return lx.lexPrefixedNumber()
	}

	for unicode.IsDigit(lx.curr()) || lx.curr() == '.' {
		if lx.curr() == '.' {
			if isFloat {
				return newErr(ErrSyntaxError, "invalid number format", &debug.Debug{Line: lx.line, Column: lx.col, File: lx.file})
			}
			isFloat = true
		}
		lx.advance()
	}
	value := string(lx.input[startPos:lx.pos])
	lx.add(TokenNumber, value, map[string]any{"isFloat": isFloat, "base": 10})
	return nil
}

func (lx *Lexer) lexPrefixedNumber() error {
	startPos := lx.pos

	lx.advance()
	lx.advance()

	prefix := lx.curr()
	lx.advance()

	var base int
	switch prefix {
	case 'b', 'B':
		base = 2
	case 'o', 'O':
		base = 8
	case 'x', 'X':
		base = 16
	}

	for unicode.IsDigit(lx.curr()) {
		lx.advance()
	}

	value := string(lx.input[startPos+2 : lx.pos])
	lx.add(TokenNumber, value, map[string]any{"isFloat": false, "base": base})
	return nil
}

// ---------- helpers ----------

func (lx *Lexer) getCharIdent(ch rune) TokenType {
	chStr := string(ch)
	for _, token := range Operators() {
		if string(token) == chStr {
			return token
		}
	}
	return TokenUnknown
}

func (lx *Lexer) getIdentType(s string) TokenType {
	switch s {
	case "true", "false":
		return TokenBool
	default:
		for _, t := range Keywords() {
			if string(t) == s {
				return t
			}
		}
		return TokenIdentifier
	}
}
