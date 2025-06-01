package parsers

import (
	"fmt"
	"runtime"
	"slices"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/lexer"
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
	reportCaller := func() string {
		pc, file, line, ok := runtime.Caller(2)
		if !ok {
			return "unknown caller"
		}
		fn := runtime.FuncForPC(pc)
		return fmt.Sprintf("called from %s (%s:%d)", fn.Name(), file, line)
	}

	if *inx >= len(tokens) {
		msg := fmt.Sprintf("unexpected end of input [%s]", reportCaller())
		return debug.NewError(ErrSyntaxError, msg, tokens[*inx-1].Debug)
	}

	*inx++

	for *inx < len(tokens) && slices.Contains(white, tokens[*inx].Type) {
		*inx++
	}

	if *inx >= len(tokens) {
		msg := fmt.Sprintf("unexpected end of input [%s]", reportCaller())
		return debug.NewError(ErrSyntaxError, msg, tokens[*inx-1].Debug)
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

func tokensPrint(tokens []*lexer.Token) {
	for _, token := range tokens {
		fmt.Printf("{[%s : %s]} ", token.Type, token.Value)
	}
	fmt.Println()
}
