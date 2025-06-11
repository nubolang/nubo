package n

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

func Err(m string, d *debug.Debug) error {
	return debug.NewError(fmt.Errorf("Function error"), m, d)
}
