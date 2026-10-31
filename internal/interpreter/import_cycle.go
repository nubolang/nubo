package interpreter

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/nubolang/nubo/internal/debug"
)

type importStackKey struct{}

// Captured contexts can outlive an import.
type importFrame struct {
	path   string
	info   os.FileInfo
	parent *importFrame
	active atomic.Bool
}

func newImportFrame(path string, parent *importFrame) *importFrame {
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}

	info, _ := os.Stat(path)
	frame := &importFrame{
		path:   filepath.Clean(path),
		info:   info,
		parent: parent,
	}

	frame.active.Store(true)

	return frame
}

func (f *importFrame) sameFile(other *importFrame) bool {
	return f.path == other.path || (f.info != nil && other.info != nil && os.SameFile(f.info, other.info))
}

func (ir *Interpreter) beginFileImport(path string, dg *debug.Debug) (context.Context, func(), error) {
	head, _ := ir.ctx.Value(importStackKey{}).(*importFrame)
	source := newImportFrame(ir.currentFile, head)

	// Include the current source if it has no active frame.
	addedSource := head == nil || !head.active.Load() || !head.sameFile(source)
	if addedSource {
		head = source
	}

	target := newImportFrame(path, head)
	finish := func() {
		target.active.Store(false)

		if addedSource {
			source.active.Store(false)
		}
	}

	var chain []string
	for frame := head; frame != nil; frame = frame.parent {
		if !frame.active.Load() {
			continue
		}

		chain = append(chain, frame.path)

		if frame.sameFile(target) {
			slices.Reverse(chain)
			chain = append(chain, frame.path)

			base := filepath.Dir(chain[0])
			for j, file := range chain {
				if relative, err := filepath.Rel(base, file); err == nil {
					chain[j] = filepath.ToSlash(relative)
				}
			}

			finish()

			return nil, nil, runExc("Circular import: %s", strings.Join(chain, " → ")).WithDebug(dg)
		}
	}

	return context.WithValue(ir.ctx, importStackKey{}, target), finish, nil
}
