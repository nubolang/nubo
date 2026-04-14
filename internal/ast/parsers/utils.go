package parsers

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strconv"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/lexer"
)

var white = []lexer.TokenType{lexer.TokenWhiteSpace, lexer.TokenSingleLineComment, lexer.TokenMultiLineComment}

func isWhite(token *lexer.Token) bool {
	return slices.Contains(white, token.Type)
}

func inxPPIf(tokens []*lexer.Token, inx *int) error {
	if *inx >= len(tokens) {
		return exception.Create("unexpected end of input").WithDebug(tokens[*inx-1].Debug).WithLevel(exception.LevelSemantic)
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

		start := max(0, *inx-5)
		end := min(len(tokens), *inx+1)

		var shortContext string
		b, err := json.Marshal(tokens[start:end]) // ignore error if safe
		if err != nil {
			shortContext = fmt.Sprintf("%v", tokens[start:end])
		} else {
			shortContext = string(b)
		}
		return fmt.Sprintf("(nubo internal ast package misfunction, go function causing the error: %s)\nexact bug produced here: %s:%d\nBug context: %s", fn.Name(), file, line, shortContext)
	}

	if *inx >= len(tokens) {
		if os.Getenv("NUBO_DEV") == strconv.FormatBool(true) {
			msg := fmt.Sprintf("unexpected end of input %s", reportCaller())
			return debug.NewError(ErrSyntaxError, msg, tokens[*inx-1].Debug)
		}

		msg := fmt.Sprintf("language specific error (nubo internal ast package misfunction [use flag --dev for more details]) (RECOMMENDED: use other syntax as this is not supported yet)")
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

func inxNlPP(tokens []*lexer.Token, inx *int) error {
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

	for *inx < len(tokens) && (slices.Contains(white, tokens[*inx].Type) || tokens[*inx].Type == lexer.TokenNewLine) {
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

func inxPPeak(tokens []*lexer.Token, inx *int) (*lexer.Token, error) {
	i := *inx

	if i >= len(tokens) {
		msg := fmt.Sprintf("unexpected end of input")
		return nil, debug.NewError(ErrSyntaxError, msg, tokens[*inx-1].Debug)
	}

	i++

	for i < len(tokens) && slices.Contains(white, tokens[i].Type) {
		i++
	}

	if i >= len(tokens) {
		msg := fmt.Sprintf("unexpected end of input")
		return nil, debug.NewError(ErrSyntaxError, msg, tokens[*inx-1].Debug)
	}

	return tokens[i], nil
}
