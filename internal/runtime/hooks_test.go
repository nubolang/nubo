package runtime

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/lexer"
	"github.com/nubolang/nubo/language"
)

func TestRecursiveAccessHooks(t *testing.T) {
	for _, tc := range []struct {
		name    string
		getBody string
		setBody string
		access  string
	}{
		{"property getter", "return self.missing", "", "return item.missing"},
		{"indexed getter", "return self[key]", "", `return item["missing"]`},
		{"property setter", "return 0", "self.value = value", "item.value = 7"},
		{"indexed setter", "return 0", "self[key] = value", `item["value"] = 7`},
		{"intermediate getter", "return self[key]", "", `item["missing"]["value"] = 7`},
		{"mixed hooks", "self.value = 7\nreturn 0", "let result = self.missing", "return item.missing"},
		{"explicit getter", "return self.__get__(key)", "", `return item.__get__("missing")`},
		{"explicit setter", "return 0", "self.__set__(key, value)", `item.__set__("value", 7)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf(`
struct Loop {
    value: int
}

impl Loop {
    fn __get__(self: Loop, key: string) any {
        %s
    }

    fn __set__(self: Loop, key: string, value: any) void {
        %s
    }
}

let item = Loop()
%s
`, tc.getBody, tc.setBody, tc.access)

			_, err := runHookTestSource(t, source)
			data, ok := exception.Unwrap(err)
			if !ok || !strings.Contains(data.Message, "getter/setter recursion limit exceeded") {
				t.Fatalf("expected a hook recursion error, got %v", err)
			}

			if data.Debug == nil || len(data.StackTrace) == 0 {
				t.Fatal("recursion error has no source location or trace")
			}
		})
	}
}

func TestHookErrorCanBeCaught(t *testing.T) {
	result, err := runHookTestSource(t, `
struct Loop {
    value: int
}

impl Loop {
    fn __get__(self: Loop, key: string) int {
        if key == "good" {
            return 7
        }

        return self.missing
    }

    fn __set__(self: Loop, key: string, value: any) void {
        self.value = value
    }
}

let item = Loop()

catch getError {
    let value = item.missing
}

catch setError {
    item.value = 9
}

let getMessage = getError.message
let setMessage = setError.message

return getMessage.includes("recursion limit") && setMessage.includes("recursion limit") && item.good == 7 && item.value == 0
`)
	if err != nil {
		t.Fatal(err)
	}

	if result == nil || result.Value() != true {
		t.Fatalf("catch, unwind, or setter protection failed: %v", result)
	}
}

func TestTerminatingHookRecursion(t *testing.T) {
	result, err := runHookTestSource(t, `
struct Counter {}

impl Counter {
    fn __get__(self: Counter, key: int) int {
        if key == 0 {
            return 7
        }

        return self[key - 1]
    }
}

let counter = Counter()
return counter[10]
`)
	if err != nil {
		t.Fatal(err)
	}

	if result == nil || result.String() != "7" {
		t.Fatalf("terminating recursion failed: %v", result)
	}
}

func runHookTestSource(t *testing.T, source string) (language.Object, error) {
	t.Helper()

	lx, err := lexer.New(strings.NewReader(source), "hooks.nubo")
	if err != nil {
		t.Fatal(err)
	}

	tokens, err := lx.Parse()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nodes, err := ast.New(ctx).Parse(tokens)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		value language.Object
		err   error
	}

	done := make(chan result, 1)
	go func() {
		value, err := New(nil).WithContext(ctx).Interpret("hooks.nubo", nodes)
		done <- result{value: value, err: err}
	}()

	select {
	case result := <-done:
		return result.value, result.err
	case <-ctx.Done():
		t.Fatal("recursive hook did not finish")
		return nil, ctx.Err()
	}
}
