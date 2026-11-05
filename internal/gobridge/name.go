package gobridge

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/lexer"
	"github.com/nubolang/nubo/language"
)

func WrapNamed(name string, fn any) (*language.Function, error) {
	lx, err := lexer.New(strings.NewReader(name), "<GoFunc>")
	if err != nil {
		return nil, err
	}

	tokens, err := lx.Parse()
	if err != nil || len(tokens) != 1 || tokens[0].Type != lexer.TokenIdentifier || tokens[0].Value != name {
		return nil, fmt.Errorf("invalid Go function name %q", name)
	}

	wrapped, err := Wrap(fn)
	if err != nil {
		return nil, fmt.Errorf("cannot register Go function %q: %w", name, err)
	}

	return wrapped, nil
}
