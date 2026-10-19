package activities

import (
	"fmt"
	"strings"
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
