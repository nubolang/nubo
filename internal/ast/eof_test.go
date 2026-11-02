package ast_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/lexer"
)

func TestIncompleteStatementsReturnErrors(t *testing.T) {
	for _, source := range []string{
		"items[0]",
		"items[0][1]",
		"items[0] =",
		"item =",
		"item.value =",
		"let item =",
		"const item =",
		"items[",
		"fn unfinished() int",
		"fn unfinished() int?",
		"fn unfinished() []",
		"fn unfinished() (int",
		"fn unfinished() fn(int",
		"call(\n",
		"call(\n    1,\n",
		"if true\n",
	} {
		t.Run(source, func(t *testing.T) {
			for _, ending := range []string{"", "   ", "\n"} {
				lx, err := lexer.New(strings.NewReader(source+ending), "incomplete.nubo")
				if err != nil {
					t.Fatal(err)
				}

				tokens, err := lx.Parse()
				if err != nil {
					t.Fatal(err)
				}

				_, err = ast.New(context.Background()).Parse(tokens)
				if err == nil {
					t.Fatalf("incomplete statement with ending %q was accepted", ending)
				}

				_, _, source := debug.Unwrap(err)
				if source == nil || source.File != "incomplete.nubo" {
					t.Fatalf("syntax error has no source location: %v", err)
				}
			}
		})
	}
}
