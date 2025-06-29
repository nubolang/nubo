package debug

import (
	"errors"
)

func Unwrap(err error) (error, string, *Debug) {
	var dg *DebugErr
	if errors.As(err, &dg) {
		return dg.err, dg.msg, dg.debug
	}
	return err, "", nil
}
