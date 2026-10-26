package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The history kinds of the Settings records (data model §11).
const (
	modelKind        = "model"
	activityTypeKind = "activityType"
	attributeKind    = "shellAttribute"
	optionKind       = "shellAttributeOption"
	currencyKind     = "currency"
	budgetKind       = "financeBudget"
	itemKind         = "financeBudgetItem"
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

// Kinds returns the history kinds of the Settings records for internal/app to register, parents first:
// model, activity type, Shell attribute, option (under its attribute), currency, finance budget and finance
// budget item (under its budget). With them, history logs their changes in words, deletes an attribute
// with its options, and brings records back with the restore rules of data model §11.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{
			Name: modelKind, Singular: modelNoun, Plural: "models",
			Snapshot: s.modelSnapshot, Remove: s.repo.DeleteModel, Put: putRecord(s.repo.PutModel),
			Describe: s.describeModel, PrepareRestore: s.prepareModelRestore,
		},
		{
			Name: activityTypeKind, Singular: activityTypeNoun, Plural: "activity types",
			Snapshot: s.activityTypeSnapshot, Remove: s.repo.DeleteActivityType, Put: putRecord(s.repo.PutActivityType),
			Describe: describeActivityType, PrepareRestore: s.prepareActivityTypeRestore,
		},
		{
			Name: attributeKind, Singular: attributeNoun, Plural: "Shell attributes",
			Snapshot: s.attributeSnapshot, Remove: s.repo.DeleteAttribute, Put: putRecord(s.repo.PutAttribute),
			Describe: s.describeAttribute, PrepareRestore: s.prepareAttributeRestore,
		},
		{
			Name: optionKind, Singular: optionNoun, Plural: "options", Parent: attributeKind,
			Snapshot: s.optionSnapshot, ChildIDs: s.optionIDs, Remove: s.repo.DeleteOption, Put: putRecord(s.repo.PutOption),
			Describe: s.describeOption, PrepareRestore: s.prepareOptionRestore,
		},
		{
			Name: currencyKind, Singular: currencyNoun, Plural: "currencies",
			Snapshot: s.currencySnapshot, Remove: s.repo.DeleteCurrency, Put: putRecord(s.repo.PutCurrency),
			Describe: describeCurrency, PrepareRestore: s.prepareCurrencyRestore,
		},
		{
			// 016: a currency's dated rates, deleted with it or one by one (D-272).
			Name: exchangeRateKind, Singular: exchangeRateNoun, Plural: "exchange rates", Parent: currencyKind,
			Snapshot: s.rateSnapshot, ChildIDs: s.rateIDs, Remove: s.repo.DeleteRate, Put: putRecord(s.repo.PutRate),
			Describe: describeRate, PrepareRestore: s.prepareRateRestore,
		},
		{
			// 016: GSAP rate imports, logged and never deleted.
			Name: rateImportKind, Singular: rateImportNoun, Plural: "rate imports",
			Snapshot: s.rateImportSnapshot, Remove: neverRemoved, Put: putRecord(s.repo.PutRateImport),
		},
		{
			// 016: edited Field help texts, logged and never in the recycle bin.
			Name: fieldHelpKind, Singular: fieldHelpNoun, Plural: "field help texts",
			Snapshot: s.fieldHelpSnapshot, Remove: neverRemoved, Put: putRecord(s.repo.PutFieldHelp),
			Describe: describeFieldHelp,
		},
		{
			Name: budgetKind, Singular: budgetNoun, Plural: "finance budgets",
			Snapshot: s.budgetSnapshot, Remove: s.repo.DeleteFinanceBudget, Put: putRecord(s.repo.PutFinanceBudget),
			Describe: describeBudget,
		},
		{
			Name: itemKind, Singular: itemNoun, Plural: "finance budget items", Parent: budgetKind,
			Snapshot: s.itemSnapshot, ChildIDs: s.itemIDs, Remove: s.repo.DeleteFinanceBudgetItem, Put: putRecord(s.repo.PutFinanceBudgetItem),
			Describe: s.describeItem, PrepareRestore: s.prepareItemRestore,
		},
	}
}

// snapshot returns record as history keeps it: its id, name, parent and stored data.
func snapshot(record any, id, name, parentID string) (history.Snapshot, bool, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: id, Name: name, ParentID: parentID, Data: data}, true, nil
}

// putRecord returns the Put of a kind: it stores the record of a snapshot exactly as the snapshot holds it.
func putRecord[T any](put func(context.Context, T) error) func(context.Context, history.Snapshot) error {
	return func(ctx context.Context, s history.Snapshot) error {
		var record T
		if err := json.Unmarshal(s.Data, &record); err != nil {
			return err
		}
		return put(ctx, record)
	}
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

func (s *Service) modelSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	m, found, err := s.repo.GetModel(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(m, m.ID, m.Name, "")
}

func (s *Service) activityTypeSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	t, found, err := s.repo.GetActivityType(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(t, t.ID, t.Name, "")
}

func (s *Service) attributeSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	a, found, err := s.repo.GetAttribute(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(a, a.ID, a.Label, "")
}

func (s *Service) optionSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	o, found, err := s.repo.GetOption(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(o, o.ID, o.Label, o.AttributeID)
}

// The snapshot name of a currency is its code (data model §11).
func (s *Service) currencySnapshot(ctx context.Context, code string) (history.Snapshot, bool, error) {
	c, found, err := s.repo.GetCurrency(ctx, code)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(c, c.Code, c.Code, "")
}

func (s *Service) budgetSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	b, found, err := s.repo.GetFinanceBudget(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(b, b.ID, b.Name, "")
}

func (s *Service) itemSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	i, found, err := s.repo.GetFinanceBudgetItem(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(i, i.ID, i.Name, i.BudgetID)
}

// optionIDs lists the ids of the options of the attribute attributeID: deleting an attribute takes them
// with it.
func (s *Service) optionIDs(ctx context.Context, attributeID string) ([]string, error) {
	options, err := s.repo.OptionsOf(ctx, attributeID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(options))
	for _, o := range options {
		list = append(list, o.ID)
	}
	return list, nil
}

// itemIDs lists the ids of the items of the finance budget budgetID.
func (s *Service) itemIDs(ctx context.Context, budgetID string) ([]string, error) {
	items, err := s.repo.ItemsOf(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(items))
	for _, i := range items {
		list = append(list, i.ID)
	}
	return list, nil
}

// The words of the attribute enums on screen and in the log (data model §4).
var (
	levelWords = map[Level]string{
		LevelPlan: "Plan", LevelCampaign: "Campaign", LevelInitiative: "Initiative", LevelActivity: "Activity", LevelLineItem: "Line item",
	}
	typeWords = map[AttributeType]string{
		TypeSingleChoice: "Single choice", TypeMultipleChoice: "Multiple choice", TypeMultipleChoiceSplit: "Multiple choice with % split",
		TypeYesNo: "Yes/no", TypeText: "Text", TypeLongText: "Long text", TypeNumber: "Number",
	}
	markingWords = map[Marking]string{
		MarkingOptional: "Optional", MarkingNeededToPublish: "Needed to publish", MarkingNeededToSubmit: "Needed to submit",
	}
)

// The Describe functions give the label of each field and the text of each value in the activity log
// (data model §12). value is the stored JSON value decoded: nil, bool, json.Number, string, []any or
// map[string]any. An empty text means history's default ("Not filled in" for an empty value).

func (s *Service) describeModel(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Name", "", nil
	case "inqToMqlRate":
		return "INQ→MQL rate", percentText(value), nil
	case "mqlToDealRate":
		return "MQL→deal rate", percentText(value), nil
	case "averageDealSize":
		return "Average deal size", moneyText(value), nil
	case "activityTypeIds":
		text, err := s.namesOf(ctx, value, func(ctx context.Context, id string) (string, bool, error) {
			t, found, err := s.repo.GetActivityType(ctx, id)
			return t.Name, found, err
		})
		return "Allowed activity types", text, err
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func describeActivityType(_ context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Name", "", nil
	case "defaultPlannedCost":
		return "Default planned cost", moneyText(value), nil
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func (s *Service) describeAttribute(ctx context.Context, field string, value any) (string, string, error) {
	text, _ := value.(string)
	switch field {
	case "label":
		return "Label", "", nil
	case "helpText":
		return "Help text", "", nil
	case "level":
		return "Level", levelWords[Level(text)], nil
	case "type":
		return "Type", typeWords[AttributeType(text)], nil
	case "marking":
		return "Marking", markingWords[Marking(text)], nil
	case "section":
		return "Section", "", nil
	case "position":
		return "Place on the form", "", nil
	case "parentAttributeId":
		label, err := s.namesOf(ctx, value, s.attributeLabel)
		return "Depends on", label, err
	case "condition":
		if list, _ := value.([]any); len(list) == 0 {
			return "Shown when the business unit includes", "Always", nil
		}
		labels, err := s.namesOf(ctx, value, s.optionLabel)
		return "Shown when the business unit includes", labels, err
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func (s *Service) describeOption(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "label":
		return "Label", "", nil
	case "parentOptionIds":
		labels, err := s.namesOf(ctx, value, s.optionLabel)
		return "Linked to", labels, err
	case "position":
		return "Place in the list", "", nil
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func describeCurrency(_ context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Name", "", nil
	case "symbol":
		return "Symbol", "", nil
	case "rate":
		text, _ := value.(string)
		if rate, err := money.ParseRate(text); err == nil {
			return "Rate (units per 1 USD)", rate.Format(), nil
		}
		return "Rate (units per 1 USD)", "", nil
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func describeBudget(_ context.Context, field string, _ any) (string, string, error) {
	if field == "name" {
		return "Budget name", "", nil
	}
	return "", "", nil
}

func (s *Service) describeItem(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Item name", "", nil
	case "code":
		return "Code", "", nil
	case "budgetId":
		name, err := s.namesOf(ctx, value, func(ctx context.Context, id string) (string, bool, error) {
			b, found, err := s.repo.GetFinanceBudget(ctx, id)
			return b.Name, found, err
		})
		return "Budget", name, err
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

// attributeLabel looks up the label of the attribute id.
func (s *Service) attributeLabel(ctx context.Context, id string) (string, bool, error) {
	a, found, err := s.repo.GetAttribute(ctx, id)
	return a.Label, found, err
}

// optionLabel looks up the label of the option id.
func (s *Service) optionLabel(ctx context.Context, id string) (string, bool, error) {
	o, found, err := s.repo.GetOption(ctx, id)
	return o.Label, found, err
}

// namesOf writes the names of the ids in value (one id or a list of ids) joined with ", ", looking each up
// with name; an id that is not stored is written as it is. No ids give "".
func (s *Service) namesOf(ctx context.Context, value any, name func(context.Context, string) (string, bool, error)) (string, error) {
	var ids []string
	switch v := value.(type) {
	case string:
		ids = []string{v}
	case []any:
		for _, item := range v {
			if id, ok := item.(string); ok {
				ids = append(ids, id)
			}
		}
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		n, found, err := name(ctx, id)
		if err != nil {
			return "", err
		}
		if !found {
			n = id
		}
		names = append(names, n)
	}
	return strings.Join(names, ", "), nil
}

// statusText writes the field "active": "Active" or "Inactive".
func statusText(value any) string {
	if active, ok := value.(bool); ok && !active {
		return "Inactive"
	}
	if value == nil {
		return ""
	}
	return "Active"
}

// percentText writes a stored percentage as people read it: "2.00 %".
func percentText(value any) string {
	text, _ := value.(string)
	p, err := money.ParsePercent(text)
	if err != nil {
		return ""
	}
	return p.Format()
}

// moneyText writes a stored money value as people read it: "10,000.00 USD".
func moneyText(value any) string {
	object, _ := value.(map[string]any)
	text, _ := object["amount"].(string)
	currency, _ := object["currency"].(string)
	amount, err := money.ParseAmount(text)
	if err != nil {
		return ""
	}
	return money.Money{Amount: amount, Currency: currency}.Format()
}

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
