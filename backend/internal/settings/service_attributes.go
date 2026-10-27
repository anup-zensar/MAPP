package settings

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

// attributeToChange reads every attribute and option, the attribute id — or answers not found — and the
// usage index, inside the caller's transaction.
func (s *Service) attributeToChange(ctx context.Context, id string) (formData, Attribute, *usageIndex, error) {
	f, err := s.loadForm(ctx, "")
	if err != nil {
		return formData{}, Attribute{}, nil, err
	}
	stored, found := f.byID[id]
	if !found {
		return formData{}, Attribute{}, nil, errs.NotFound(attributeGoneMessage)
	}
	u, err := s.usage(ctx)
	if err != nil {
		return formData{}, Attribute{}, nil, err
	}
	return f, stored, u, nil
}

// attributeChange is a checked change of a stored Shell attribute: the attribute after it, its options
// whose links change (with their new links), and the options it removes because its new type has none.
type attributeChange struct {
	attribute Attribute
	linked    []Option
	removed   []Option
}

// planAttributeChange checks the change update of the stored attribute, adding every problem to problems,
// and works out what it changes: the attribute with its locks (data model §4.1), the options a type
// without options removes (D-049), and the links of its options (FR-022) — set from optionLinks, which must
// link every active option when the parent changes, and removed when it no longer depends on anything.
func (f formData) planAttributeChange(stored Attribute, update ShellAttributeUpdate, u *usageIndex, problems *errs.Problems) attributeChange {
	change := attributeChange{attribute: f.applyAttribute(stored, update, attributeLocks(stored, u), problems)}
	changed, options := change.attribute, f.options[stored.ID]
	if stored.IsChoice() && !changed.IsChoice() {
		change.removed = options
		return change
	}
	if changed.ParentAttributeID == "" {
		for _, o := range options {
			if len(o.ParentOptionIDs) > 0 {
				o.ParentOptionIDs = []string{}
				change.linked = append(change.linked, o)
			}
		}
		return change
	}
	parent, valid := f.parentOf(changed)
	if !valid {
		return change // applyAttribute reported it
	}
	links := map[string][]string{}
	if update.OptionLinks != nil {
		for _, l := range *update.OptionLinks {
			if o, found := f.optionByID[l.OptionID]; !found || o.AttributeID != stored.ID {
				problems.Add("optionLinks."+l.OptionID, optionGoneMessage)
				continue
			}
			links[l.OptionID] = l.ParentOptionIDs
		}
	}
	newParent := changed.ParentAttributeID != stored.ParentAttributeID
	for _, o := range options {
		field := "optionLinks." + o.ID
		given, sent := links[o.ID]
		previous := o.ParentOptionIDs
		if newParent {
			previous = nil // links to the old parent's options are not links to the new one's
		}
		var next []string
		switch {
		case sent && !o.Active && len(unique(given)) == 0:
			next = []string{} // an inactive option may stay unlinked
		case sent:
			next = f.checkLinks(problems, field, o.Label, parent, given, previous)
		case !newParent:
			continue // keeps its links
		case o.Active:
			problems.Add(field, fmt.Sprintf(linkOptionMessage, o.Label, midSentence(parent.Label)))
			continue
		default:
			next = []string{}
		}
		if !slices.Equal(next, o.ParentOptionIDs) {
			o.ParentOptionIDs = next
			change.linked = append(change.linked, o)
		}
	}
	return change
}

// applyAttribute returns the attribute a with the fields update sends, and adds every problem with them to
// problems (data model §4): each field's own rules; the lock reasons of locked fields that change (data
// model §4.1); and the rules between fields — % split and the marking by level and type, the label unique
// within the level, the place of an attribute that is new or changes section or level (last in its
// section, §4.2), its parent and its condition. A type without options depends on nothing. Fields not sent
// stay as they are; f holds every attribute and option.
func (f formData) applyAttribute(a Attribute, update ShellAttributeUpdate, locks []Lock, problems *errs.Problems) Attribute {
	before := a
	allowed := func(field string, changes bool) bool { // false, with the lock's reason, when a locked field changes
		reason, locked := lockOf(locks, field)
		if locked && changes {
			problems.Add(field, reason)
			return false
		}
		return true
	}
	if update.Label != nil {
		a.Label = strings.TrimSpace(*update.Label)
		checkText(problems, "label", a.Label, enterLabelMessage, maxAttributeLabel)
	}
	if update.HelpText != nil {
		a.HelpText = strings.TrimSpace(*update.HelpText)
		checkLength(problems, "helpText", a.HelpText, maxHelpText)
	}
	levelKnown, typeKnown, markingKnown := true, true, true
	if update.Level != nil {
		if level, ok := parseLevel(strings.TrimSpace(*update.Level)); !ok {
			problems.Add("level", chooseLevelMessage)
			levelKnown = false
		} else if allowed("level", level != before.Level) {
			a.Level = level
		}
	}
	if update.Type != nil {
		if t := AttributeType(strings.TrimSpace(*update.Type)); typeWords[t] == "" {
			problems.Add("type", chooseTypeMessage)
			typeKnown = false
		} else if allowed("type", t != before.Type) {
			a.Type = t
		}
	}
	if update.Marking != nil {
		if m := Marking(strings.TrimSpace(*update.Marking)); markingWords[m] == "" {
			problems.Add("marking", chooseMarkingMessage)
			markingKnown = false
		} else {
			a.Marking = m
		}
	}
	if update.Section != nil {
		section := cmp.Or(strings.TrimSpace(*update.Section), DefaultSection)
		checkLength(problems, "section", section, maxSectionName)
		if allowed("section", section != before.Section) {
			a.Section = section
		}
	}
	if update.ParentAttributeID != nil {
		if parentID := strings.TrimSpace(*update.ParentAttributeID); allowed("parentAttributeId", parentID != before.ParentAttributeID) {
			a.ParentAttributeID = parentID
		}
	} else if typeKnown && !a.Type.IsChoice() {
		a.ParentAttributeID = ""
	}
	if update.ConditionOptionIDs != nil {
		if condition := unique(*update.ConditionOptionIDs); allowed("condition", !sameSet(condition, before.Condition)) {
			a.Condition = condition
		}
	}

	if levelKnown && typeKnown && a.Type == TypeMultipleChoiceSplit && publishes(a.Level) {
		problems.Add("type", splitLevelMessage)
	}
	if levelKnown && typeKnown && markingKnown {
		switch {
		case a.Type == TypeYesNo && a.Marking != MarkingOptional:
			problems.Add("marking", yesNoMarkingMessage)
		case a.Marking == MarkingNeededToSubmit && publishes(a.Level):
			problems.Add("marking", fmt.Sprintf(publishMarkingMessage, midSentence(levelWords[a.Level])))
		case a.Marking == MarkingNeededToPublish && !publishes(a.Level):
			problems.Add("marking", fmt.Sprintf(submitMarkingMessage, midSentence(levelWords[a.Level])))
		}
	}
	if !levelKnown {
		return a
	}
	if a.Label != "" && utf8.RuneCountInString(a.Label) <= maxAttributeLabel {
		if other, taken := findNamed(f.levelAttributes(a.Level), a.Label, a.ID, attributeKey); taken {
			problems.Add("label", fmt.Sprintf(attributeLabelTaken, levelWords[a.Level], other.Label))
		}
	}
	moved := before.ID == "" || a.Level != before.Level || a.Section != before.Section
	if moved {
		a.Position = f.lastPosition(a)
	}
	if a.ParentAttributeID != "" && typeKnown {
		parent, valid := f.parentOf(a)
		switch {
		case !a.Type.IsChoice():
			problems.Add("parentAttributeId", onlyChoiceDependsMessage)
		case valid:
		case a.ParentAttributeID != before.ParentAttributeID || parent.ID == "":
			problems.Add("parentAttributeId", fmt.Sprintf(chooseParentMessage, levelWords[a.Level]))
		default: // the parent stays, but the attribute moved above it
			problems.Add("section", fmt.Sprintf(stayBelowMessage, a.Label, parent.Label))
		}
	}
	if moved && before.ID != "" {
		for _, d := range f.levelAttributes(a.Level) {
			if d.ParentAttributeID == a.ID && !f.isBefore(a, d) {
				problems.Add("section", fmt.Sprintf(stayBelowMessage, d.Label, a.Label))
			}
		}
	}
	if update.ConditionOptionIDs != nil && len(a.Condition) > 0 && !sameSet(a.Condition, before.Condition) {
		f.checkCondition(a.Condition, problems)
	}
	return a
}
