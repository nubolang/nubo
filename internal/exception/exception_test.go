package exception

import (
	"errors"
	"fmt"
	"sync"
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

func TestFromDoesNotChangeSharedException(t *testing.T) {
	source := &debug.Debug{File: "module.nubo", Line: 2}
	trace := &debug.Debug{File: "helper.nubo", Line: 4}
	caller := &debug.Debug{File: "caller.nubo", Line: 6}
	cause := errors.New("original cause")
	original := Create("shared failure").WithBase(cause).WithDebug(source).WithStatusCode(422)
	original.trace = make([]*debug.Debug, 1, 8)
	original.trace[0] = trace

	wrapped := From(errors.Join(original), caller).WithLevel(LevelRuntime)
	if wrapped == original {
		t.Fatal("wrapping should create a separate exception")
	}
	if original.level != LevelFatal || len(original.trace) != 1 {
		t.Fatal("wrapping changed the shared exception")
	}
	if wrapped.StatusCode() != 422 || !errors.Is(wrapped, cause) {
		t.Fatal("wrapping lost the status or original cause")
	}

	wrapped.trace[0] = caller
	if original.trace[0] != trace {
		t.Fatal("wrapped exceptions shared trace storage")
	}
}

func TestConcurrentWrappingOfSharedException(t *testing.T) {
	source := &debug.Debug{File: "module.nubo", Line: 2}
	trace := &debug.Debug{File: "helper.nubo", Line: 4}
	original := Create("shared failure").WithDebug(source).WithTrace(trace)
	shared := errors.Join(original)
	start := make(chan struct{})
	results := make(chan *Expection, 16)
	var workers sync.WaitGroup

	for index := range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start

			caller := &debug.Debug{File: fmt.Sprintf("caller%d.nubo", index), Line: 6}
			wrapped := From(shared, caller).WithLevel(LevelSemantic)
			results <- wrapped

			for step := range 16 {
				next := &debug.Debug{File: fmt.Sprintf("next%d.nubo", step), Line: 8}
				_ = From(wrapped, next).WithLevel(LevelRuntime).Error()
				_ = shared.Error()
			}
		}()
	}

	close(start)
	for range 16 {
		wrapped := <-results
		_ = wrapped.Error()
		data := wrapped.Details()
		if data.Level != LevelSemantic || data.Debug != source || len(data.StackTrace) != 2 {
			t.Errorf("caller exception was changed by another wrapper: %#v", data)
		}
	}
	workers.Wait()

	if original.level != LevelFatal || len(original.trace) != 1 || original.trace[0] != trace {
		t.Fatal("concurrent wrapping changed the shared exception")
	}
}
