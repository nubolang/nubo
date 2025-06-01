package parsers

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

type parser interface {
	Parse(tokens []*lexer.Token) ([]*astnode.Node, error)
}
