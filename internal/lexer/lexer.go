package lexer

import (
	"io"
	"strings"

	"github.com/nubogo/nubo/internal/debug"
)

type Lexer struct {
	file string
}

// New creates a new lexer
func New(file string) *Lexer {
	return &Lexer{
		file: file,
	}
}

// Parse reads the content of the reader and returns the tokens
func (lx *Lexer) Parse(r io.Reader) ([]*Token, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, newErr(ErrReadFailed, err.Error())
	}

	return lx.parse(string(b))
}

// parse reads the content of the string and returns the tokens
func (lx *Lexer) parse(s string) ([]*Token, error) {
	// Fix Carriage Return error on Windows PCs
	s = strings.ReplaceAll(s, "\r", "")

	var (
		runes  = []rune(s)
		pos    int
		line   int = 1
		col    int = 1
		parsed []*Token
	)

	for pos < len(runes) {
		switch runes[pos] {
		// Handle new lines
		case '\n':
			// Add the new line token for debugging purposes
			parsed = append(parsed, &Token{
				Type:  TokenNewLine,
				Value: "\n",
				Debug: &debug.Debug{
					Line:   line,
					Column: col,
					File:   lx.file,
				},
			})
			line++
			col = 1
		// Handle single line comments
		case '/':
			if pos+1 < len(s) && runes[pos+1] == '/' {
				// Skip the entire comment line
				pos += 2

				var sb strings.Builder
				sb.WriteString("//")

				for pos < len(s) && runes[pos] != '\n' {
					sb.WriteByte(s[pos])
					pos++
				}

				// Add the single line comment token for debugging purposes
				sb.WriteByte('\n')
				parsed = append(parsed, &Token{
					Type:  TokenSingleLineComment,
					Value: sb.String(),
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				sb.Reset()
				// New line reached
				line++
				col = 1
			} else if pos+1 < len(s) && runes[pos+1] == '*' {
				pos += 2

				var sb strings.Builder
				sb.WriteString("/*")

				for pos < len(s) {
					if runes[pos] == '*' && pos+1 < len(s) && runes[pos+1] == '/' {
						pos += 2

						// Add the multi line comment token for debugging purposes
						sb.WriteString("*/")
						parsed = append(parsed, &Token{
							Type:  TokenMultiLineComment,
							Value: sb.String(),
							Debug: &debug.Debug{
								Line:   line,
								Column: col,
								File:   lx.file,
							},
						})
						sb.Reset()
						break
					} else if runes[pos] == '\n' {
						sb.WriteByte('\n')
						line++
						col = 1
						pos++
					} else {
						sb.WriteByte(s[pos])
						pos++
					}
				}
			} else if pos+1 < len(s) && runes[pos+1] == '>' {
				pos++
				parsed = append(parsed, &Token{
					Type:  TokenSelfClosingTag,
					Value: "/>",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				col++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenSlash,
					Value: "/",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				col++
			}
		// Handle strings
		case '"', '\'', '`':
			// Skip the opening quote
			quote := runes[pos] // Store the opening quote character
			pos++

			var value string
			start := pos // Start of the string content

			for pos < len(s) {
				if runes[pos] == '\\' {
					// Escape sequence detected
					if pos+1 < len(s) {
						pos += 2
					} else {
						// Syntax error: escape character at the end
						return nil, newErr(ErrSyntaxError, "incomplete escape sequence", &debug.Debug{
							Line:   line,
							Column: col,
							File:   lx.file,
						})
					}
				} else if runes[pos] == quote {
					// Closing quote found
					value = string(runes[start:pos]) // Extract the string content
					pos++                            // Skip the closing quote
					break
				} else {
					pos++
				}
			}

			// If we exited the loop without finding a closing quote
			if value == "" && pos >= len(s) {
				return nil, newErr(ErrSyntaxError, "missing closing quote for string starting", &debug.Debug{
					Line:   line,
					Column: col,
					File:   lx.file,
				})
			}

			typ := TokenString

			// Add the string token to the parsed tokens
			parsed = append(parsed, &Token{
				Type:  typ,
				Value: value,
				Debug: &debug.Debug{
					Line:   line,
					Column: col,
					File:   lx.file,
				},
				Map: map[string]any{
					"quote": string(quote),
				},
			})
			pos--
		case ' ', '\t':
			// Whitespace token for debugging purposes
			parsed = append(parsed, &Token{
				Type:  TokenWhiteSpace,
				Value: string(s[pos]),
				Debug: &debug.Debug{
					Line:   line,
					Column: col,
					File:   lx.file,
				},
			})
			col++
		case '?':
			parsed = append(parsed, &Token{
				Type:  TokenQuestion,
				Value: "?",
				Debug: &debug.Debug{
					Line:   line,
					Column: col,
					File:   lx.file,
				},
			})
			col++
		case ';', ':', ',', '.', '(', ')', '{', '}', '[', ']':
			ch := runes[pos]
			parsed = append(parsed, &Token{
				Type:  lx.getCharIdent(ch),
				Value: string(ch),
				Debug: &debug.Debug{
					Line:   line,
					Column: col,
					File:   lx.file,
				},
			})
			col++
		case '=':
			if pos+1 < len(s) && runes[pos+1] == '=' {
				parsed = append(parsed, &Token{
					Type:  TokenEqual,
					Value: "==",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else if pos+1 < len(s) && runes[pos+1] == '>' {
				parsed = append(parsed, &Token{
					Type:  TokenArrow,
					Value: "=>",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenAssign,
					Value: "=",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
			}
		case '!':
			if pos+1 < len(s) && runes[pos+1] == '=' {
				parsed = append(parsed, &Token{
					Type:  TokenNotEqual,
					Value: "!=",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				col += 2
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenNot,
					Value: "!",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
			}
		case '&':
			if pos+1 < len(s) && runes[pos+1] == '&' {
				parsed = append(parsed, &Token{
					Type:  TokenAnd,
					Value: "&&",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				col += 2
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenAnd,
					Value: "&",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				col++
			}
		case '|':
			if pos+1 < len(s) && runes[pos+1] == '|' {
				parsed = append(parsed, &Token{
					Type:  TokenOr,
					Value: "||",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				col += 2
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenOr,
					Value: "|",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				col++
			}
		case '+':
			if pos+1 < len(s) && runes[pos+1] == '+' {
				parsed = append(parsed, &Token{
					Type:  TokenIncrement,
					Value: "++",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenPlus,
					Value: "+",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
			}
			col++
		case '-':
			if pos+1 < len(s) && runes[pos+1] == '-' {
				parsed = append(parsed, &Token{
					Type:  TokenDecrement,
					Value: "--",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else if pos+1 < len(s) && runes[pos+1] == '>' {
				parsed = append(parsed, &Token{
					Type:  TokenFnReturnArrow,
					Value: "->",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenMinus,
					Value: "-",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
			}
			col++
		case '*':
			if pos+1 < len(s) && runes[pos+1] == '*' {
				parsed = append(parsed, &Token{
					Type:  TokenPower,
					Value: "**",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenAsterisk,
					Value: "*",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
			}
		case '<':
			if pos+1 < len(s) && runes[pos+1] == '=' {
				parsed = append(parsed, &Token{
					Type:  TokenLessEqual,
					Value: "<=",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else if pos+1 < len(s) && runes[pos+1] == '/' {
				parsed = append(parsed, &Token{
					Type:  TokenClosingStartTag,
					Value: "</",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenLessThan,
					Value: "<",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
			}
		case '>':
			if pos+1 < len(s) && runes[pos+1] == '=' {
				parsed = append(parsed, &Token{
					Type:  TokenGreaterEqual,
					Value: ">=",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos++
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenGreaterThan,
					Value: ">",
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
			}
		default:
			start := pos
			if isLetter(runes[pos]) {
				// Identifier parsing
				for pos < len(runes) && (isLetter(runes[pos]) || isDigit(runes[pos])) {
					pos++
				}
				value := string(runes[start:pos])
				// Appending the identifier
				parsed = append(parsed, &Token{
					Type:  lx.getIdentType(value),
					Value: value,
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos--
			} else if isDigit(runes[pos]) || (s[pos] == '.' && pos+1 < len(s) && isDigit(runes[pos+1])) {
				// Number parsing (integer or float)
				isFloat := false
				for pos < len(s) && (isDigit(runes[pos]) || runes[pos] == '.') {
					if runes[pos] == '.' {
						if isFloat {
							// Second dot found, invalid number
							return nil, newErr(ErrSyntaxError, "invalid number format", &debug.Debug{
								Line:   line,
								Column: col,
								File:   lx.file,
							})
						}
						isFloat = true
					}
					pos++
				}
				value := string(runes[start:pos])
				// Appending the number
				parsed = append(parsed, &Token{
					Type:  TokenNumber,
					Value: value,
					Map: map[string]any{
						"isFloat": isFloat,
					},
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				pos--
			} else {
				parsed = append(parsed, &Token{
					Type:  TokenUnknown,
					Value: string(s[pos]),
					Debug: &debug.Debug{
						Line:   line,
						Column: col,
						File:   lx.file,
					},
				})
				col++
			}
		}
		pos++
	}

	return parsed, nil
}

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
