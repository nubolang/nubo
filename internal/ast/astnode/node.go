package astnode

import (
	"slices"

	"github.com/nubolang/nubo/internal/debug"
)

type ForValue struct {
	Iterator *Node `yaml:"iterator,omitempty"`
	Value    *Node `yaml:"value"`
}

type Node struct {
	Type NodeType `yaml:"type"`
	Kind string   `yaml:"kind,omitempty"`

	Content       string `yaml:"content,omitempty"`
	Value         any    `yaml:"value,omitempty"`
	ValueType     *Node  `yaml:"value_type,omitempty"`
	FallbackValue *Node  `yaml:"fallback_value,omitempty"`

	IsReference bool `yaml:"is_reference,omitempty"`

	Children    []*Node `yaml:"children,omitempty"`
	Args        []*Node `yaml:"args,omitempty"`
	Body        []*Node `yaml:"body,omitempty"`
	ArrayAccess []*Node `yaml:"array_access,omitempty"`

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
