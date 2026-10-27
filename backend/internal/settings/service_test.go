package settings

import (
	"slices"
	"testing"

	"mapp/internal/history"
)

// The Settings page counts the active records of each section (FR-001).
func TestOverview_ActiveCounts_FR001(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	f.addType(t, "Webinar", false)
	f.addModel(t, "Lubricants 1.0", event.ID)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.92", true)
	f.addCurrency(t, "JPY", "150", false)
	f.addAttributes(t, testAttribute{level: LevelPlan, label: "Business unit", typ: TypeMultipleChoice, options: []string{"Mobility"}})
	f.addItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", "LUB-EU-DIG"), true)
	f.addItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Events", ""), false)

	overview, err := f.service.Overview(t.Context(), alex)

	want := SettingsOverview{Models: 1, ActivityTypes: 1, ShellAttributes: 1, Currencies: 2, FinanceBudgetItems: 1}
	if err != nil || overview != want {
		t.Errorf("Overview = %+v, %v; want %+v", overview, err, want)
	}
}

// Every create, change, inactivation, reactivation, delete and restore of every Settings record writes one
// activity-log entry naming the record by type and name (FR-006). Finance budgets are created with their
// first item and renamed; they are never set inactive or deleted on their own (D-047).
func TestEveryChangeLogged_FR006(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	ctx := t.Context()
	event := f.addType(t, "Event", true)
	parent, err := f.service.CreateAttribute(ctx, alex, ShellAttributeInput{Label: "Media channel", Level: "activity", Type: "singleChoice",
		Marking: "optional", Options: []NewOption{{Label: "TV"}}})
	if err != nil {
		t.Fatalf("CreateAttribute: %v", err)
	}

	// The life of one record of a kind: create it, change it from version 1, set it inactive from version
	// 2 and reactivate it from version 3, delete it.
	type life struct {
		kind   string
		create func() (string, error)
		change func(id string) error
		status func(id string, active bool, version int) error
		remove func(id string) (history.BinEntry, error)
	}
	status := func(active bool, v int) StatusUpdate { return StatusUpdate{Active: yes(active), Version: version(v)} }
	lives := []life{
		{kind: "model",
			create: func() (string, error) {
				m, err := f.service.CreateModel(ctx, alex, modelInput("Model", event.ID))
				return m.ID, err
			},
			change: func(id string) error {
				_, err := f.service.UpdateModel(ctx, alex, id, ModelUpdate{Name: text("Model 2"), Version: version(1)})
				return err
			},
			status: func(id string, active bool, v int) error {
				_, err := f.service.SetModelActive(ctx, alex, id, status(active, v))
				return err
			},
			remove: func(id string) (history.BinEntry, error) { return f.service.DeleteModel(ctx, alex, id) }},
		{kind: "activityType",
			create: func() (string, error) {
				at, err := f.service.CreateActivityType(ctx, alex, typeInput("Webinar", ""))
				return at.ID, err
			},
			change: func(id string) error {
				_, err := f.service.UpdateActivityType(ctx, alex, id, ActivityTypeUpdate{Name: text("Webinars"), Version: version(1)})
				return err
			},
			status: func(id string, active bool, v int) error {
				_, err := f.service.SetActivityTypeActive(ctx, alex, id, status(active, v))
				return err
			},
			remove: func(id string) (history.BinEntry, error) { return f.service.DeleteActivityType(ctx, alex, id) }},
		{kind: "shellAttribute",
			create: func() (string, error) {
				a, err := f.service.CreateAttribute(ctx, alex, ShellAttributeInput{Label: "Notes", Level: "activity", Type: "text", Marking: "optional"})
				return a.ID, err
			},
			change: func(id string) error {
				_, err := f.service.UpdateAttribute(ctx, alex, id, ShellAttributeUpdate{Label: text("Remarks"), Version: version(1)})
				return err
			},
			status: func(id string, active bool, v int) error {
				_, err := f.service.SetAttributeActive(ctx, alex, id, status(active, v))
				return err
			},
			remove: func(id string) (history.BinEntry, error) { return f.service.DeleteAttribute(ctx, alex, id) }},
		{kind: "shellAttributeOption",
			create: func() (string, error) {
				o, err := f.service.CreateOption(ctx, alex, parent.ID, OptionInput{Label: "Radio"})
				return o.ID, err
			},
			change: func(id string) error {
				_, err := f.service.UpdateOption(ctx, alex, id, OptionUpdate{Label: text("Online radio"), Version: version(1)})
				return err
			},
			status: func(id string, active bool, v int) error {
				_, err := f.service.SetOptionActive(ctx, alex, id, status(active, v))
				return err
			},
			remove: func(id string) (history.BinEntry, error) { return f.service.DeleteOption(ctx, alex, id) }},
		{kind: "currency",
			create: func() (string, error) { c, err := f.service.CreateCurrency(ctx, alex, jpy()); return c.Code, err },
			change: func(id string) error {
				// Since 016 a new rate is logged on the exchange rate; the name is the currency's own change.
				_, err := f.service.UpdateCurrency(ctx, alex, id, CurrencyUpdate{Name: text("Yen"), Rate: text("151"), Version: version(1)})
				return err
			},
			status: func(id string, active bool, v int) error {
				_, err := f.service.SetCurrencyActive(ctx, alex, id, status(active, v))
				return err
			},
			remove: func(id string) (history.BinEntry, error) { return f.service.DeleteCurrency(ctx, alex, id) }},
		{kind: "financeBudgetItem",
			create: func() (string, error) {
				i, err := f.service.CreateFinanceBudgetItem(ctx, alex, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))
				return i.ID, err
			},
			change: func(id string) error {
				_, err := f.service.UpdateFinanceBudgetItem(ctx, alex, id, ItemUpdate{Code: text("LUB-EU-DIG"), Version: version(1)})
				return err
			},
			status: func(id string, active bool, v int) error {
				_, err := f.service.SetFinanceBudgetItemActive(ctx, alex, id, status(active, v))
				return err
			},
			remove: func(id string) (history.BinEntry, error) { return f.service.DeleteFinanceBudgetItem(ctx, alex, id) }},
	}

	for _, l := range lives {
		id, err := l.create()
		if err != nil {
			t.Fatalf("%s: create: %v", l.kind, err)
		}
		if err := l.change(id); err != nil {
			t.Fatalf("%s: change: %v", l.kind, err)
		}
		if err := l.status(id, false, 2); err != nil {
			t.Fatalf("%s: set inactive: %v", l.kind, err)
		}
		if err := l.status(id, true, 3); err != nil {
			t.Fatalf("%s: reactivate: %v", l.kind, err)
		}
		entry, err := l.remove(id)
		if err != nil {
			t.Fatalf("%s: delete: %v", l.kind, err)
		}
		if _, err := f.history.Restore(ctx, daniel, entry.ID); err != nil {
			t.Fatalf("%s: restore: %v", l.kind, err)
		}

		var got []string
		for _, e := range f.logEntries(t) {
			if e.RecordType == l.kind && e.RecordID == id {
				got = append(got, e.Verb)
			}
		}
		want := []string{"created", "changed", "deactivated", "reactivated", "deleted", "restored"}
		if !slices.Equal(got, want) {
			t.Errorf("%s: log = %q, want %q", l.kind, got, want)
		}
	}

	item, err := f.service.CreateFinanceBudgetItem(ctx, alex, itemInput(2027, "2027 Budget", "China", ""))
	if err != nil {
		t.Fatalf("CreateFinanceBudgetItem: %v", err)
	}
	if _, err := f.service.RenameFinanceBudget(ctx, alex, item.BudgetID, RenameInput{Name: "2027 Mobility Budget", Version: version(1)}); err != nil {
		t.Fatalf("RenameFinanceBudget: %v", err)
	}
	var budget []string
	for _, e := range f.logEntries(t) {
		if e.RecordType == "financeBudget" && e.RecordID == item.BudgetID {
			budget = append(budget, e.Verb+" "+e.RecordName)
		}
	}
	if want := []string{"created 2027 Budget", "changed 2027 Mobility Budget"}; !slices.Equal(budget, want) {
		t.Errorf("finance budget: log = %q, want %q", budget, want)
	}
}
