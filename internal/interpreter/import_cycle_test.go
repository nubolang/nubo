package interpreter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
)

func TestImportCycleChain(t *testing.T) {
	dir := t.TempDir()
	a := &Interpreter{ctx: context.Background(), currentFile: filepath.Join(dir, "a.nubo")}

	ctx, finish, err := a.beginFileImport(filepath.Join(dir, "b.nubo"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer finish()

	b := &Interpreter{ctx: ctx, currentFile: filepath.Join(dir, "b.nubo")}
	dg := &debug.Debug{File: b.currentFile, Line: 3}

	_, _, err = b.beginFileImport(a.currentFile, dg)
	data, ok := exception.Unwrap(err)
	if !ok {
		t.Fatalf("expected a Nubo error, got %v", err)
	}

	if data.Message != "Circular import: a.nubo → b.nubo → a.nubo" || data.Debug != dg {
		t.Fatalf("unexpected cycle error: %+v", data)
	}

	// Completed imports must not stay active in captured contexts.
	finish()

	_, finishLater, err := b.beginFileImport(a.currentFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	finishLater()
}

func TestImportCycleFileAliases(t *testing.T) {
	for _, link := range []struct {
		name   string
		create func(string, string) error
	}{
		{name: "symlink", create: os.Symlink},
		{name: "hardlink", create: os.Link},
	} {
		t.Run(link.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "source.nubo")
			alias := filepath.Join(dir, "alias.nubo")

			if err := os.WriteFile(file, nil, 0600); err != nil {
				t.Fatal(err)
			}

			if err := link.create(file, alias); err != nil {
				t.Skipf("file links unavailable: %v", err)
			}

			ir := &Interpreter{ctx: context.Background(), currentFile: file}
			_, _, err := ir.beginFileImport(alias, nil)

			data, ok := exception.Unwrap(err)
			if !ok || data.Message != "Circular import: source.nubo → source.nubo" {
				t.Fatalf("file alias hid the cycle: %v", err)
			}
		})
	}
}

func TestIndependentImportBranches(t *testing.T) {
	dir := t.TempDir()
	ir := &Interpreter{ctx: context.Background(), currentFile: filepath.Join(dir, "main.nubo")}
	results := make(chan error, 2)
	release := make(chan struct{})
	defer close(release)

	for range 2 {
		go func() {
			_, finish, err := ir.beginFileImport(filepath.Join(dir, "shared.nubo"), nil)
			results <- err

			if err == nil {
				defer finish()
				<-release
			}
		}()
	}

	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}

type detachTestRuntime struct {
	Runtime
	removed map[uint]int
}

func (r *detachTestRuntime) RemoveInterpreter(id uint) {
	r.removed[id]++
}

func TestDetachImportCycle(t *testing.T) {
	runtime := &detachTestRuntime{removed: make(map[uint]int)}
	a := &Interpreter{ID: 1, runtime: runtime}
	b := &Interpreter{ID: 2, runtime: runtime}
	c := &Interpreter{ID: 3, runtime: runtime}

	a.imports = map[string]*Interpreter{"self": a, "b": b, "c": c}
	b.imports = map[string]*Interpreter{"a": a, "c": c}
	c.imports = map[string]*Interpreter{"a": a}

	done := make(chan struct{})
	go func() {
		a.MustDetach()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup deadlocked on an import cycle")
	}

	for _, ir := range []*Interpreter{a, b, c} {
		if runtime.removed[ir.ID] != 1 {
			t.Fatalf("interpreter %d cleaned up %d times", ir.ID, runtime.removed[ir.ID])
		}
	}
}
