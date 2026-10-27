package settings

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

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
