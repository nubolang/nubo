package lexer

import "unicode"

// isLetter returns true if the given character is a letter
func isLetter(r rune) bool {
	return unicode.IsLetter(r) || r == '$' || r == '_'
}

func IsIdentifier(s string) bool {
	for _, r := range s {
		if !isLetter(r) {
			return false
		}
	}
	return true
}

// isDigit returns true if the given character is a digit
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
