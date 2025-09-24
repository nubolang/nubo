package n

import (
	"errors"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
)

func Err(m string, d *debug.Debug) error {
	return exception.Create(m).WithBase(errors.New("function error")).WithDebug(d).WithLevel(exception.LevelRuntime)
}
