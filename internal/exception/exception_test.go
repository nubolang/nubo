package exception

import (
	"errors"
	"testing"

	"github.com/nubolang/nubo/internal/debug"
)

func TestFromFillsMissingSource(t *testing.T) {
	base := errors.New("original error")
	err := Create("missing source").WithBase(base)
	source := &debug.Debug{File: "main.nubo", Line: 7}
	caller := &debug.Debug{File: "caller.nubo", Line: 3}

	wrapped := From(errors.Join(err), source)
	wrapped = From(wrapped, caller)
	data, ok := Unwrap(wrapped)

	if !ok || data.Debug != source || data.Message != "missing source" {
		t.Fatalf("missing source was not filled: %#v", data)
	}

	if len(data.StackTrace) != 1 || data.StackTrace[0] != caller {
		t.Fatalf("caller trace was not preserved: %#v", data.StackTrace)
	}

	if !errors.Is(wrapped, base) {
		t.Fatal("wrapping lost the original error")
	}
}

func TestFromPreservesExistingSource(t *testing.T) {
	source := &debug.Debug{File: "original.nubo", Line: 2}
	caller := &debug.Debug{File: "caller.nubo", Line: 8}
	err := Create("original").WithDebug(source)

	wrapped := From(From(err, nil), caller)
	data, ok := Unwrap(wrapped)

	if !ok || data.Debug != source {
		t.Fatal("wrapping replaced the original source")
	}

	if len(data.StackTrace) != 1 || data.StackTrace[0] != caller {
		t.Fatal("wrapping lost the caller trace")
	}
}
