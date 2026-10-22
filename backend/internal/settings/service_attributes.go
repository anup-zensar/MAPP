package settings

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// The limits of a Shell attribute and its options (data model §4, §5).
const (
	maxAttributeLabel = 80
	maxHelpText       = 300
	maxSectionName    = 60
	maxOptionLabel    = 120
	maxOptions        = 5000
	maxListedLabels   = 5 // option labels an impact warning names
)

// Messages of the plan's "User-visible messages" table about creating and changing Shell attributes, and
// about what a change affects.
const (
	enterLabelMessage        = "Enter a label."
	attributeLabelTaken      = "The %s form already has a Shell attribute called '%s'."
	chooseLevelMessage       = "Choose a level."
	chooseTypeMessage        = "Choose a type."
	chooseMarkingMessage     = "Choose a marking."
	publishMarkingMessage    = "At %s level the marking is Optional or Needed to publish."
	submitMarkingMessage     = "At %s level the marking is Optional or Needed to submit."
	yesNoMarkingMessage      = "Yes/no attributes are always optional — for a needed yes-or-no answer, use a single choice with the options Yes and No."
	splitLevelMessage        = "Multiple choice with % split is only for activities and line items, which carry cost."
	addOptionMessage         = "Add at least one option."
	appearsTwiceMessage      = "'%s' appears twice."
	optionLabelMessage       = "Enter a label for every option."
	tooManyOptionsMessage    = "An attribute can have at most 5,000 options."
	onlyChoiceOptionsMessage = "Only choice attributes have options."
	onlyChoiceDependsMessage = "Only choice attributes can depend on another attribute."
	chooseParentMessage      = "Choose a single or multiple choice attribute that comes before this one on the %s form."
	linkOptionMessage        = "Link '%s' to at least one %s option."
	linkParentOnlyMessage    = "'%s' can only be linked to options of %s."
	inactiveLinkMessage      = "'%s' is inactive — link active %s options only."
	conditionOptionsMessage  = "Choose business units for the condition from the Business unit options."
	noBusinessUnitMessage    = "There is no Business unit attribute yet, so conditions can't be set."
	stayBelowMessage         = "%s must stay below %s."
	withOptionsDetails       = "with %s"
	removedOptionsDetails    = "Removed its %s: %s."
	incompleteMessage        = "%s %s no value yet and will show as incomplete. %s"
	activitiesKeepStatus     = "Submitted and approved activities keep their status."
	plansKeepStatus          = "Published plans keep their status."
	optionsRemovedMessage    = "Its %d options will be removed: %s."
	optionRemovedMessage     = "Its option '%s' will be removed."
)

// Messages of the plan's "User-visible messages" table about Shell attributes: not found and locks.
const (
	attributeGoneMessage = "This Shell attribute no longer exists — someone may have deleted it."
	typeLockedMessage    = "Type can't change once records use this attribute — create a new attribute instead."
	levelLockedMessage   = "Level can't change once records use this attribute — create a new attribute instead."
	parentTypeLock       = "%s depends on it, so its type can't change."
	levelDependencyLock  = "Level can't change while it depends on, or is depended on by, another attribute."
	businessUnitLock     = "Business unit can't be deleted, set inactive, moved or given another type, because other " +
		"attributes are shown or hidden by it."
	dependentActiveLock  = "Set '%s' inactive first — it depends on '%s'."
	reactivateParentLock = "Reactivate '%s' first — '%s' depends on it."
)

// businessUnitLocks are the fields and actions the Business unit attribute never allows, because the
// conditions of other attributes depend on it (FR-023, D-055).
var businessUnitLocks = []string{"type", "level", "section", "position", "condition", "parentAttributeId", "active", "delete"}

// ListAttributes returns the Shell attributes of level, or of every level top-down when level is nil, in
// display order (FR-002, FR-019), each with its options counted, its usage and its locks. Every signed-in
// person may see them (FR-008).
func (s *Service) ListAttributes(ctx context.Context, a actor.Actor, level *Level) ([]ShellAttributeView, error) {
	var views []ShellAttributeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		views, err = s.attributeList(ctx, level)
		return err
	})
	return views, err
}

// attributeList returns the attributes of level (every level when nil) as the API lists them, inside the
// caller's transaction.
func (s *Service) attributeList(ctx context.Context, level *Level) ([]ShellAttributeView, error) {
	f, err := s.loadForm(ctx, "")
	if err != nil {
		return nil, err
	}
	u, err := s.usage(ctx)
	if err != nil {
		return nil, err
	}
	levels := Levels
	if level != nil {
		levels = []Level{*level}
	}
	all := make([]Attribute, 0, len(f.byID))
	for _, attribute := range f.byID {
		all = append(all, attribute)
	}
	views := []ShellAttributeView{}
	for _, l := range levels {
		for _, attribute := range ofLevel(all, l) {
			views = append(views, f.attributeView(attribute, u))
		}
	}
	return views, nil
}

// GetAttribute returns the Shell attribute id with its usage and locks; its options are listed separately
// (FR-020).
func (s *Service) GetAttribute(ctx context.Context, a actor.Actor, id string) (ShellAttributeView, error) {
	var view ShellAttributeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		view, err = s.attributeViewOf(ctx, id)
		return err
	})
	return view, err
}

// attributeViewOf reads the attribute id as the API answers it, or answers not found.
func (s *Service) attributeViewOf(ctx context.Context, id string) (ShellAttributeView, error) {
	f, err := s.loadForm(ctx, "")
	if err != nil {
		return ShellAttributeView{}, err
	}
	attribute, found := f.byID[id]
	if !found {
		return ShellAttributeView{}, errs.NotFound(attributeGoneMessage)
	}
	u, err := s.usage(ctx)
	if err != nil {
		return ShellAttributeView{}, err
	}
	return f.attributeView(attribute, u), nil
}

// storedAttribute reads the attribute id, or answers not found.
func (s *Service) storedAttribute(ctx context.Context, id string) (Attribute, error) {
	attribute, found, err := s.repo.GetAttribute(ctx, id)
	if err != nil {
		return Attribute{}, err
	}
	if !found {
		return Attribute{}, errs.NotFound(attributeGoneMessage)
	}
	return attribute, nil
}

// attributeView returns the attribute a as the API answers it.
func (f formData) attributeView(a Attribute, u *usageIndex) ShellAttributeView {
	var parent *SettingRef
	if p, found := f.byID[a.ParentAttributeID]; found {
		parent = &SettingRef{ID: p.ID, Name: p.Label, Active: p.Active}
	}
	condition := []SettingRef{}
	for _, id := range a.Condition {
		o := f.optionByID[id]
		condition = append(condition, SettingRef{ID: id, Name: o.Label, Active: o.Active})
	}
	view := ShellAttributeView{
		ID: a.ID, Label: a.Label, HelpText: a.HelpText, Level: a.Level, Type: a.Type, Marking: a.Marking,
		Section: a.Section, Position: a.Position, ParentAttribute: parent, Condition: condition,
		BusinessUnit: a.Role == RoleBusinessUnit, Active: a.Active, Usage: u.attribute(a), Locks: attributeLocks(a, u),
		Meta: a.Meta,
	}
	for _, o := range f.options[a.ID] {
		view.OptionCount++
		if o.Active {
			view.ActiveOptionCount++
		}
	}
	return view
}

// attributeLocks returns what cannot change about the attribute a, and why (data model §4.1): the Business
// unit attribute's type, level, section, place, condition, parent, status and delete; the type and level
// once a record has a value (also a record in the recycle bin); the type of a parent and the level of an
// attribute in a dependency; setting inactive a parent of active dependents, and reactivating a dependent
// of an inactive parent.
func attributeLocks(a Attribute, u *usageIndex) []Lock {
	locks := []Lock{}
	add := func(field, reason string) {
		for _, l := range locks {
			if l.Field == field {
				return // one reason per field: the first that applies
			}
		}
		locks = append(locks, Lock{Field: field, Reason: reason})
	}
	if a.Role == RoleBusinessUnit {
		for _, field := range businessUnitLocks {
			add(field, businessUnitLock)
		}
		return locks
	}
	if u.valuesByAttribute[a.ID] != nil {
		add("type", typeLockedMessage)
		add("level", levelLockedMessage)
	}
	dependents := u.dependents[a.ID]
	if len(dependents) > 0 {
		add("type", fmt.Sprintf(parentTypeLock, dependents[0].record.Label))
	}
	if len(dependents) > 0 || a.ParentAttributeID != "" {
		add("level", levelDependencyLock)
	}
	if a.Active {
		for _, d := range dependents {
			if !d.inBin && d.record.Active {
				add("active", fmt.Sprintf(dependentActiveLock, d.record.Label, a.Label))
			}
		}
	} else if parent, found := u.attributes[a.ParentAttributeID]; found && !parent.Active {
		add("active", fmt.Sprintf(reactivateParentLock, parent.Label, a.Label))
	}
	return locks
}

// lockOf returns the reason of the lock on field among locks, and false when the field is not locked.
func lockOf(locks []Lock, field string) (string, bool) {
	for _, l := range locks {
		if l.Field == field {
			return l.Reason, true
		}
	}
	return "", false
}

// CreateAttribute creates an active Shell attribute, last in its section (FR-016 – FR-019, FR-022, FR-023,
// US2-1): a label unique within its level, its help text, its level, type and marking combined as data
// model §4 allows, its section, the attribute it depends on and its condition, and — for a choice type —
// its options, each linked to options of the parent when it depends on another attribute. Every problem is
// reported at once (US2-2). One "created" entry says how many options it has.
func (s *Service) CreateAttribute(ctx context.Context, a actor.Actor, input ShellAttributeInput) (ShellAttributeView, error) {
	condition := input.ConditionOptionIDs
	if condition == nil {
		condition = []string{}
	}
	all := ShellAttributeUpdate{
		Label: &input.Label, HelpText: &input.HelpText, Level: &input.Level, Type: &input.Type, Marking: &input.Marking,
		Section: &input.Section, ParentAttributeID: &input.ParentAttributeID, ConditionOptionIDs: &condition,
	}
	var view ShellAttributeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, err := s.loadForm(ctx, "")
		if err != nil {
			return err
		}
		var problems errs.Problems
		attribute := f.applyAttribute(Attribute{Condition: []string{}}, all, nil, &problems)
		options := f.newOptions(attribute, input.Options, &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		if attribute.ID, err = s.ids.Next(ctx, ids.PrefixShellAttribute); err != nil {
			return err
		}
		attribute.Active, attribute.Meta = true, record.NewMeta(a, s.now())
		if err := s.repo.PutAttribute(ctx, attribute); err != nil {
			return err
		}
		for i, o := range options {
			if o.ID, err = s.ids.Next(ctx, ids.PrefixShellOption); err != nil {
				return err
			}
			o.AttributeID, o.Position, o.Active, o.Meta = attribute.ID, i+1, true, record.NewMeta(a, s.now())
			if err := s.repo.PutOption(ctx, o); err != nil {
				return err
			}
		}
		details := ""
		if len(options) > 0 {
			details = fmt.Sprintf(withOptionsDetails, plural(len(options), "option", "options"))
		}
		if err := s.logCreated(ctx, a, attributeKind, attribute.ID, attribute.Label, details); err != nil {
			return err
		}
		view, err = s.attributeViewOf(ctx, attribute.ID)
		return err
	})
	return view, err
}

// UpdateAttribute changes the fields of the Shell attribute id that update sends (FR-016 – FR-023, US2-4),
// from the version the person started at (FR-007). Locked fields cannot change (data model §4.1); a new
// section or level puts it last in its section; a type without options removes its options permanently
// (D-049, said in the entry's details); a new parent needs optionLinks for every active option, and no
// parent removes the links. A save that changes nothing writes no version and no log entry; otherwise one
// "changed" entry on the attribute and one on each option whose links changed.
func (s *Service) UpdateAttribute(ctx context.Context, a actor.Actor, id string, update ShellAttributeUpdate) (ShellAttributeView, error) {
	var view ShellAttributeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, stored, u, err := s.attributeToChange(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		checkVersion(&problems, update.Version)
		change := f.planAttributeChange(stored, update, u, &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, attributeNoun); err != nil {
			return err
		}
		changes, err := history.Diff(stored, change.attribute)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			details := ""
			if len(change.removed) > 0 {
				labels := make([]string, 0, len(change.removed))
				for _, o := range change.removed {
					labels = append(labels, o.Label)
					if err := s.repo.DeleteOption(ctx, o.ID); err != nil {
						return err
					}
				}
				details = fmt.Sprintf(removedOptionsDetails, plural(len(labels), "option", "options"), strings.Join(labels, ", "))
			}
			changed := change.attribute
			changed.Touch(a, s.now())
			if err := s.repo.PutAttribute(ctx, changed); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, attributeKind, changed.ID, changed.Label, changes, details); err != nil {
				return err
			}
		}
		for _, o := range change.linked {
			linkChanges, err := history.Diff(f.optionByID[o.ID], o)
			if err != nil {
				return err
			}
			o.Touch(a, s.now())
			if err := s.repo.PutOption(ctx, o); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, optionKind, o.ID, o.Label, linkChanges, ""); err != nil {
				return err
			}
		}
		view, err = s.attributeViewOf(ctx, id)
		return err
	})
	return view, err
}

// AttributeImpact says, before a change of the Shell attribute id is saved, what it would affect (FR-021,
// US2-5, D-044): a marking that becomes needed makes the live records of its level without a value
// incomplete — counted as MissingAttributes counts, so records where it is hidden, or a dependent without
// options for the chosen parent (D-034), are not; a type without options removes its options. The body is
// checked as UpdateAttribute checks it, except the version; nothing is saved.
func (s *Service) AttributeImpact(ctx context.Context, a actor.Actor, id string, update ShellAttributeUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, stored, u, err := s.attributeToChange(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		change := f.planAttributeChange(stored, update, u, &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		changed := change.attribute
		if n := f.becomeIncomplete(stored, changed, u.refs); n > 0 {
			nouns, verb, keep := levelNouns[changed.Level], "have", plansKeepStatus
			if n == 1 {
				verb = "has"
			}
			if !publishes(changed.Level) {
				keep = activitiesKeepStatus
			}
			impact.Warnings = append(impact.Warnings, ImpactWarning{
				Kind: ImpactBecomesIncomplete, Message: fmt.Sprintf(incompleteMessage, plural(n, nouns[0], nouns[1]), verb, keep), Count: n,
			})
		}
		if n := len(change.removed); n > 0 {
			message := fmt.Sprintf(optionRemovedMessage, change.removed[0].Label)
			if n > 1 {
				labels := []string{}
				for _, o := range change.removed[:min(n, maxListedLabels)] {
					labels = append(labels, o.Label)
				}
				listed := strings.Join(labels, ", ")
				if n > maxListedLabels {
					listed += fmt.Sprintf(" and %d more", n-maxListedLabels)
				}
				message = fmt.Sprintf(optionsRemovedMessage, n, listed)
			}
			impact.Warnings = append(impact.Warnings, ImpactWarning{Kind: ImpactOptionsRemoved, Message: message, Count: n})
		}
		return nil
	})
	return impact, err
}

// SetAttributeActive sets the Shell attribute id inactive or reactivates it (FR-004, FR-023, US2-7): forms
// stop offering an inactive attribute, and values already on records stay. The Business unit attribute,
// an attribute that active attributes depend on, and a dependent of an inactive attribute (reactivating)
// are refused with the lock's reason. The same status again changes nothing.
func (s *Service) SetAttributeActive(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (ShellAttributeView, error) {
	active, version, err := checkStatus(update)
	if err != nil {
		return ShellAttributeView{}, err
	}
	var view ShellAttributeView
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		_, stored, u, err := s.attributeToChange(ctx, id)
		if err != nil {
			return err
		}
		if err := stored.CheckVersion(version, attributeNoun); err != nil {
			return err
		}
		if stored.Active != active {
			if reason, locked := lockOf(attributeLocks(stored, u), "active"); locked {
				return errs.Validation(errs.FieldError{Field: "active", Message: reason})
			}
			stored.Active = active
			stored.Touch(a, s.now())
			if err := s.repo.PutAttribute(ctx, stored); err != nil {
				return err
			}
			if err := s.logStatus(ctx, a, attributeKind, stored.ID, stored.Label, active); err != nil {
				return err
			}
		}
		view, err = s.attributeViewOf(ctx, id)
		return err
	})
	return view, err
}

// DeleteAttribute moves the Shell attribute id to the recycle bin with its options, so that it can be
// undone (FR-004, US2-7). The Business unit attribute is refused with its lock's reason, and an attribute
// in use — values on records (also in the recycle bin), dependents, conditions — with in_use: set it
// inactive instead.
func (s *Service) DeleteAttribute(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, stored, u, err := s.attributeToChange(ctx, id)
		if err != nil {
			return err
		}
		if reason, locked := lockOf(attributeLocks(stored, u), "delete"); locked {
			return errs.InUse(reason)
		}
		if usage := u.attribute(stored); usage.InUse {
			return inUseError(stored.Label, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, attributeKind, id)
		return err
	})
	return entry, err
}

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

// newOptions checks the options of the new attribute a (FR-018, FR-022) and returns them in the order
// given, without ids: a choice type has 1 to 5,000, each with a label of at most 120 characters that no
// other one has, ignoring letter case, and — when a depends on another attribute — linked to at least one
// active option of it; other types have none.
func (f formData) newOptions(a Attribute, inputs []NewOption, problems *errs.Problems) []Option {
	switch {
	case a.Type == "":
		return nil // no type chosen: its options cannot be checked yet
	case !a.IsChoice():
		if len(inputs) > 0 {
			problems.Add("options", onlyChoiceOptionsMessage)
		}
		return nil
	case len(inputs) == 0:
		problems.Add("options", addOptionMessage)
		return nil
	case len(inputs) > maxOptions:
		problems.Add("options", tooManyOptionsMessage)
		return nil
	}
	parent, dependent := f.parentOf(a)
	options := make([]Option, 0, len(inputs))
	var labels, twice []string
	for i, in := range inputs {
		field := fmt.Sprintf("options[%d]", i)
		label := strings.TrimSpace(in.Label)
		switch first := firstNamed(labels, label); {
		case label == "":
			problems.Add(field+".label", optionLabelMessage)
		case utf8.RuneCountInString(label) > maxOptionLabel:
			problems.Add(field+".label", tooLongMessage(maxOptionLabel))
		case first == "":
			labels = append(labels, label)
		case firstNamed(twice, first) == "":
			problems.Add(field+".label", fmt.Sprintf(appearsTwiceMessage, first))
			twice = append(twice, first)
		}
		links := []string{}
		if dependent {
			links = f.checkLinks(problems, field+".parentOptionIds", label, parent, in.ParentOptionIDs, nil)
		}
		options = append(options, Option{Label: label, ParentOptionIDs: links})
	}
	return options
}

// checkLinks checks the parent options that the option label of a dependent attribute is linked to
// (FR-022): at least one, each an option of parent, and each active unless it was linked already
// (previous). It returns them once each, in the order given.
func (f formData) checkLinks(problems *errs.Problems, field, label string, parent Attribute, links, previous []string) []string {
	links = unique(links)
	parentName := midSentence(parent.Label)
	if len(links) == 0 {
		problems.Add(field, fmt.Sprintf(linkOptionMessage, label, parentName))
		return links
	}
	for _, id := range links {
		o, found := f.optionByID[id]
		switch {
		case !found || o.AttributeID != parent.ID:
			problems.Add(field, fmt.Sprintf(linkParentOnlyMessage, label, parentName))
			return links
		case !o.Active && !slices.Contains(previous, id):
			problems.Add(field, fmt.Sprintf(inactiveLinkMessage, o.Label, parentName))
		}
	}
	return links
}

// checkCondition adds the problem of a condition whose options are not all options of the Business unit
// attribute (FR-023).
func (f formData) checkCondition(condition []string, problems *errs.Problems) {
	if f.businessUnit == nil {
		problems.Add("conditionOptionIds", noBusinessUnitMessage)
		return
	}
	for _, id := range condition {
		if o, found := f.optionByID[id]; !found || o.AttributeID != f.businessUnit.ID {
			problems.Add("conditionOptionIds", conditionOptionsMessage)
			return
		}
	}
}

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

// becomeIncomplete counts the live records of the attribute's level that changing before into after makes
// incomplete (FR-021): when its marking goes from optional to needed, the records that MissingAttributes
// would list it for — not those where it is hidden by its condition, nor a dependent without options for
// the chosen parent (D-034).
func (f formData) becomeIncomplete(before, after Attribute, refs []Reference) int {
	if before.Marking != MarkingOptional || after.Marking == MarkingOptional || !after.Active {
		return 0
	}
	form := f
	form.level = after.Level
	form.attributes = displayOrder(append(filter(f.levelAttributes(after.Level), func(a Attribute) bool { return a.ID != after.ID }), after))
	planValues := map[string]AttributeValues{}
	for _, ref := range refs {
		if ref.Level == LevelPlan {
			planValues[ref.PlanID] = ref.Attributes
		}
	}
	field, n := "attributes."+after.ID, 0
	for _, ref := range refs {
		if ref.InBin || ref.Level != after.Level {
			continue
		}
		for _, m := range form.missing(ref.Attributes, f.businessUnitOf(planValues[ref.PlanID])) {
			if m.Field == field {
				n++
				break
			}
		}
	}
	return n
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
