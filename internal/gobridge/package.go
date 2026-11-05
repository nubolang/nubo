package gobridge

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/nubolang/nubo/native/n"
)

func WrapPackage(name string, funcs map[string]any) (*n.Package, error) {
	module, ok := strings.CutPrefix(name, "@std/")
	if !ok || module == "." || !fs.ValidPath(module) {
		return nil, fmt.Errorf("invalid Go package name %q: expected @std/name", name)
	}

	for _, char := range module {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' && char != '-' && char != '/' {
			return nil, fmt.Errorf("invalid Go package name %q", name)
		}
	}

	pkg := n.NewPackage(module, nil)
	proto := pkg.GetPrototype()
	for _, member := range slices.Sorted(maps.Keys(funcs)) {
		wrapped, err := WrapNamed(member, funcs[member])
		if err != nil {
			return nil, fmt.Errorf("cannot register Go package %q: %w", name, err)
		}

		if err := proto.SetObject(context.Background(), member, wrapped); err != nil {
			return nil, fmt.Errorf("cannot register Go package %q: %w", name, err)
		}
	}

	return pkg, nil
}
