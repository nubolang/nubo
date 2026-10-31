package runtime

import (
	"os"
	"testing"
)

func TestVoidValues(t *testing.T) {
	source, err := os.ReadFile("../../tests/runtime/void_values.nubo")
	if err != nil {
		t.Fatal(err)
	}

	result, err := runHookTestSource(t, string(source))
	if err != nil {
		t.Fatal(err)
	}

	if result == nil || result.Value() != true {
		t.Fatalf("void regression checks did not finish: %v", result)
	}
}
