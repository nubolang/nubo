package debug

import (
	"errors"
	"fmt"

	"github.com/fatih/color"
)

type DebugErr struct {
	err   error
	msg   string
	debug *Debug
}

func (de *DebugErr) Error() string {
	redBold := color.New(color.FgRed, color.Bold).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgHiBlue).SprintFunc()

	var location string
	if de.debug != nil {
		location = fmt.Sprintf(": %s:%s:%s", blue(de.debug.File), blue(de.debug.Line), blue(de.debug.Column))
	}

	return fmt.Sprintf("%s %s%s", redBold(de.err.Error()+":"), red(de.msg), location)
}

func (de *DebugErr) Unwrap() error {
	return de.err
}

func NewError(base error, err string, debug ...*Debug) error {
	var dg *DebugErr
	if errors.As(base, &dg) {
		return dg
	}

	var d *Debug
	if len(debug) > 0 {
		d = debug[0]
	}

	return &DebugErr{
		err:   base,
		msg:   err,
		debug: d,
	}
}
