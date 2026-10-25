package lexer

import (
	"strconv"
	"strings"
)

func escapeString(s string, quote rune) (string, error) {
	if quote == '\'' {
		var normalized strings.Builder
		for i := 0; i < len(s); i++ {
			switch s[i] {
			case '\\':
				if i+1 < len(s) {
					i++
					if s[i] != '\'' {
						normalized.WriteByte('\\')
					}
					normalized.WriteByte(s[i])
				} else {
					normalized.WriteByte('\\')
				}
			case '"':
				normalized.WriteString(`\"`)
			default:
				normalized.WriteByte(s[i])
			}
		}
		s = normalized.String()
		quote = '"'
	}

	s = string(quote) + s + string(quote)
	return strconv.Unquote(s)
}
