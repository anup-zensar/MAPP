package settings

import (
	"encoding/json"
	"fmt"
	"slices"
)

// isOptionOf reports whether id is an option of the attribute a.
func (f formData) isOptionOf(a Attribute, id string) bool {
	o, found := f.optionByID[id]
	return found && o.AttributeID == a.ID
}

// newOptionIDs returns the options of value that before did not have: the ones newly chosen.
func newOptionIDs(a Attribute, value, before json.RawMessage) []string {
	if !a.IsChoice() || value == nil {
		return nil
	}
	had := optionIDsOf(a, before)
	return filter(optionIDsOf(a, value), func(id string) bool { return !slices.Contains(had, id) })
}

// newInactiveOption returns the problem of an inactive option chosen newly in value (FR-004): one the record
// already had stays (FR-027).
func (f formData) newInactiveOption(a Attribute, value, before json.RawMessage) string {
	for _, id := range newOptionIDs(a, value, before) {
		if o := f.optionByID[id]; !o.Active {
			return fmt.Sprintf(inactiveOptionMessage, o.Label)
		}
	}
	return ""
}

// linked reports whether the option id is linked to any of the parent options selected.
func (f formData) linked(id string, selected []string) bool {
	for _, parentID := range f.optionByID[id].ParentOptionIDs {
		if slices.Contains(selected, parentID) {
			return true
		}
	}
	return false
}

// clearUnlinked returns the options of the dependent attribute a's value that no longer fit the parent
// options selected (FR-025).
func (f formData) clearUnlinked(a Attribute, value json.RawMessage, selected []string) ClearedValue {
	cleared := ClearedValue{AttributeID: a.ID, Label: a.Label, OptionIDs: []string{}, OptionLabels: []string{}}
	for _, id := range optionIDsOf(a, value) {
		if !f.linked(id, selected) {
			cleared.OptionIDs = append(cleared.OptionIDs, id)
			cleared.OptionLabels = append(cleared.OptionLabels, f.optionByID[id].Label)
		}
	}
	return cleared
}
