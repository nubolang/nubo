package parsers

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

func newErr(base error, err string, d ...*debug.Debug) error {
	return debug.NewError(base, err, d...)
}

var (
	ErrSyntaxError     = fmt.Errorf("Syntax error")
	ErrUnexpectedToken = fmt.Errorf("Unexpected token")
	ErrValueError      = fmt.Errorf("Value error")
	ErrUnexpectedEOF   = fmt.Errorf("Unexpected end of tokens")
	ErrInvalidEventID  = fmt.Errorf("Invalid event ID")
)
