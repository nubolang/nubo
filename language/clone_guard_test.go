package language

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRecursiveCloneGuardUnwinds(t *testing.T) {
	for _, outcome := range []string{"success", "error", "panic"} {
		t.Run(outcome, func(t *testing.T) {
			var instance *StructInstance
			var captured context.Context
			calls := 0
			failure := errors.New("clone failed after recursion")
			definition := NewStruct("RecursiveClone", nil, nil)

			err := definition.GetPrototype().SetObject(context.Background(), "__clone__", NewFunction(func(ctx context.Context, _ []Object) (Object, error) {
				calls++
				captured = ctx

				for range 2 {
					result, err := Clone(ctx, instance)
					if err != nil || result != instance {
						return nil, errors.New("recursive clone lost its fallback")
					}
				}

				switch outcome {
				case "error":
					return nil, failure
				case "panic":
					panic(failure)
				default:
					return instance, nil
				}
			}, nil))
			if err != nil {
				t.Fatal(err)
			}

			instance, err = definition.NewInstance()
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			for range 3 {
				result, err := Clone(ctx, instance)
				if outcome == "success" {
					if err != nil || result != instance {
						t.Fatalf("clone failed: %v", err)
					}
				} else if result != nil || !errors.Is(err, failure) {
					t.Fatalf("clone lost the expected failure: %v", err)
				}

				ctx = captured
			}

			if calls != 3 {
				t.Fatalf("completed clones kept their guard active: %d calls", calls)
			}
		})
	}
}

func TestConcurrentClonesAreIndependent(t *testing.T) {
	ctx := context.Background()
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	defer close(release)

	var instance *StructInstance
	definition := NewStruct("ConcurrentClone", nil, nil)
	err := definition.GetPrototype().SetObject(ctx, "__clone__", NewFunction(func(context.Context, []Object) (Object, error) {
		entered <- struct{}{}
		<-release

		return instance, nil
	}, nil))
	if err != nil {
		t.Fatal(err)
	}

	instance, err = definition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := Clone(ctx, instance)
			done <- err
		}()
	}

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for range 2 {
		select {
		case <-entered:
		case err := <-done:
			t.Fatalf("independent clone skipped its hook: %v", err)
		case <-timer.C:
			t.Fatal("concurrent clones interfered")
		}
	}

	for range 2 {
		select {
		case release <- struct{}{}:
		case <-timer.C:
			t.Fatal("concurrent clones did not resume")
		}
	}

	for range 2 {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-timer.C:
			t.Fatal("concurrent clones did not finish")
		}
	}
}

func TestNewCloneContextStartsIndependentChain(t *testing.T) {
	ctx, leave, ok := enterStructClone(context.Background(), "type")
	if !ok {
		t.Fatal("could not enter the parent clone")
	}
	defer leave()

	_, _, ok = enterStructClone(ctx, "type")
	if ok {
		t.Fatal("recursive clone was not guarded")
	}

	_, leaveChild, ok := enterStructClone(NewCloneContext(ctx), "type")
	if !ok {
		t.Fatal("independent chain inherited its parent's guard")
	}
	leaveChild()
}
