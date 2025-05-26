package astnode

import (
	"github.com/bndrmrtn/tea/internal/debug"
	"github.com/bndrmrtn/tea/internal/lexer"
)

type Node struct {
	Type lexer.TokenType `yaml:"type"`
	Kind string          `yaml:"kind,omitempty"`

	Content   string `yaml:"content"`
	Value     any    `yaml:"value,omitempty"`
	ValueType *Node  `yaml:"value_type,omitempty"`

	IsReference bool `yaml:"is_reference,omitempty"`

	Children []*Node `yaml:"children,omitempty"`
	Args     []*Node `yaml:"args,omitempty"`
	Body     []*Node `yaml:"body,omitempty"`

	Attrs map[string]any `yaml:"attrs,omitempty"`
	Flags []string       `yaml:"flags,omitempty"`

	Debug *debug.Debug `yaml:"-"`
}
