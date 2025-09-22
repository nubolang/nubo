package lexer

import (
	"unicode"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/html"
)

// lexHtmlBlock reads one complete HTML element (root + children) and
// returns it as a single TokenHtmlBlock.
func (lx *Lexer) lexHtmlBlock() error {
	start := lx.pos
	startCol := lx.col
	startLine := lx.line
	lx.advance() // consume '<'

	// Parse tag name
	nameStart := lx.pos
	for unicode.IsLetter(lx.curr()) {
		lx.advance()
	}
	tag := string(lx.input[nameStart:lx.pos])

	selfClosing := false

	// Scan tag attributes and check for self-closing
scanAttrs:
	for lx.curr() != 0 {
		switch lx.curr() {
		case '"', '\'':
			quote := lx.curr()
			lx.advance()
			for lx.curr() != 0 && lx.curr() != quote {
				if lx.curr() == '\\' {
					lx.advance()
				}
				lx.advance()
			}
			if lx.curr() != 0 {
				lx.advance()
			}
		case '/':
			if lx.peek(1) == '>' {
				selfClosing = true
				lx.advance()
				lx.advance()
				break scanAttrs
			}
			lx.advance()
		case '>':
			lx.advance()
			break scanAttrs
		default:
			lx.advance()
		}
	}

	if selfClosing || html.VoidTags[tag] {
		lx.add(TokenHtmlBlock, string(lx.input[start:lx.pos]), nil)
		return nil
	}

	depth := 1
	for lx.curr() != 0 && depth > 0 {
		switch lx.curr() {
		case '{':
			lx.skipBraces()
		case '<':
			if lx.peek(1) == '/' {
				// Found closing tag
				lx.advance()
				lx.advance()
				nameStart := lx.pos
				for unicode.IsLetter(lx.curr()) {
					lx.advance()
				}
				name := string(lx.input[nameStart:lx.pos])
				if name == tag {
					depth--
				}
				// Skip until '>'
				for lx.curr() != 0 && lx.curr() != '>' {
					lx.advance()
				}
				if lx.curr() == '>' {
					lx.advance()
				}
			} else if unicode.IsLetter(lx.peek(1)) {
				// Found nested same tag
				lx.advance()
				nestedStart := lx.pos
				for unicode.IsLetter(lx.curr()) {
					lx.advance()
				}
				nestedName := string(lx.input[nestedStart:lx.pos])
				if nestedName == tag {
					depth++
				}
				// Skip attributes or to tag end
				for lx.curr() != 0 && lx.curr() != '>' {
					lx.advance()
				}
				if lx.curr() == '>' {
					lx.advance()
				}
			} else {
				lx.advance()
			}
		default:
			lx.advance()
		}
	}

	if depth != 0 {
		return newErr(ErrSyntaxError, "Unterminated html block", &debug.Debug{
			File:   lx.file,
			Column: startCol,
			Line:   startLine,
		})
	}

	lx.add(TokenHtmlBlock, string(lx.input[start:lx.pos]), nil)
	return nil
}

func (lx *Lexer) skipBraces() {
	if lx.curr() == '!' && lx.peek(1) == '{' {
		lx.advance() // '!'
	}
	if lx.curr() == '{' {
		depth := 1
		lx.advance() // '{'
		for lx.curr() != 0 && depth > 0 {
			switch lx.curr() {
			case '{':
				depth++
			case '}':
				depth--
			case '\\': // escape
				lx.advance() // '\'
			}
			lx.advance()
		}
	}
}
