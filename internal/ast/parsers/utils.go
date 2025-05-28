package parsers

import (
	"fmt"
	"slices"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/debug"
	"github.com/nubogo/nubo/internal/lexer"
)

var white = []lexer.TokenType{lexer.TokenWhiteSpace, lexer.TokenSingleLineComment, lexer.TokenMultiLineComment}

func inxPPIf(tokens []*lexer.Token, inx *int) error {
	if *inx >= len(tokens) {
		return debug.NewError(ErrSyntaxError, "unexpected end of input", tokens[*inx-1].Debug)
	}

	if slices.Contains(white, tokens[*inx].Type) {
		return inxPP(tokens, inx)
	}

	return nil
}

func inxPP(tokens []*lexer.Token, inx *int) error {
	*inx++

	for *inx < len(tokens) && slices.Contains(white, tokens[*inx].Type) {
		*inx++
	}

	if *inx >= len(tokens) {
		return debug.NewError(ErrSyntaxError, "unexpected end of input", tokens[*inx-1].Debug)
	}

	return nil
}

func skipSemi(tokens []*lexer.Token, inx *int, node *astnode.Node) *astnode.Node {
	if *inx >= len(tokens) {
		return node
	}

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

func safeIncr(tokens []*lexer.Token, inx *int) {
	if *inx < len(tokens) {
		*inx++
	}
}
