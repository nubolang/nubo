package parsers

import (
	"fmt"

	"github.com/bndrmrtn/tea/internal/ast/astnode"
	"github.com/bndrmrtn/tea/internal/debug"
	"github.com/bndrmrtn/tea/internal/lexer"
)

func inxPPWs(tokens []*lexer.Token, inx *int) error {
	*inx++

	if *inx >= len(tokens) {
		return debug.NewError(ErrSyntaxError, "unexpected end of input", tokens[*inx-1].Debug)
	}

	return nil
}

func inxPP(tokens []*lexer.Token, inx *int) error {
	*inx++

	for *inx < len(tokens) && tokens[*inx].Type == lexer.TokenWhiteSpace {
		*inx++
	}

	if *inx >= len(tokens) {
		return debug.NewError(ErrSyntaxError, "unexpected end of input", tokens[*inx-1].Debug)
	}

	return nil
}

func skipSemi(tokens []*lexer.Token, inx *int, node *astnode.Node) *astnode.Node {
	if err := inxPP(tokens, inx); err != nil {
		return node
	}

	token := tokens[*inx]
	if token.Type == lexer.TokenSemicolon {
		*inx++
	}

	return node
}

func nl(tokens []*lexer.Token, inx *int) error {
	*inx++

	for *inx < len(tokens) && tokens[*inx].Type == lexer.TokenNewLine {
		return nil
	}

	return newErr(ErrUnexpectedToken, fmt.Sprintf("expected newline, got %s", tokens[*inx-1].Type), tokens[*inx-1].Debug)
}
