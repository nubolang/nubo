package debug

import (
	"errors"
	"fmt"

	"github.com/fatih/color"
)

type DebugErr struct {
	err   error
	debug *Debug
}

func (de *DebugErr) Error() string {
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	blue := color.New(color.FgHiBlue).SprintFunc()

	var location string
	if de.debug != nil {
		location = fmt.Sprintf(": %s:%s:%s", blue(de.debug.File), blue(de.debug.Line), blue(de.debug.Column))
	}

	return fmt.Sprintf("%s%s", red(de.err.Error()), location)
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
