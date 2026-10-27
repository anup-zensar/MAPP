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
		if err = problems.Err(); err != nil {
			return err
		}
		if attribute.ID, err = s.ids.Next(ctx, ids.PrefixShellAttribute); err != nil {
			return err
		}
		attribute.Active, attribute.Meta = true, record.NewMeta(a, s.now())
		if err = s.repo.PutAttribute(ctx, attribute); err != nil {
			return err
		}
		for i := range options {
			options[i].AttributeID, options[i].Active = attribute.ID, true
		}
		if err = s.addOptions(ctx, a, options, 0); err != nil {
			return err
		}
		if err = s.logCreated(ctx, a, attributeKind, attribute.ID, attribute.Label, withOptions(options)); err != nil {
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
		if err = problems.Err(); err != nil {
			return err
		}
		if err = stored.CheckVersion(*update.Version, attributeNoun); err != nil {
			return err
		}
		changes, err := history.Diff(stored, change.attribute)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			if err = s.saveAttributeChange(ctx, a, change, changes); err != nil {
				return err
			}
		}
		if err = s.saveLinkedOptions(ctx, a, f, change.linked); err != nil {
			return err
		}
		view, err = s.attributeViewOf(ctx, id)
		return err
	})
	return view, err
}

// saveAttributeChange first removes the options a type without options can't keep (D-049).
func (s *Service) saveAttributeChange(ctx context.Context, a actor.Actor, change attributeChange, changes []history.Change) error {
	details, err := s.removeOptions(ctx, change.removed)
	if err != nil {
		return err
	}
	changed := change.attribute
	changed.Touch(a, s.now())
	if err = s.repo.PutAttribute(ctx, changed); err != nil {
		return err
	}
	return s.logChanged(ctx, a, attributeKind, changed.ID, changed.Label, changes, details)
}

// removeOptions returns what the log entry says of the removed options: "" when there are none.
func (s *Service) removeOptions(ctx context.Context, options []Option) (string, error) {
	if len(options) == 0 {
		return "", nil
	}
	labels, removed := make([]string, 0, len(options)), make([]string, 0, len(options))
	for _, o := range options {
		labels, removed = append(labels, o.Label), append(removed, o.ID)
	}
	if err := s.repo.DeleteOptions(ctx, removed); err != nil {
		return "", err
	}
	return fmt.Sprintf(removedOptionsDetails, plural(len(labels), "option", "options"), strings.Join(labels, ", ")), nil
}

func (s *Service) saveLinkedOptions(ctx context.Context, a actor.Actor, f formData, linked []Option) error {
	for _, o := range linked {
		linkChanges, err := history.Diff(f.optionByID[o.ID], o)
		if err != nil {
			return err
		}
		o.Touch(a, s.now())
		if err = s.repo.PutOption(ctx, o); err != nil {
			return err
		}
		if err = s.logChanged(ctx, a, optionKind, o.ID, o.Label, linkChanges, ""); err != nil {
			return err
		}
	}
	return nil
}

// withOptions is "with 3 options", or "" for an attribute without options.
func withOptions(options []Option) string {
	if len(options) == 0 {
		return ""
	}
	return fmt.Sprintf(withOptionsDetails, plural(len(options), "option", "options"))
}
