//go:build !windows

package terminal

func prepareOutput() (func() error, error) {
	return func() error {
		return nil
	}, nil
}
