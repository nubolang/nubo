package interpreter

import (
	"fmt"

	"github.com/nubogo/nubo/internal/debug"
)

func newErr(base error, err string, d ...*debug.Debug) error {
	if len(d) > 0 {
		return debug.NewError(base, err, d[0])
	}

	return fmt.Errorf("%w: %v", base, err)
}

var (
	ErrUnknownNode       = fmt.Errorf("Unknown node")
	ErrImmutableVariable = fmt.Errorf("Variable is immutable")
	ErrExpression        = fmt.Errorf("Expression error")
)
