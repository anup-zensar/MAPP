package settings

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

// The people of these tests: Alex administers Settings, Daniel changes them too (FR-008).
var (
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The Settings page counts the active records of each section (FR-001).
func TestOverview_ActiveCounts_FR001(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	f.addType(t, "Webinar", false)
	f.addModel(t, "Lubricants 1.0", event.ID)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.92", true)
	f.addCurrency(t, "JPY", "150", false)
	f.addAttributes(t, testAttribute{level: LevelPlan, label: "Business unit", typ: TypeMultipleChoice, options: []string{"Mobility"}})
	f.addItem(t, 2026, "2026 Lubricants Budget", "Europe – Digital", "LUB-EU-DIG", true)
	f.addItem(t, 2026, "2026 Lubricants Budget", "Europe – Events", "", false)

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

// testClock is a clock that moves only when a test moves it.
type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

// fixture is the Settings service on a temporary store, with history and a fake reference source whose
// records a test sets in refs.
type fixture struct {
	service *Service
	history *history.Service
	db      *store.DB
	clock   *testClock
	ids     *ids.Generator
	refs    []Reference
}

// newTestService returns the service on an empty temporary store, with the Settings kinds registered with
// history, the clock at start, and one reference source answering f.refs.
func newTestService(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONRepository(), generator, clock.Now)
	f := &fixture{history: hist, db: db, clock: clock, ids: generator}
	f.service = NewService(db, NewJSONRepository(), generator, hist, clock.Now)
	for _, kind := range f.service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	f.service.AddReferenceSource(func(context.Context) ([]Reference, error) { return f.refs, nil })
	return f
}

// write runs fn in a Write of the store and stops the test when it fails.
func (f *fixture) write(t *testing.T, fn func(ctx context.Context, repo Repository) error) {
	t.Helper()
	repo := f.service.repo
	if err := f.db.Write(t.Context(), func(ctx context.Context) error { return fn(ctx, repo) }); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// next returns the next ID of prefix.
func (f *fixture) next(ctx context.Context, prefix string) string {
	id, err := f.ids.Next(ctx, prefix)
	if err != nil {
		panic(err)
	}
	return id
}

// addType stores an activity type directly, as the seed does.
func (f *fixture) addType(t *testing.T, name string, active bool) ActivityType {
	t.Helper()
	var at ActivityType
	f.write(t, func(ctx context.Context, repo Repository) error {
		at = ActivityType{ID: f.next(ctx, ids.PrefixActivityType), Name: name, Active: active, Meta: record.NewMeta(actor.System, start)}
		return repo.PutActivityType(ctx, at)
	})
	return at
}

// addModel stores a model directly: 2.00 %, 5.00 % and 10,000.00 USD, allowing the activity types.
func (f *fixture) addModel(t *testing.T, name string, typeIDs ...string) Model {
	t.Helper()
	var m Model
	f.write(t, func(ctx context.Context, repo Repository) error {
		m = Model{
			ID: f.next(ctx, ids.PrefixModel), Name: name, InqToMqlRate: money.PercentFromHundredths(200),
			MqlToDealRate: money.PercentFromHundredths(500), AverageDealSize: money.USD(money.FromCents(1_000_000)),
			ActivityTypeIDs: typeIDs, Active: true, Meta: record.NewMeta(actor.System, start),
		}
		return repo.PutModel(ctx, m)
	})
	return m
}

// addCurrency stores a currency directly.
func (f *fixture) addCurrency(t *testing.T, code, rate string, active bool) Currency {
	t.Helper()
	r, err := money.ParseRate(rate)
	if err != nil {
		t.Fatalf("ParseRate(%q): %v", rate, err)
	}
	c := Currency{Code: code, Name: code + " name", Rate: r, RateChangedAt: start, RateChangedBy: actor.System, Active: active,
		Meta: record.NewMeta(actor.System, start)}
	f.write(t, func(ctx context.Context, repo Repository) error { return repo.PutCurrency(ctx, c) })
	return c
}

// addItem stores a finance budget item directly, with its budget when the budget is new.
func (f *fixture) addItem(t *testing.T, year int, budgetName, name, code string, active bool) FinanceBudgetItem {
	t.Helper()
	var item FinanceBudgetItem
	f.write(t, func(ctx context.Context, repo Repository) error {
		budgets, err := repo.ListFinanceBudgets(ctx)
		if err != nil {
			return err
		}
		budget, found := findNamed(budgets, budgetName, "", func(b FinanceBudget) (string, string) { return b.ID, b.Name })
		if !found || budget.Year != year {
			budget = FinanceBudget{ID: f.next(ctx, ids.PrefixFinanceBudget), Year: year, Name: budgetName, Meta: record.NewMeta(actor.System, start)}
			if err := repo.PutFinanceBudget(ctx, budget); err != nil {
				return err
			}
		}
		item = FinanceBudgetItem{ID: f.next(ctx, ids.PrefixFinanceBudgetItem), BudgetID: budget.ID, Name: name, Code: code, Active: active,
			Meta: record.NewMeta(actor.System, start)}
		return repo.PutFinanceBudgetItem(ctx, item)
	})
	return item
}

// testAttribute describes a Shell attribute for addAttributes. parent is the label of an earlier attribute
// of the same level; condition lists Business unit option labels; links gives each option's parent option
// labels.
type testAttribute struct {
	level           Level
	label           string
	typ             AttributeType
	marking         Marking
	section         string
	parent          string
	condition       []string
	role            string
	inactive        bool
	options         []string
	links           map[string][]string
	inactiveOptions []string
	help            string // its help text (016 FR-048)
}

// testForm is what addAttributes stored: attributes by "<level>/<label>", options by
// "<level>/<attribute>/<option>".
type testForm struct {
	attributes map[string]Attribute
	options    map[string]Option
}

// attribute returns the attribute label of level.
func (tf testForm) attribute(level Level, label string) Attribute {
	return tf.attributes[string(level)+"/"+label]
}

// option returns the id of the option of the attribute label of level.
func (tf testForm) option(level Level, label, option string) string {
	return tf.options[string(level)+"/"+label+"/"+option].ID
}

// addAttributes stores Shell attributes with their options directly, positions 1… per level in the order
// given, and marks the plan-level "Business unit" with the business-unit role.
func (f *fixture) addAttributes(t *testing.T, list ...testAttribute) testForm {
	t.Helper()
	tf := testForm{attributes: map[string]Attribute{}, options: map[string]Option{}}
	f.write(t, func(ctx context.Context, repo Repository) error {
		existing, err := repo.ListAttributes(ctx)
		if err != nil {
			return err
		}
		positions := map[Level]int{}
		for _, a := range existing {
			positions[a.Level] = max(positions[a.Level], a.Position)
			tf.attributes[string(a.Level)+"/"+a.Label] = a
		}
		options, err := repo.ListOptions(ctx)
		if err != nil {
			return err
		}
		for _, o := range options {
			for _, a := range existing {
				if a.ID == o.AttributeID {
					tf.options[string(a.Level)+"/"+a.Label+"/"+o.Label] = o
				}
			}
		}
		for _, ta := range list {
			positions[ta.level]++
			a := Attribute{
				ID: f.next(ctx, ids.PrefixShellAttribute), Label: ta.label, HelpText: ta.help, Level: ta.level, Type: ta.typ,
				Marking: cmp.Or(ta.marking, MarkingOptional), Section: cmp.Or(ta.section, DefaultSection),
				Position: positions[ta.level], Condition: []string{}, Role: ta.role, Active: !ta.inactive,
				Meta: record.NewMeta(actor.System, start),
			}
			if ta.level == LevelPlan && ta.label == "Business unit" {
				a.Role = RoleBusinessUnit
			}
			if ta.parent != "" {
				a.ParentAttributeID = tf.attribute(ta.level, ta.parent).ID
			}
			for _, unit := range ta.condition {
				a.Condition = append(a.Condition, tf.option(LevelPlan, "Business unit", unit))
			}
			if err := repo.PutAttribute(ctx, a); err != nil {
				return err
			}
			tf.attributes[string(a.Level)+"/"+a.Label] = a
			for i, label := range ta.options {
				o := Option{
					ID: f.next(ctx, ids.PrefixShellOption), AttributeID: a.ID, Label: label, Position: i + 1,
					ParentOptionIDs: []string{}, Active: !slices.Contains(ta.inactiveOptions, label), Meta: record.NewMeta(actor.System, start),
				}
				for _, parent := range ta.links[label] {
					o.ParentOptionIDs = append(o.ParentOptionIDs, tf.option(ta.level, ta.parent, parent))
				}
				if err := repo.PutOption(ctx, o); err != nil {
					return err
				}
				tf.options[string(a.Level)+"/"+a.Label+"/"+label] = o
			}
		}
		return nil
	})
	return tf
}

// planRef is a plan using the model modelID, as feature 004's source reports it.
func planRef(id, name, modelID string, inBin bool, values AttributeValues) Reference {
	return Reference{Record: history.RecordRef{Type: "plan", ID: id, Name: name}, InBin: inBin, Level: LevelPlan, PlanID: id,
		ModelID: modelID, Attributes: values}
}

// activityRef is an activity of the plan planID and the activity type typeID.
func activityRef(id, planID, typeID string, inBin bool, values AttributeValues) Reference {
	return Reference{Record: history.RecordRef{Type: "activity", ID: id, Name: "Activity " + id}, InBin: inBin,
		Level: LevelActivity, PlanID: planID, ActivityID: id, ActivityTypeID: typeID, Attributes: values}
}

// lineItemRef is a line item of the activity activityID with a planned cost in currency.
func lineItemRef(id, planID, activityID, currency, cost string, inBin bool) Reference {
	amount, err := money.ParseAmount(cost)
	if err != nil {
		panic(err)
	}
	return Reference{Record: history.RecordRef{Type: "lineItem", ID: id, Name: "Line item " + id}, InBin: inBin,
		Level: LevelLineItem, PlanID: planID, ActivityID: activityID, Currency: currency, PlannedCost: &amount}
}

// personRef is a person showing amounts in the display currency currency.
func personRef(id, currency string) Reference {
	return Reference{Record: history.RecordRef{Type: "person", ID: id, Name: "Person " + id}, Currency: currency}
}

// values builds attribute values from pairs of attribute id and JSON value.
func values(pairs ...string) AttributeValues {
	v := AttributeValues{}
	for i := 0; i+1 < len(pairs); i += 2 {
		v[pairs[i]] = json.RawMessage(pairs[i+1])
	}
	return v
}

// version returns a pointer to v, as an update sends it.
func version(v int) *int { return &v }

// text returns a pointer to s, as an update sends a field.
func text(s string) *string { return &s }

// yes returns a pointer to b, as a status update sends it.
func yes(b bool) *bool { return &b }

// logEntries returns the activity log, oldest first.
func (f *fixture) logEntries(t *testing.T) []history.Entry {
	t.Helper()
	var entries []history.Entry
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entries, err = history.NewJSONRepository().Entries(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("reading the log: %v", err)
	}
	return entries
}

// lastEntry returns the newest activity-log entry.
func (f *fixture) lastEntry(t *testing.T) history.Entry {
	t.Helper()
	entries := f.logEntries(t)
	if len(entries) == 0 {
		t.Fatal("the activity log is empty")
	}
	return entries[len(entries)-1]
}

// assertError checks that err is the API error want.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v\nwant    %#v", err, want)
	}
}

// problems returns the validation error of the field problems given as field, message pairs.
func problems(pairs ...string) *errs.Error {
	var fields []errs.FieldError
	for i := 0; i+1 < len(pairs); i += 2 {
		fields = append(fields, errs.FieldError{Field: pairs[i], Message: pairs[i+1]})
	}
	return errs.Validation(fields...)
}
