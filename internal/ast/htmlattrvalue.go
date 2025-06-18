package ast

import (
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/ast/parsers"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/lexer"
)

func (a *Ast) ParseHTMLAttrValue(s string) (*astnode.Node, error) {
	lx, err := lexer.New(strings.NewReader(s+"\n"), "<html>")
	if err != nil {
		return nil, err
	}
	tokens, err := lx.Parse()
	if err != nil {
		return nil, err
	}

	inx := 0
	return parsers.ValueParser(a.ctx, a, tokens, &inx)
}

func (a *Ast) ParseHTML(s string, dg *debug.Debug) ([]*lexer.Token, error) {
	lx := lexer.NewHtmlLexer(s, dg)
	return lx.Lex()
}
