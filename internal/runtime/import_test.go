package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
)

func TestCircularImports(t *testing.T) {
	importTestDirectory(t)

	for _, tc := range []struct {
		name  string
		edges map[string]string
		cycle string
	}{
		{
			name:  "self",
			edges: map[string]string{"a": "a"},
			cycle: "a.nubo → a.nubo",
		},
		{
			name:  "entry",
			edges: map[string]string{"a": "b", "b": "a"},
			cycle: "a.nubo → b.nubo → a.nubo",
		},
		{
			name:  "unregistered",
			edges: map[string]string{"a": "b", "b": "c", "c": "b"},
			cycle: "b.nubo → c.nubo → b.nubo",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			for from, to := range tc.edges {
				source := fmt.Sprintf("import dependency from %q\n", filepath.Join(dir, to))
				writeImportTestFile(t, filepath.Join(dir, from+".nubo"), source)
			}

			runtime := New(nil)
			entry := filepath.Join(dir, "a.nubo")

			// A failed import must not poison the next attempt.
			for range 2 {
				_, err := runImportTestFile(t, runtime, entry)
				data, ok := exception.Unwrap(err)
				if !ok || data.Message != "Circular import: "+tc.cycle {
					t.Fatalf("unexpected import error: %v", err)
				}

				if len(runtime.filemap) != 0 || len(runtime.interpreters) != 0 {
					t.Fatal("failed import left registered interpreters")
				}
			}
		})
	}
}

func TestSharedImportDependency(t *testing.T) {
	dir := importTestDirectory(t)
	shared := filepath.Join(dir, "shared.nubo")
	left := filepath.Join(dir, "left.nubo")
	right := filepath.Join(dir, "right.nubo")
	entry := filepath.Join(dir, "main.nubo")

	writeImportTestFile(t, shared, "")
	writeImportTestFile(t, left, fmt.Sprintf("import shared from %q\n", shared))
	writeImportTestFile(t, right, fmt.Sprintf("import shared from %q\n", shared))
	writeImportTestFile(t, entry, fmt.Sprintf("import left from %q\nimport right from %q\nreturn 7\n", left, right))

	result, err := runImportTestFile(t, New(nil), entry)
	if err != nil {
		t.Fatal(err)
	}

	if result == nil || result.String() != "7" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func importTestDirectory(t *testing.T) string {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Keep the parser cache in the temporary directory.
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Error(err)
		}
	})

	return dir
}

func writeImportTestFile(t *testing.T, file, source string) {
	t.Helper()

	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
}

func runImportTestFile(t *testing.T, runtime *Runtime, file string) (language.Object, error) {
	t.Helper()

	nodes, err := native.NodesFromFile(file, file)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		value language.Object
		err   error
	}

	done := make(chan result, 1)
	go func() {
		value, err := runtime.Interpret(file, nodes)
		done <- result{value: value, err: err}
	}()

	select {
	case result := <-done:
		return result.value, result.err
	case <-time.After(5 * time.Second):
		t.Fatal("import did not finish; possible cycle or cleanup deadlock")
		return nil, nil
	}
}
