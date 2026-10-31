package parsers

import (
	"testing"

	"github.com/nubolang/nubo/internal/lexer"
)

func TestTokenHelpersHandleEmptyInput(t *testing.T) {
	for _, advance := range []func([]*lexer.Token, *int) error{inxPP, inxNlPP, inxPPIf, skipLineBreaks} {
		for _, offset := range []int{0, 1} {
			index := offset
			if err := advance(nil, &index); err == nil {
				t.Fatal("empty input should return an EOF error")
			}
		}
	}

	index := 0
	if _, err := inxPPeak(nil, &index); err == nil {
		t.Fatal("peeking at empty input should return an EOF error")
	}
}
