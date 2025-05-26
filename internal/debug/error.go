package debug

import (
	"errors"
	"fmt"
)

type DebugErr struct {
	err   error
	debug *Debug
}

func (de *DebugErr) Error() string {
	return fmt.Sprintf("%v: %v", de.err, de.debug)
}

func NewError(base error, err string, debug ...*Debug) error {
	var d *Debug
	if len(debug) > 0 {
		d = debug[0]
	}

	return &DebugErr{
		err:   fmt.Errorf("%w: %w", base, errors.New(err)),
		debug: d,
	}
}
