package campaign

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"

	"mapp/internal/history"
	"mapp/internal/settings"
)

// recordChanges lists what a save changes (data model §8): every changed stored field of the record
// without its Shell attributes (history.Diff), then one change per changed attribute, field
// "attributes.<id>", sorted by attribute id. Values that are empty — missing, null, "", [] or {} — count as
// the same.
func recordChanges(before, after any, beforeValues, afterValues settings.AttributeValues) ([]history.Change, error) {
	changes, err := history.Diff(before, after)
	if err != nil {
		return nil, err
	}
	attributeIDs := map[string]bool{}
	for id := range beforeValues {
		attributeIDs[id] = true
	}
	for id := range afterValues {
		attributeIDs[id] = true
	}
	for _, id := range slices.Sorted(maps.Keys(attributeIDs)) {
		old, updated := compactJSON(beforeValues[id]), compactJSON(afterValues[id])
		if bytes.Equal(old, updated) || (emptyJSON(old) && emptyJSON(updated)) {
			continue
		}
		changes = append(changes, history.Change{Field: "attributes." + id, Old: old, New: updated})
	}
	return changes, nil
}

// compactJSON returns value without spaces, so that values sent with spaces compare equal; null for a
// missing value.
func compactJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("null")
	}
	var b bytes.Buffer
	if err := json.Compact(&b, value); err != nil {
		return value // not JSON: compared as it is
	}
	return b.Bytes()
}

// emptyJSON reports whether a stored value counts as empty: null, "", [] or {}.
func emptyJSON(value json.RawMessage) bool {
	switch string(value) {
	case "null", `""`, "[]", "{}":
		return true
	}
	return false
}
