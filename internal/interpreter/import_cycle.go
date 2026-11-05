package interpreter

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/source"
)

type importStackKey struct{}

type importFrame struct {
	loader *source.Loader
	path   string
	info   os.FileInfo
	parent *importFrame
	active atomic.Bool
}

func newImportFrame(loader *source.Loader, path string, parent *importFrame) *importFrame {
	path = loader.Key(path)

	var info os.FileInfo
	if loader.Disk() {
		info, _ = loader.Stat(path)
	}

	frame := &importFrame{
		loader: loader,
		path:   filepath.Clean(path),
		info:   info,
		parent: parent,
	}
	frame.active.Store(true)

	return frame
}

func (f *importFrame) sameFile(other *importFrame) bool {
	if f.loader != other.loader && (!f.loader.Disk() || !other.loader.Disk()) {
		return false
	}

	return f.path == other.path || (f.info != nil && other.info != nil && os.SameFile(f.info, other.info))
}

func (ir *Interpreter) beginFileImport(target string, dg *debug.Debug) (context.Context, func(), error) {
	var l *source.Loader
	if ir.runtime != nil {
		l = ir.runtime.Source()
	}
	ctx := ir.ctx
	current := ir.currentFile

	head, _ := ctx.Value(importStackKey{}).(*importFrame)
	currentFrame := newImportFrame(l, current, head)

	addedCurrentFile := head == nil || !head.active.Load() || !head.sameFile(currentFrame)
	if addedCurrentFile {
		head = currentFrame
	}

	targetFrame := newImportFrame(l, target, head)
	finish := func() {
		targetFrame.active.Store(false)
		if addedCurrentFile {
			currentFrame.active.Store(false)
		}
	}

	var chain []string
	for frame := head; frame != nil; frame = frame.parent {
		if !frame.active.Load() {
			continue
		}

		chain = append(chain, frame.path)
		if frame.sameFile(targetFrame) {
			slices.Reverse(chain)
			chain = append(chain, frame.path)

			base := filepath.Dir(chain[0])
			for index, file := range chain {
				if relative, err := filepath.Rel(base, file); err == nil {
					chain[index] = filepath.ToSlash(relative)
				}
			}

			finish()

			return nil, nil, runExc("Circular import: %s", strings.Join(chain, " → ")).WithDebug(dg)
		}
	}

	return context.WithValue(ctx, importStackKey{}, targetFrame), finish, nil
}
