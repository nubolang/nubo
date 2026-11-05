package nubo

import (
	"io/fs"

	"github.com/nubolang/nubo/internal/gobridge"
)

// Package describes a Go-backed module.
type Package struct {
	Source string
	Files  fs.FS
	Funcs  map[string]any
}

// AddPackage registers Go functions and an optional Nubo wrapper.
func (c *Ctx) AddPackage(name string, pkg Package) error {
	wrapped, err := gobridge.WrapPackage(name, pkg.Funcs)
	if err != nil {
		return err
	}

	if pkg.Source != "" || pkg.Files != nil {
		return c.r.AddSourcePackage(name, wrapped, pkg.Source, pkg.Files)
	}

	return c.r.AddPackage(name, wrapped)
}
