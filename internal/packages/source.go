package packages

import (
	"io/fs"
	"strings"

	"github.com/nubolang/nubo/internal/packages/hex"
	"github.com/nubolang/nubo/internal/packages/terminal"
)

type SourceDefinition struct {
	Source string
	Files  fs.FS
	Funcs  map[string]any
}

func SourcePackage(name string) (SourceDefinition, bool) {
	name, ok := strings.CutPrefix(name, "@std/")
	if !ok {
		return SourceDefinition{}, false
	}

	switch name {
	case "terminal":
		return SourceDefinition{Source: terminal.Source, Funcs: terminal.Functions()}, true
	case "hex":
		return SourceDefinition{Source: hex.Source, Funcs: hex.Functions()}, true
	}

	return SourceDefinition{}, false
}
