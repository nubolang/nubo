package parsers

import (
	"fmt"

	"github.com/bndrmrtn/tea/internal/debug"
)

func newErr(base error, err string, d ...*debug.Debug) error {
	if len(d) > 0 {
		return debug.NewError(base, err, d[0])
	}

	return fmt.Errorf("%w: %v", base, err)
}

var (
	ErrSyntaxError     = fmt.Errorf("Syntax error")
	ErrUnexpectedToken = fmt.Errorf("Unexpected token")
)
