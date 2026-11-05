package nubo

import "github.com/nubolang/nubo/internal/gobridge"

// GoFunction describes a host function to register with an app.
type GoFunction struct {
	value any
}

// GoFunc defers signature validation until AddFunc.
func GoFunc(fn any) GoFunction {
	return GoFunction{value: fn}
}

// AddFunc exposes a Go function to this app's scripts.
func (c *Ctx) AddFunc(name string, fn GoFunction) error {
	wrapped, err := gobridge.WrapNamed(name, fn.value)
	if err != nil {
		return err
	}

	return c.r.AddFunction(name, wrapped)
}
