package settings

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

func (s *Service) ListAttributes(ctx context.Context, _ actor.Actor, level *Level) ([]ShellAttributeView, error) {
	var views []ShellAttributeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		views, err = s.attributeList(ctx, level)
		return err
	})
	return views, err
}

func (s *Service) attributeList(ctx context.Context, level *Level) ([]ShellAttributeView, error) {
	f, err := s.loadForm(ctx, "")
	if err != nil {
		return nil, err
	}
	u, err := s.usage(ctx)
	if err != nil {
		return nil, err
	}
	levels := AllLevels()
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

func (s *Service) GetAttribute(ctx context.Context, _ actor.Actor, id string) (ShellAttributeView, error) {
	var view ShellAttributeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		view, err = s.attributeViewOf(ctx, id)
		return err
	})
	return view, err
}

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
		if _, locked := lockOf(locks, field); !locked {
			locks = append(locks, Lock{Field: field, Reason: reason})
		}
	}
	if a.Role == RoleBusinessUnit {
		for _, field := range businessUnitLocks() {
			add(field, businessUnitLock)
		}
		return locks
	}
	if a.Role != "" {
		for _, field := range []string{fieldType, fieldLevel, actionDelete} {
			add(field, roleLock)
		}
	}
	if u.valuesByAttribute[a.ID] != nil {
		add(fieldType, typeLockedMessage)
		add(fieldLevel, levelLockedMessage)
	}
	dependents := u.dependents[a.ID]
	if len(dependents) > 0 {
		add(fieldType, fmt.Sprintf(parentTypeLock, dependents[0].record.Label))
	}
	if len(dependents) > 0 || a.ParentAttributeID != "" {
		add(fieldLevel, levelDependencyLock)
	}
	if reason, locked := statusLock(a, u); locked {
		add(fieldActive, reason)
	}
	return locks
}

// businessUnitLocks are the fields and actions the Business unit attribute never allows, because the
// conditions of other attributes depend on it (FR-023, D-055).
func businessUnitLocks() []string {
	return []string{fieldType, fieldLevel, fieldSection, fieldPosition, fieldCondition, fieldParent, fieldActive, actionDelete}
}

func statusLock(a Attribute, u *usageIndex) (string, bool) {
	if !a.Active {
		parent, found := u.attributes[a.ParentAttributeID]
		if found && !parent.Active {
			return fmt.Sprintf(reactivateParentLock, parent.Label, a.Label), true
		}
		return "", false
	}
	for _, d := range u.dependents[a.ID] {
		if !d.inBin && d.record.Active {
			return fmt.Sprintf(dependentActiveLock, d.record.Label, a.Label), true
		}
	}
	return "", false
}

func lockOf(locks []Lock, field string) (string, bool) {
	for _, l := range locks {
		if l.Field == field {
			return l.Reason, true
		}
	}
	return "", false
}
