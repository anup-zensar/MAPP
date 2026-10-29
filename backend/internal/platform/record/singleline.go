package record

import "strings"

const msgNotSingleLine = "Names can't contain tabs, line breaks or text-direction characters."

// CheckSingleLine returns the message for a name holding a tab, a line break or a text-direction control
// character (U+200E/F, U+202A–U+202E, U+2066–U+2069), or "" when the name is fine.
func CheckSingleLine(name string) string {
	if strings.IndexFunc(name, notSingleLine) >= 0 {
		return msgNotSingleLine
	}
	return ""
}

func notSingleLine(r rune) bool {
	switch {
	case r == '\t', r == '\n', r == '\r', r == '\u200e', r == '\u200f':
		return true
	case r >= '\u202a' && r <= '\u202e':
		return true
	default:
		return r >= '\u2066' && r <= '\u2069'
	}
}
