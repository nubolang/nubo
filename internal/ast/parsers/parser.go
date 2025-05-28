package parsers

import (
	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/lexer"
)

type parser interface {
	Parse(tokens []*lexer.Token) ([]*astnode.Node, error)
}
