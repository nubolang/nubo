package lexer

import (
	"strconv"
	"strings"
)

func escapeString(s string, quote rune) (string, error) {
	if quote == '\'' {
		s = strings.ReplaceAll(s, "'", "\"")
		quote = '"'
	}

	s = string(quote) + s + string(quote)
	return strconv.Unquote(s)
}
