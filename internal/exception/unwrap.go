package exception

import (
	"errors"

	"github.com/nubolang/nubo/internal/debug"
)

type ExceptionData struct {
	Base    string
	Cause   error
	Message string

	Level      Level
	Debug      *debug.Debug
	StackTrace []*debug.Debug
}

func (e *Expection) Details() *ExceptionData {
	return &ExceptionData{
		Cause:      e.base,
		Message:    e.msg,
		Level:      e.level,
		Debug:      e.debug,
		StackTrace: e.traceFrames(),
	}
}

func Unwrap(err error) (*ExceptionData, bool) {
	var excp *Expection
	if errors.As(err, &excp) {
		data := excp.Details()
		if data.Cause != nil {
			data.Base = data.Cause.Error()
		}

		return data, true
	}

	return nil, false
}

func Is(err error) bool {
	var excp *Expection
	return errors.As(err, &excp)
}
