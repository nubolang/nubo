package formatter

import (
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
)

type Formatter struct {
	nodes []*astnode.Node
	sb    strings.Builder
}

func New(nodes []*astnode.Node) *Formatter {
	return &Formatter{
		nodes: nodes,
	}
}

func (f *Formatter) Format() string {
	for _, node := range f.nodes {
		f.formatNode(node)
	}
	return f.sb.String()
}
