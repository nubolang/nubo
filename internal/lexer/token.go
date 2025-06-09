package lexer

import (
	"strconv"

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

func (t *Token) Text() string {
	switch t.Type {
	default:
		return t.Value
	case TokenString:
		return strconv.Quote(t.Value)
	}
}
