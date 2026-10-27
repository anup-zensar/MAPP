package settings

import (
	"slices"
)

// parentOf returns the attribute a depends on, and whether a may depend on it (FR-022): a is a choice
// attribute and its parent a single or multiple choice attribute of the same level that comes before it.
func (f formData) parentOf(a Attribute) (Attribute, bool) {
	parent, found := f.byID[a.ParentAttributeID]
	if !found || parent.ID == a.ID {
		return Attribute{}, false
	}
	return parent, a.IsChoice() && parent.Level == a.Level &&
		(parent.Type == TypeSingleChoice || parent.Type == TypeMultipleChoice) && f.isBefore(parent, a)
}

// isBefore reports whether first comes before second in the display order of their level, with both as
// given and every other attribute as stored.
func (f formData) isBefore(first, second Attribute) bool {
	level := []Attribute{first, second}
	for _, a := range f.byID {
		if a.Level == second.Level && a.ID != first.ID && a.ID != second.ID {
			level = append(level, a)
		}
	}
	for _, a := range displayOrder(level) {
		switch a.ID {
		case first.ID:
			return true
		case second.ID:
			return false
		}
	}
	return false
}

// levelAttributes returns the stored attributes of level in display order.
func (f formData) levelAttributes(level Level) []Attribute {
	all := make([]Attribute, 0, len(f.byID))
	for _, a := range f.byID {
		all = append(all, a)
	}
	return ofLevel(all, level)
}

// lastPosition is the place of the attribute a put last in its section (data model §4.2): after the last
// attribute of its section, or after the last of its level when its section is new.
func (f formData) lastPosition(a Attribute) int {
	inSection, inLevel := 0, 0
	for _, other := range f.byID {
		if other.ID == a.ID || other.Level != a.Level {
			continue
		}
		inLevel = max(inLevel, other.Position)
		if other.Section == a.Section {
			inSection = max(inSection, other.Position)
		}
	}
	if inSection > 0 {
		return inSection + 1
	}
	return inLevel + 1
}

// publishes reports whether records of level are published (plan, campaign, initiative) rather than
// submitted (activity, line item).
func publishes(level Level) bool {
	return level == LevelPlan || level == LevelCampaign || level == LevelInitiative
}

// attributeKey gives an attribute's id and label, for findNamed.
func attributeKey(a Attribute) (string, string) {
	return a.ID, a.Label
}

// firstNamed returns the name among names that is name, ignoring letter case and surrounding spaces, or
// "" when there is none.
func firstNamed(names []string, name string) string {
	for _, n := range names {
		if sameName(n, name) {
			return n
		}
	}
	return ""
}

// sameSet reports whether two lists of ids hold the same ids, in any order.
func sameSet(a, b []string) bool {
	a, b = unique(a), unique(b)
	if len(a) != len(b) {
		return false
	}
	for _, id := range a {
		if !slices.Contains(b, id) {
			return false
		}
	}
	return true
}
