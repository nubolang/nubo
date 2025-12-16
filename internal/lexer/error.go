package lexer

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

func newErr(base error, err string, d ...*debug.Debug) error {
	return debug.NewError(base, err, d...)
}

var (
	ErrReadFailed  = fmt.Errorf("failed to read file content")
	ErrSyntaxError = fmt.Errorf("syntax error")
	ErrHtmlBrace   = fmt.Errorf("html brace content error")
)
