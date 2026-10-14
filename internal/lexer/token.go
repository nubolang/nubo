package lexer

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/nubolang/nubo/internal/debug"
)

type Token struct {
	// TokenType is the type of the token
	Type TokenType `yaml:"type"`
	// Value is the value of the token
	Value string `yaml:"value"`

	// Map is a map of key value pairs
	Map map[string]any `yaml:"map,omitempty"`
	// Debug is a debug object
	Debug *debug.Debug `yaml:"-"`
}

func newToken(t TokenType, value, source string, extra map[string]any, file string, line, col int) *Token {
	firstLine, _, _ := strings.Cut(source, "\n")
	end := col + utf8.RuneCountInString(firstLine) - 1
	if end < col {
		end = col
	}
	if source != value {
		if extra == nil {
			extra = make(map[string]any)
		}
		extra["source"] = source
	}
	// Debug ranges describe the opening line; there is no ending-line field.
	return &Token{Type: t, Value: value, Map: extra, Debug: &debug.Debug{Line: line, Column: col, ColumnEnd: end, File: file}}
}

func (t *Token) Text() string {
	switch t.Type {
	default:
		return t.Value
	case TokenString:
		return strconv.Quote(t.Value)
	}
}
