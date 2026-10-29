package record

import "strings"

const singleLineMessage = "Names can't contain tabs, line breaks or text-direction characters."

// CheckSingleLine returns the message for a name holding a tab, a line break or a text-direction character
// (which can make a name read differently from what it is), or "" for a usable name.
func CheckSingleLine(name string) string {
	if strings.ContainsFunc(name, notInAName) {
		return singleLineMessage
	}
	return ""
}

func notInAName(r rune) bool {
	switch {
	case r == '\t', r == '\n', r == '\r', r == ' ', r == ' ', r == '‎', r == '‏':
		return true
	case r >= '‪' && r <= '‮', r >= '⁦' && r <= '⁩':
		return true
	}
	return false
}
