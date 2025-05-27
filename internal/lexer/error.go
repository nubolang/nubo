package lexer

import (
	"errors"
	"fmt"

	"github.com/nubogo/nubo/internal/debug"
)

func newErr(base error, err string, d ...*debug.Debug) error {
	return fmt.Errorf("%w: %w", base, errors.New(err))
}

var (
	ErrReadFailed  = fmt.Errorf("Failed to read file content")
	ErrSyntaxError = fmt.Errorf("Syntax error")
)
