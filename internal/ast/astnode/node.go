package astnode

import (
	"slices"

	"github.com/nubogo/nubo/internal/debug"
)

type Node struct {
	Type NodeType `yaml:"type"`
	Kind string   `yaml:"kind,omitempty"`

	Content   string `yaml:"content,omitempty"`
	Value     any    `yaml:"value,omitempty"`
	ValueType *Node  `yaml:"value_type,omitempty"`

	IsReference bool `yaml:"is_reference,omitempty"`

	Children []*Node `yaml:"children,omitempty"`
	Args     []*Node `yaml:"args,omitempty"`
	Body     []*Node `yaml:"body,omitempty"`

	Attrs map[string]any `yaml:"attrs,omitempty"`
	Flags AppendFlags    `yaml:"flags,omitempty"`

	Debug *debug.Debug `yaml:"-"`
}

type AppendFlags []string

func (a *AppendFlags) Append(flags ...string) {
	*a = append(*a, flags...)
}

func (a *AppendFlags) Contains(flag string) bool {
	return slices.Contains(*a, flag)
}
