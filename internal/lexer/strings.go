package lexer

import (
	"strconv"
)

func escapeString(s string, quote rune) (string, error) {
	s = string(quote) + s + string(quote)
	return strconv.Unquote(s)
}
