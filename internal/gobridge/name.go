package gobridge

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/lexer"
	"github.com/nubolang/nubo/language"
)

func WrapNamed(name string, fn any) (*language.Function, error) {
	return wrapNamed(name, fn, false)
}

// Source packages can also supply a binding that accepts Nubo objects directly.
func wrapNamed(name string, fn any, allowNativeBinding bool) (*language.Function, error) {
	lx, err := lexer.New(strings.NewReader(name), "<GoFunc>")
	if err != nil {
		return nil, err
	}

	tokens, err := lx.Parse()
	if err != nil || len(tokens) != 1 || tokens[0].Type != lexer.TokenIdentifier || tokens[0].Value != name {
		return nil, fmt.Errorf("invalid Go function name %q", name)
	}

	if wrapped, ok := fn.(*language.Function); allowNativeBinding && ok && wrapped != nil && wrapped.Data != nil {
		return wrapped, nil
	}

	wrapped, err := Wrap(fn)
	if err != nil {
		return nil, fmt.Errorf("cannot register Go function %q: %w", name, err)
	}

	return wrapped, nil
}
