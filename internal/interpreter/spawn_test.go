package interpreter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
)

type spawnTestRuntime struct {
	Runtime
	builtins map[string]language.Object
	removed  int
}

func (r *spawnTestRuntime) GetBuiltin(name string) (language.Object, bool) {
	obj, ok := r.builtins[name]
	return obj, ok
}

func (r *spawnTestRuntime) RemoveInterpreter(uint) {
	r.removed++
}

func TestSpawnFailureStopsOnlyWorker(t *testing.T) {
	for _, kind := range []string{"error", "panic"} {
		t.Run(kind, func(t *testing.T) {
			after := 0
			rt := &spawnTestRuntime{builtins: map[string]language.Object{
				"fail": language.NewFunction(func(context.Context, []language.Object) (language.Object, error) {
					if kind == "panic" {
						panic("worker failed")
					}

					return nil, errors.New("worker failed")
				}, nil),
				"after": language.NewFunction(func(context.Context, []language.Object) (language.Object, error) {
					after++
					return nil, nil
				}, nil),
			}}

			ir := &Interpreter{ctx: context.Background(), runtime: rt, objects: make(map[uint32]*entry)}
			source := &debug.Debug{File: "spawn.nubo", Line: 3}
			call := func(name string) *astnode.Node {
				return &astnode.Node{
					Type: astnode.NodeTypeExpression, Debug: source,
					Body: []*astnode.Node{{Type: astnode.NodeTypeFunctionCall, Content: name, Debug: source}},
				}
			}

			err := ir.runSpawn(&astnode.Node{Debug: source, Children: []*astnode.Node{call("fail"), call("after")}})
			data, ok := exception.Unwrap(err)
			if !ok || !strings.Contains(data.Message, "worker failed") || data.Debug == nil {
				t.Fatalf("expected a located worker error, got %v", err)
			}

			if after != 0 || rt.removed != 0 {
				t.Fatal("failed worker continued or detached its parent")
			}

			if err := ir.runSpawn(&astnode.Node{Debug: source, Children: []*astnode.Node{call("after")}}); err != nil {
				t.Fatal(err)
			}

			if after != 1 {
				t.Fatal("another worker could not continue")
			}
		})
	}
}
