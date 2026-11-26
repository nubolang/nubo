package execution

import (
	"context"
	"errors"
	"testing"
)

func TestCleanup(t *testing.T) {
	_, r := New(context.Background())
	n := 0
	r.Add(func() error { n++; return errors.New("cleanup") })
	if r.Close() == nil || n != 1 {
		t.Fatal("cleanup failed")
	}
	if r.Close() == nil || n != 1 {
		t.Fatal("repeated cleanup")
	}
}
func TestCallbacks(t *testing.T) {
	_, r := New(context.Background())
	n := 0
	r.Enqueue(func() error { n++; return nil })
	if r.Check() != nil || n != 1 {
		t.Fatal("callback")
	}
	r.Stop(errors.New("stop"))
	if r.Check() == nil {
		t.Fatal("termination")
	}
}

func TestCleanupOnPanic(t *testing.T) {
	_, r := New(context.Background())
	cleaned := false
	r.Add(func() error { cleaned = true; return nil })
	func() { defer func() { _ = recover() }(); defer r.Close(); panic("recoverable panic") }()
	if !cleaned {
		t.Fatal("panic skipped native cleanup")
	}
	if _, err := r.Add(func() error { return nil }); err == nil {
		t.Fatal("registered resource after execution ended")
	}
}
