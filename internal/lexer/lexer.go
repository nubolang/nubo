package lexer

import (
	"io"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

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

func (lx *Lexer) SetDebugLineColData(line int, col int) {
	if line > -1 {
		lx.line = line
	}
	if col > -1 {
		lx.col = col
	}
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
	lx.tokens = append(lx.tokens, newToken(t, v, v, extra, lx.file, lx.line, lx.col))
}

func (lx *Lexer) addAt(t TokenType, v string, extra map[string]any, start, line, col int) {
	source := string(lx.input[start:lx.pos])
	lx.tokens = append(lx.tokens, newToken(t, v, source, extra, lx.file, line, col))
}

func (lx *Lexer) newErr(base error, err string) error {
	return newErr(base, err, &debug.Debug{Line: lx.line, Column: lx.col, ColumnEnd: lx.col, File: lx.file})
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
				lx.add(TokenSelfClosingTag, "/>", nil)
				lx.advance() // consume '/'
				lx.advance() // consume '>'
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
		case '%':
			lx.add(TokenPercent, "%", nil)
			lx.advance()
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
			} else if lx.peek(1) == '{' {
				lx.add(TokenUnescapedBrace, "!{", nil)
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
		case '>':
			if lx.peek(1) == '=' {
				lx.add(TokenGreaterEqual, ">=", nil)
				lx.advance()
				lx.advance()
			} else {
				lx.add(TokenGreaterThan, ">", nil)
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
	start, startLine, startCol := lx.pos, lx.line, lx.col
	lx.advance()
	lx.advance() // skip "//"
	var sb strings.Builder
	sb.WriteString("//")
	for lx.curr() != 0 && lx.curr() != '\n' {
		sb.WriteRune(lx.curr())
		lx.advance()
	}
	lx.addAt(TokenSingleLineComment, sb.String(), nil, start, startLine, startCol)
	// newline token (kept for debug parity with old impl)
	if lx.curr() == '\n' {
		lx.add(TokenNewLine, "\n", nil)
		lx.advance()
	}
	return nil
}

func (lx *Lexer) lexMultiLineComment() error {
	start, startLine, startCol := lx.pos, lx.line, lx.col
	lx.advance()
	lx.advance() // skip "/*"
	var sb strings.Builder
	sb.WriteString("/*")
	for lx.curr() != 0 {
		if lx.curr() == '*' && lx.peek(1) == '/' {
			sb.WriteString("*/")
			lx.advance()
			lx.advance()
			lx.addAt(TokenMultiLineComment, sb.String(), nil, start, startLine, startCol)
			return nil
		}
		sb.WriteRune(lx.curr())
		lx.advance()
	}
	return newErr(ErrSyntaxError, "unterminated multi‑line comment", &debug.Debug{Line: startLine, Column: startCol, ColumnEnd: startCol, File: lx.file})
}

func (lx *Lexer) lexString() error {
	start, startLine, startCol := lx.pos, lx.line, lx.col
	quote := lx.curr()
	lx.advance() // skip opening quote
	var sb strings.Builder
	for lx.curr() != 0 {
		if lx.curr() == '\\' { // escape
			sb.WriteRune(lx.curr())
			lx.advance()
			if lx.curr() == 0 {
				return newErr(ErrSyntaxError, "incomplete escape sequence", &debug.Debug{Line: lx.line, Column: lx.col, ColumnEnd: lx.col, File: lx.file})
			}
			sb.WriteRune(lx.curr())
			lx.advance()
			continue
		}
		if lx.curr() == quote {
			lx.advance() // consume closing quote
			strVal, err := escapeString(sb.String(), quote)
			if err != nil {
				return newErr(ErrSyntaxError, err.Error(), &debug.Debug{Line: startLine, Column: startCol, ColumnEnd: startCol, File: lx.file})
			}
			lx.addAt(TokenString, strVal, map[string]any{"quote": string(quote)}, start, startLine, startCol)
			return nil
		}
		sb.WriteRune(lx.curr())
		lx.advance()
	}
	return newErr(ErrSyntaxError, "missing closing quote", &debug.Debug{Line: startLine, Column: startCol, ColumnEnd: startCol, File: lx.file})
}

func (lx *Lexer) lexIdentifierOrKeyword() {
	startPos, startLine, startCol := lx.pos, lx.line, lx.col
	for unicode.IsLetter(lx.curr()) || unicode.IsDigit(lx.curr()) || lx.curr() == '_' {
		lx.advance()
	}
	value := string(lx.input[startPos:lx.pos])
	lx.addAt(lx.getIdentType(value), value, nil, startPos, startLine, startCol)
}

func (lx *Lexer) lexNumber() error {
	startPos, startLine, startCol := lx.pos, lx.line, lx.col
	isFloat := false

	if lx.curr() == '0' && slices.Contains([]rune{'b', 'B', 'o', 'O', 'x', 'X'}, lx.peek(1)) {
		return lx.lexPrefixedNumber()
	}

	for unicode.IsDigit(lx.curr()) || lx.curr() == '.' || lx.curr() == '_' {
		if lx.curr() == '_' {
			lx.advance()
			continue
		}
		if lx.curr() == '.' {
			if isFloat {
				return newErr(ErrSyntaxError, "invalid number format", &debug.Debug{Line: lx.line, Column: lx.col, ColumnEnd: lx.col, File: lx.file})
			}
			isFloat = true
		}
		lx.advance()
	}
	value := strings.ReplaceAll(string(lx.input[startPos:lx.pos]), "_", "")
	lx.addAt(TokenNumber, value, map[string]any{"isFloat": isFloat, "base": 10}, startPos, startLine, startCol)
	return nil
}

func (lx *Lexer) lexPrefixedNumber() error {
	startPos, startLine, startCol := lx.pos, lx.line, lx.col

	lx.advance()

	prefix := lx.curr()
	lx.advance()

	var base int = 10
	switch prefix {
	case 'b', 'B':
		base = 2
	case 'o', 'O':
		base = 8
	case 'x', 'X':
		base = 16
	}

	for unicode.IsDigit(lx.curr()) || lx.curr() == '_' || (base == 16 && (lx.curr() == 'a' || lx.curr() == 'A' || lx.curr() == 'b' || lx.curr() == 'B' || lx.curr() == 'c' || lx.curr() == 'C' || lx.curr() == 'd' || lx.curr() == 'D' || lx.curr() == 'e' || lx.curr() == 'E' || lx.curr() == 'f' || lx.curr() == 'F')) {
		if lx.curr() == '_' {
			lx.advance()
			continue
		}
		lx.advance()
	}

	value := strings.ReplaceAll(string(lx.input[startPos+2:lx.pos]), "_", "")
	lx.addAt(TokenNumber, value, map[string]any{"isFloat": false, "base": base}, startPos, startLine, startCol)
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
