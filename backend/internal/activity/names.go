package activity

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// sameName reports whether two activity names are the same name: letter case and surrounding spaces are
// ignored (FR-006).
func sameName(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// namedLike returns the activity among siblings, other than exceptID, whose name is name, and false when
// there is none.
func namedLike(siblings []Activity, name, exceptID string) (Activity, bool) {
	for _, s := range siblings {
		if s.ID != exceptID && sameName(s.Name, name) {
			return s, true
		}
	}
	return Activity{}, false
}

// restoredName returns the first of "<name> (restored)", "<name> (restored 2)", … that no activity among
// siblings other than id has (FR-031, feature 011 FR-022).
func restoredName(siblings []Activity, name, id string) string {
	candidate := name + " (restored)"
	for n := 2; ; n++ {
		if _, taken := namedLike(siblings, candidate, id); !taken {
			return candidate
		}
		candidate = fmt.Sprintf("%s (restored %d)", name, n)
	}
}

// copyName returns the first of "<name> (copy)", "<name> (copy 2)", … that no activity among siblings has
// (FR-030, R13). When the name and the suffix together pass 120 characters, the name is shortened, without
// trailing spaces, so the suffix fits (edge case).
func copyName(siblings []Activity, name string) string {
	for n := 1; ; n++ {
		suffix := " (copy)"
		if n > 1 {
			suffix = fmt.Sprintf(" (copy %d)", n)
		}
		base := name
		if room := maxNameLength - utf8.RuneCountInString(suffix); utf8.RuneCountInString(base) > room {
			base = strings.TrimRight(string([]rune(base)[:room]), " ")
		}
		if _, taken := namedLike(siblings, base+suffix, ""); !taken {
			return base + suffix
		}
	}
}
