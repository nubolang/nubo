package interpreter

import (
	"context"
	"strings"
	"testing"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
)

func TestCatchWithoutErrorSource(t *testing.T) {
	for _, source := range []*debug.Debug{nil, {File: "catch.nubo", Line: 4}} {
		ir := &Interpreter{
			ctx:       context.Background(),
			dependent: true,
			objects:   make(map[uint32]*entry),
			runtime:   hookTestRuntime{},
		}

		_, err := ir.handleTry(&astnode.Node{
			Content: "caught",
			Debug:   source,
			Body:    []*astnode.Node{{Type: astnode.NodeType(-1)}},
		})
		if err != nil {
			t.Fatal(err)
		}

		caught, ok := ir.GetObject("caught")
		if !ok {
			t.Fatal("catch did not store the error")
		}

		get := func(obj language.Object, name string) language.Object {
			t.Helper()

			value, ok := obj.GetPrototype().GetObject(ir.ctx, name)
			if !ok {
				t.Fatalf("caught error has no field %q", name)
			}

			return value
		}

		if message := get(caught, "message").String(); !strings.Contains(message, "unknown node") {
			t.Fatalf("catch lost the original error: %s", message)
		}

		instance, typed := caught.(*language.StructInstance)
		if !typed || instance.Definition() != language.ErrorStruct() {
			t.Fatal("catch should construct an Error instance")
		}

		metadata := get(caught, "metaData")
		if source == nil {
			if metadata != language.Nil {
				t.Fatal("missing error source should produce nil metadata")
			}
			continue
		}

		wantFile := source.File

		if got := get(metadata, "file").String(); got != wantFile {
			t.Fatalf("caught file is %q, want %q", got, wantFile)
		}
	}
}
