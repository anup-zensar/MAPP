package activities

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// A copy is named "<name> (copy)", then " (copy 2)", " (copy 3)", … — the first that no activity of the
// initiative has, ignoring letter case and surrounding spaces (FR-030, R13).
func TestCopyName_Suffixes_FR030(t *testing.T) {
	siblings := []Activity{{ID: "A-1", Name: "Launch film"}, {ID: "A-2", Name: " launch FILM (copy) "}, {ID: "A-3", Name: "Launch film (copy 2)"}}

	if got := copyName(nil, "Launch film"); got != "Launch film (copy)" {
		t.Errorf("first copy = %q", got)
	}
	if got := copyName(siblings, "Launch film"); got != "Launch film (copy 3)" {
		t.Errorf("third copy = %q, want Launch film (copy 3)", got)
	}
}

// A copy's name stays within 120 characters: the original part is shortened, without trailing spaces, so
// the suffix fits (edge case).
func TestCopyName_LongNameShortened_EdgeCase(t *testing.T) {
	long := strings.Repeat("ö", 112) + " " + strings.Repeat("y", 7) // 120 characters

	if got, want := copyName(nil, long), strings.Repeat("ö", 112)+" (copy)"; got != want {
		t.Errorf("copy name = %q, want %q", got, want)
	}
	second := copyName([]Activity{{ID: "A-1", Name: strings.Repeat("ö", 112) + " (copy)"}}, long)
	if utf8.RuneCountInString(second) != 120 || second != strings.Repeat("ö", 111)+" (copy 2)" {
		t.Errorf("second copy name = %q (%d characters), want 120 ending in (copy 2)", second, utf8.RuneCountInString(second))
	}
}
