package exception

import (
	"errors"

	"github.com/nubolang/nubo/internal/debug"
)

type ExceptionData struct {
	Base    string
	Message string

	Level      Level
	Debug      *debug.Debug
	StackTrace []*debug.Debug
}

func Unwrap(err error) (*ExceptionData, bool) {
	var excp *Expection
	if errors.As(err, &excp) {
		var base string
		if excp.base != nil {
			base = excp.base.Error()
		}

		return &ExceptionData{
			Base:       base,
			Message:    excp.msg,
			Level:      excp.level,
			Debug:      excp.debug,
			StackTrace: excp.trace,
		}, true
	}
	return nil, false
}

func Is(err error) bool {
	var excp *Expection
	return errors.As(err, &excp)
}
