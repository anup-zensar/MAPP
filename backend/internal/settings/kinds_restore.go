package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// Messages of the plan's "User-visible messages" table about bringing records back (Restore (API), Log
// details).
const (
	undoNameTakenMessage     = "Can't undo — there is now another %s called '%s'. Restore it from the recycle bin instead."
	renamedNote              = "Renamed to '%s' because the name was taken."
	undoCodeTakenMessage     = "Can't undo — there is now another currency with the code '%s'. Restore it from the recycle bin instead."
	restoreCodeTakenMessage  = "Can't restore — there is now another currency with the code '%s'."
	noOptionsAnyMoreMessage  = "Can't restore — '%s' no longer has options."
	undoItemCodeTakenMessage = "Can't undo — the code '%s' is now used by '%s'. Restore it from the recycle bin instead."
	itemCodeRemovedNote      = "Code '%s' removed because another item now uses it."
)

// prepareModelRestore applies the name rule of models before one comes back (data model §11).
func (s *Service) prepareModelRestore(ctx context.Context, rc history.RestoreContext, snap *history.Snapshot) ([]string, error) {
	var m Model
	if err := json.Unmarshal(snap.Data, &m); err != nil {
		return nil, err
	}
	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	name, notes, err := restoreName(rc, modelNoun, m.Name, func(name string) (string, bool) {
		other, taken := findNamed(models, name, m.ID, func(m Model) (string, string) { return m.ID, m.Name })
		return other.Name, taken
	})
	if err != nil || name == m.Name {
		return nil, err
	}
	m.Name = name
	m.Touch(rc.Actor, rc.Now)
	return notes, resnapshot(snap, m, m.Name)
}

// prepareActivityTypeRestore applies the name rule of activity types before one comes back.
func (s *Service) prepareActivityTypeRestore(ctx context.Context, rc history.RestoreContext, snap *history.Snapshot) ([]string, error) {
	var t ActivityType
	if err := json.Unmarshal(snap.Data, &t); err != nil {
		return nil, err
	}
	types, err := s.repo.ListActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	name, notes, err := restoreName(rc, activityTypeNoun, t.Name, func(name string) (string, bool) {
		other, taken := findNamed(types, name, t.ID, func(t ActivityType) (string, string) { return t.ID, t.Name })
		return other.Name, taken
	})
	if err != nil || name == t.Name {
		return nil, err
	}
	t.Name = name
	t.Touch(rc.Actor, rc.Now)
	return notes, resnapshot(snap, t, t.Name)
}

// prepareAttributeRestore applies the label rule of Shell attributes, within their level, before one comes
// back. It keeps its position; ties sort by id (data model §4.2).
func (s *Service) prepareAttributeRestore(ctx context.Context, rc history.RestoreContext, snap *history.Snapshot) ([]string, error) {
	var a Attribute
	if err := json.Unmarshal(snap.Data, &a); err != nil {
		return nil, err
	}
	attributes, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	sameLevel := filter(attributes, func(other Attribute) bool { return other.Level == a.Level })
	label, notes, err := restoreName(rc, attributeNoun, a.Label, func(label string) (string, bool) {
		other, taken := findNamed(sameLevel, label, a.ID, func(a Attribute) (string, string) { return a.ID, a.Label })
		return other.Label, taken
	})
	if err != nil || label == a.Label {
		return nil, err
	}
	a.Label = label
	a.Touch(rc.Actor, rc.Now)
	return notes, resnapshot(snap, a, a.Label)
}

// prepareOptionRestore refuses an option whose attribute no longer has options (its type changed), and
// applies the label rule within its attribute.
func (s *Service) prepareOptionRestore(ctx context.Context, rc history.RestoreContext, snap *history.Snapshot) ([]string, error) {
	var o Option
	if err := json.Unmarshal(snap.Data, &o); err != nil {
		return nil, err
	}
	attribute, found, err := s.repo.GetAttribute(ctx, o.AttributeID)
	if err != nil {
		return nil, err
	}
	if found && !attribute.IsChoice() {
		return nil, errs.CannotRestore(fmt.Sprintf(noOptionsAnyMoreMessage, attribute.Label))
	}
	options, err := s.repo.OptionsOf(ctx, o.AttributeID)
	if err != nil {
		return nil, err
	}
	label, notes, err := restoreName(rc, optionNoun, o.Label, func(label string) (string, bool) {
		other, taken := findNamed(options, label, o.ID, func(o Option) (string, string) { return o.ID, o.Label })
		return other.Label, taken
	})
	if err != nil || label == o.Label {
		return nil, err
	}
	o.Label = label
	o.Touch(rc.Actor, rc.Now)
	return notes, resnapshot(snap, o, o.Label)
}

// prepareCurrencyRestore refuses a currency whose code another currency has taken meanwhile: a code cannot
// take a suffix (D-054). Its name follows the name rule.
func (s *Service) prepareCurrencyRestore(ctx context.Context, rc history.RestoreContext, snap *history.Snapshot) ([]string, error) {
	var c Currency
	if err := json.Unmarshal(snap.Data, &c); err != nil {
		return nil, err
	}
	_, codeTaken, err := s.repo.GetCurrency(ctx, c.Code)
	if err != nil {
		return nil, err
	}
	if codeTaken && rc.Undo {
		return nil, errs.NameTaken(fmt.Sprintf(undoCodeTakenMessage, c.Code))
	}
	if codeTaken {
		return nil, errs.CannotRestore(fmt.Sprintf(restoreCodeTakenMessage, c.Code))
	}
	currencies, err := s.repo.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	name, notes, err := restoreName(rc, currencyNoun, c.Name, func(name string) (string, bool) {
		other, taken := findNamed(currencies, name, c.Code, func(c Currency) (string, string) { return c.Code, c.Name })
		return other.Name, taken
	})
	if err != nil || name == c.Name {
		return nil, err
	}
	c.Name = name
	c.Touch(rc.Actor, rc.Now)
	return notes, resnapshot(snap, c, c.Code)
}

// prepareItemRestore applies the name rule of finance budget items within their budget, and the code rule
// within their year: an Undo is refused when another item now has the code, a Restore brings the item back
// without it (D-054).
func (s *Service) prepareItemRestore(ctx context.Context, rc history.RestoreContext, snap *history.Snapshot) ([]string, error) {
	var item FinanceBudgetItem
	if err := json.Unmarshal(snap.Data, &item); err != nil {
		return nil, err
	}
	siblings, err := s.repo.ItemsOf(ctx, item.BudgetID)
	if err != nil {
		return nil, err
	}
	name, notes, err := restoreName(rc, itemNoun, item.Name, func(name string) (string, bool) {
		other, taken := findNamed(siblings, name, item.ID, func(i FinanceBudgetItem) (string, string) { return i.ID, i.Name })
		return other.Name, taken
	})
	if err != nil {
		return nil, err
	}
	changed := name != item.Name
	item.Name = name
	if item.Code != "" {
		other, taken, err := s.itemWithCode(ctx, item.BudgetID, item.Code, item.ID)
		if err != nil {
			return nil, err
		}
		if taken && rc.Undo {
			return nil, errs.NameTaken(fmt.Sprintf(undoItemCodeTakenMessage, item.Code, other.Name))
		}
		if taken {
			notes = append(notes, fmt.Sprintf(itemCodeRemovedNote, item.Code))
			item.Code, changed = "", true
		}
	}
	if !changed {
		return nil, nil
	}
	item.Touch(rc.Actor, rc.Now)
	return notes, resnapshot(snap, item, item.Name)
}

// resnapshot stores the changed record back into s, with its new name, after a PrepareRestore changed it.
func resnapshot(s *history.Snapshot, record any, name string) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	s.Name, s.Data = name, data
	return nil
}
