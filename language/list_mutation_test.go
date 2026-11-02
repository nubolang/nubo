package language

import (
	"context"
	"testing"
	"time"
)

func TestListConcurrentMutations(t *testing.T) {
	const workers = 4
	const calls = 100
	const total = workers * calls

	index := NewInt(0, nil)
	value := NewInt(7, nil)

	for _, tc := range []struct {
		name  string
		args  []Object
		start int
		want  int
	}{
		{"push", []Object{value}, 0, total},
		{"insert", []Object{index, value}, 0, total},
		{"unshift", []Object{value}, 0, total},
		{"del", []Object{index}, total, 0},
		{"pop", nil, total, 0},
		{"shift", nil, total, 0},
		{"clear", nil, total, 0},
		{"reverse", nil, total, total},
		{"__set__", []Object{index, value}, total, total},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := make([]Object, tc.start)
			for i := range items {
				items[i] = NewInt(int64(i), nil)
			}

			list := NewList(items, TypeInt, nil)
			proto := list.GetPrototype().(*ListPrototype)
			mutate := listTestMethod(t, proto, tc.name)
			length := listTestMethod(t, proto, "length")
			ctx := context.Background()

			start := make(chan struct{})
			done := make(chan error, workers+1)

			for range workers {
				go func() {
					<-start

					for range calls {
						if _, err := mutate.Call(ctx, tc.args); err != nil {
							done <- err
							return
						}
					}

					done <- nil
				}()
			}

			go func() {
				<-start

				for range total {
					if _, err := length.Call(ctx, nil); err != nil {
						done <- err
						return
					}

					proto.callbackItems()
				}

				done <- nil
			}()

			close(start)

			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()

			for range workers + 1 {
				select {
				case err := <-done:
					if err != nil {
						t.Error(err)
					}
				case <-timer.C:
					t.Fatal("concurrent list operations did not finish")
				}
			}

			if got := len(proto.callbackItems()); got != tc.want {
				t.Fatalf("got %d entries, want %d", got, tc.want)
			}
		})
	}
}
