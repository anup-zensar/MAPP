package settings

import (
	"context"
	"fmt"
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
