package settings

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// The paging of option lists (FR-018, SC-007).
const (
	defaultOptionLimit = 100
	maxOptionLimit     = 500
)

// Messages of the plan's "User-visible messages" table about options.
const (
	optionGoneMessage     = "This option no longer exists — someone may have deleted it."
	limitMessage          = "Choose a number from 1 to 500."
	offsetMessage         = "Choose 0 or more."
	optionStatusMessage   = "Choose active, inactive or all."
	optionTakenMessage    = "'%s' is already an option of %s."
	noOptionsMessage      = "%s has no options — its type is %s."
	lastOptionLockMessage = "'%s' needs at least one option, so '%s' can't be deleted."
	firstOptionMessage    = "'%s' is already first."
	lastOptionMessage     = "'%s' is already last."
)

// CreateOption adds an active option, last in the list, to the choice attribute attributeID (FR-018,
// FR-022): a label of at most 120 characters that no other option of the attribute has, ignoring letter
// case; at most 5,000 options; and, when the attribute depends on another, links to at least one active
// option of it. One "created" entry.
func (s *Service) CreateOption(ctx context.Context, a actor.Actor, attributeID string, input OptionInput) (OptionView, error) {
	var view OptionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, err := s.loadForm(ctx, "")
		if err != nil {
			return err
		}
		attribute, found := f.byID[attributeID]
		if !found {
			return errs.NotFound(attributeGoneMessage)
		}
		if !attribute.IsChoice() {
			return errs.Validation(errs.FieldError{Field: "label", Message: fmt.Sprintf(noOptionsMessage, attribute.Label, typeWords[attribute.Type])})
		}
		var problems errs.Problems
		options := f.options[attribute.ID]
		o := Option{AttributeID: attribute.ID, ParentOptionIDs: []string{}, Position: 1, Active: true}
		o.Label = f.checkOptionLabel(&problems, input.Label, attribute, "")
		if len(options) >= maxOptions {
			problems.Add("label", tooManyOptionsMessage)
		}
		if parent, dependent := f.byID[attribute.ParentAttributeID]; dependent {
			o.ParentOptionIDs = f.checkLinks(&problems, "parentOptionIds", o.Label, parent, input.ParentOptionIDs, nil)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if o.ID, err = s.ids.Next(ctx, ids.PrefixShellOption); err != nil {
			return err
		}
		if len(options) > 0 {
			o.Position = options[len(options)-1].Position + 1
		}
		o.Meta = record.NewMeta(a, s.now())
		if err := s.repo.PutOption(ctx, o); err != nil {
			return err
		}
		if err := s.logCreated(ctx, a, optionKind, o.ID, o.Label, ""); err != nil {
			return err
		}
		view, err = s.optionViewOf(ctx, f, o, attribute)
		return err
	})
	return view, err
}

// UpdateOption renames the option id or changes the parent options it is linked to (FR-018, FR-022,
// US2-4), from the version the person started at (FR-007): a dependent attribute's option keeps at least
// one link, and links newly added must be active. A save that changes nothing writes no version and no log
// entry; otherwise one "changed" entry.
func (s *Service) UpdateOption(ctx context.Context, a actor.Actor, id string, update OptionUpdate) (OptionView, error) {
	var view OptionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, stored, attribute, err := s.optionToChange(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		checkVersion(&problems, update.Version)
		changed := stored
		if update.Label != nil {
			changed.Label = f.checkOptionLabel(&problems, *update.Label, attribute, stored.ID)
		}
		if parent, dependent := f.byID[attribute.ParentAttributeID]; dependent && update.ParentOptionIDs != nil {
			changed.ParentOptionIDs = f.checkLinks(&problems, "parentOptionIds", changed.Label, parent, *update.ParentOptionIDs, stored.ParentOptionIDs)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, optionNoun); err != nil {
			return err
		}
		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.PutOption(ctx, changed); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, optionKind, changed.ID, changed.Label, changes, ""); err != nil {
				return err
			}
			stored = changed
		}
		view, err = s.optionViewOf(ctx, f, stored, attribute)
		return err
	})
	return view, err
}

// SetOptionActive sets the option id inactive or reactivates it (FR-018, FR-004): forms stop offering an
// inactive option, and records that have it keep it. The same status again changes nothing.
func (s *Service) SetOptionActive(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (OptionView, error) {
	active, version, err := checkStatus(update)
	if err != nil {
		return OptionView{}, err
	}
	var view OptionView
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		f, stored, attribute, err := s.optionToChange(ctx, id)
		if err != nil {
			return err
		}
		if err := stored.CheckVersion(version, optionNoun); err != nil {
			return err
		}
		if stored.Active != active {
			stored.Active = active
			stored.Touch(a, s.now())
			if err := s.repo.PutOption(ctx, stored); err != nil {
				return err
			}
			if err := s.logStatus(ctx, a, optionKind, stored.ID, stored.Label, active); err != nil {
				return err
			}
		}
		view, err = s.optionViewOf(ctx, f, stored, attribute)
		return err
	})
	return view, err
}

// MoveOption swaps the option id with its neighbour above or below in its attribute's list (FR-018), from
// the version the person started at. At the ends it is refused on direction. Both options get one
// "changed" entry (field position).
func (s *Service) MoveOption(ctx context.Context, a actor.Actor, id string, in OptionMoveInput) (OptionView, error) {
	var problems errs.Problems
	if in.Direction != MoveUp && in.Direction != MoveDown {
		problems.Add("direction", reloadMessage)
	}
	checkVersion(&problems, in.Version)
	if err := problems.Err(); err != nil {
		return OptionView{}, err
	}
	var view OptionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, stored, attribute, err := s.optionToChange(ctx, id)
		if err != nil {
			return err
		}
		if err := stored.CheckVersion(*in.Version, optionNoun); err != nil {
			return err
		}
		options := f.options[attribute.ID]
		i := slices.IndexFunc(options, func(o Option) bool { return o.ID == id })
		j := i - 1
		if in.Direction == MoveDown {
			j = i + 1
		}
		switch {
		case j < 0:
			return errs.Validation(errs.FieldError{Field: "direction", Message: fmt.Sprintf(firstOptionMessage, stored.Label)})
		case j >= len(options):
			return errs.Validation(errs.FieldError{Field: "direction", Message: fmt.Sprintf(lastOptionMessage, stored.Label)})
		}
		order := slices.Clone(options)
		order[i], order[j] = order[j], order[i]
		for _, changed := range repositionedOptions(options, order) {
			change := history.Change{Field: "position", Old: jsonOf(f.optionByID[changed.ID].Position), New: jsonOf(changed.Position)}
			changed.Touch(a, s.now())
			if err := s.repo.PutOption(ctx, changed); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, optionKind, changed.ID, changed.Label, []history.Change{change}, ""); err != nil {
				return err
			}
			if changed.ID == id {
				stored = changed
			}
		}
		view, err = s.optionViewOf(ctx, f, stored, attribute)
		return err
	})
	return view, err
}

// GetOption returns the option id as it is stored now, with its parent options, usage and locks (D-211):
// the web app reads it after a version conflict. Every signed-in person may read it (FR-008), so a is not
// used yet.
func (s *Service) GetOption(ctx context.Context, a actor.Actor, id string) (OptionView, error) {
	var view OptionView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, stored, attribute, err := s.optionToChange(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.optionViewOf(ctx, f, stored, attribute)
		return err
	})
	return view, err
}

// DeleteOption moves the option id to the recycle bin, so that it can be undone (FR-004, FR-018). The last
// option of a choice attribute is refused with in_use and the reason — which the option's delete lock
// also gives (D-211) — and so is an option in use — values on records (also in the recycle bin), options
// of dependent attributes linked to it, conditions: set it inactive instead.
func (s *Service) DeleteOption(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, stored, attribute, err := s.optionToChange(ctx, id)
		if err != nil {
			return err
		}
		if attribute.IsChoice() && len(f.options[attribute.ID]) <= 1 {
			return errs.InUse(fmt.Sprintf(lastOptionLockMessage, attribute.Label, stored.Label))
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		if usage := u.option(stored, attribute); usage.InUse {
			return inUseError(stored.Label, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, optionKind, id)
		return err
	})
	return entry, err
}

// optionToChange reads every attribute and option, and returns the option id — or answers not found — with
// its attribute, inside the caller's transaction.
func (s *Service) optionToChange(ctx context.Context, id string) (formData, Option, Attribute, error) {
	f, err := s.loadForm(ctx, "")
	if err != nil {
		return formData{}, Option{}, Attribute{}, err
	}
	o, found := f.optionByID[id]
	if !found {
		return formData{}, Option{}, Attribute{}, errs.NotFound(optionGoneMessage)
	}
	return f, o, f.byID[o.AttributeID], nil
}

// optionViewOf returns the option o of the attribute a as the API answers it, reading the usage index.
// f must hold the parent attribute's options.
func (s *Service) optionViewOf(ctx context.Context, f formData, o Option, a Attribute) (OptionView, error) {
	u, err := s.usage(ctx)
	if err != nil {
		return OptionView{}, err
	}
	return f.optionView(o, a, u), nil
}

// checkOptionLabel trims label and adds its problems to problems: empty, longer than 120 characters, or the
// label of another option of the attribute than exceptID, ignoring letter case (FR-018).
func (f formData) checkOptionLabel(problems *errs.Problems, label string, attribute Attribute, exceptID string) string {
	label = strings.TrimSpace(label)
	checkText(problems, "label", label, enterLabelMessage, maxOptionLabel)
	if other, taken := findNamed(f.options[attribute.ID], label, exceptID, func(o Option) (string, string) { return o.ID, o.Label }); taken && label != "" {
		problems.Add("label", fmt.Sprintf(optionTakenMessage, other.Label, attribute.Label))
	}
	return label
}

// repositionedOptions gives the options of order — the options of one attribute, which are in display
// order, in a new order — the positions that make that order, and returns those whose position changes: a
// move swaps two positions, or, when both have the same position, numbers the list 1…n.
func repositionedOptions(options, order []Option) []Option {
	next := slices.Clone(order)
	for k := range next {
		next[k].Position = options[k].Position
	}
	if !slices.EqualFunc(optionOrder(next), next, func(x, y Option) bool { return x.ID == y.ID }) {
		for k := range next {
			next[k].Position = k + 1
		}
	}
	positions := make(map[string]int, len(options))
	for _, o := range options {
		positions[o.ID] = o.Position
	}
	changed := []Option{}
	for _, o := range next {
		if positions[o.ID] != o.Position {
			changed = append(changed, o)
		}
	}
	return changed
}

// The statuses an option list can be filtered by; "" means all.
const (
	statusAll      = "all"
	statusActive   = "active"
	statusInactive = "inactive"
)

// ListOptions returns the options of the attribute attributeID in display order (FR-018, FR-020): those
// whose label contains filter.Q ignoring letter case and surrounding spaces ("0220" finds "CN04000220",
// SC-007), of the status asked, at most filter.Limit (1 to 500) after skipping filter.Offset; the page's
// total counts every match. Each option has its parent options and its own usage (US2-3).
func (s *Service) ListOptions(ctx context.Context, a actor.Actor, attributeID string, filter OptionFilter) (OptionPage, error) {
	var problems errs.Problems
	if filter.Limit < 1 || filter.Limit > maxOptionLimit {
		problems.Add("limit", limitMessage)
	}
	if filter.Offset < 0 {
		problems.Add("offset", offsetMessage)
	}
	if filter.Status != "" && filter.Status != statusAll && filter.Status != statusActive && filter.Status != statusInactive {
		problems.Add("status", optionStatusMessage)
	}
	if err := problems.Err(); err != nil {
		return OptionPage{}, err
	}

	page := OptionPage{Items: []OptionView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.loadForm(ctx, "")
		if err != nil {
			return err
		}
		attribute, found := f.byID[attributeID]
		if !found {
			return errs.NotFound(attributeGoneMessage)
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		search := strings.ToLower(strings.TrimSpace(filter.Q))
		var matches []Option
		for _, o := range f.options[attributeID] {
			if (filter.Status == statusActive && !o.Active) || (filter.Status == statusInactive && o.Active) {
				continue
			}
			if strings.Contains(strings.ToLower(o.Label), search) {
				matches = append(matches, o)
			}
		}
		page.Total = len(matches)
		for _, o := range matches[min(filter.Offset, len(matches)):min(filter.Offset+filter.Limit, len(matches))] {
			page.Items = append(page.Items, f.optionView(o, attribute, u))
		}
		return nil
	})
	return page, err
}

// optionView returns the option o of the attribute a as the API answers it.
func (f formData) optionView(o Option, a Attribute, u *usageIndex) OptionView {
	parents := []SettingRef{}
	for _, id := range o.ParentOptionIDs {
		p := f.optionByID[id]
		parents = append(parents, SettingRef{ID: id, Name: p.Label, Active: p.Active})
	}
	return OptionView{
		ID: o.ID, AttributeID: o.AttributeID, Label: o.Label, Position: o.Position, ParentOptions: parents,
		Active: o.Active, Usage: u.option(o, a), Locks: f.optionLocks(o, a), Meta: o.Meta,
	}
}

// optionLocks says what can't be done with the option o of the attribute a, and why (design system 6.6,
// D-211): the last option of a choice attribute can't be deleted (FR-018), as DeleteOption answers. An
// option that f does not hold yet (one just created) counts as one more option. Never nil.
func (f formData) optionLocks(o Option, a Attribute) []Lock {
	count := len(f.options[a.ID])
	if _, held := f.optionByID[o.ID]; !held {
		count++
	}
	if a.IsChoice() && count <= 1 {
		return []Lock{{Field: "delete", Reason: fmt.Sprintf(lastOptionLockMessage, a.Label, o.Label)}}
	}
	return []Lock{}
}
