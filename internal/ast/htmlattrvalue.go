package ast

import (
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/ast/parsers"
)

func (a *Ast) ParseHTMLAttrValue(s string) (*astnode.Node, error) {
	tokens, err := a.lx.Parse(strings.NewReader(s + "\n"))
	if err != nil {
		return nil, err
	}

	inx := 0
	return parsers.ValueParser(a.ctx, a, tokens, &inx)
}
