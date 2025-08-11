package ast

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/ast/parsers"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/lexer"
)

func (a *Ast) ParseHTMLAttrValue(dg *debug.Debug, s string) (*astnode.Node, error) {
	var dS string
	if dg != nil {
		dS = fmt.Sprintf("%s:%d:%d->", dg.File, dg.Line, dg.Column)
	}

	lx, err := lexer.New(strings.NewReader(s+"\n"), dS+"<html>")
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
