package settings

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
)

// The people of these tests: Alex administers Settings, Daniel changes them too (FR-008).
var (
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

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
	hist := history.NewService(db, history.NewJSONStore(), generator, clock.Now)
	f := &fixture{history: hist, db: db, clock: clock, ids: generator}
	f.service = NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}, NewJSONStore())
	for _, kind := range f.service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	f.service.AddReferenceSource(func(context.Context) ([]Reference, error) { return f.refs, nil })
	return f
}

// write runs fn in a Write of the store and stops the test when it fails.
func (f *fixture) write(t *testing.T, fn func(ctx context.Context, repo Store) error) {
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
	f.write(t, func(ctx context.Context, repo Store) error {
		at = ActivityType{ID: f.next(ctx, ids.PrefixActivityType), Name: name, Active: active, Meta: record.NewMeta(actor.System, start)}
		return repo.PutActivityType(ctx, at)
	})
	return at
}

// addModel stores a model directly: 2.00 %, 5.00 % and 10,000.00 USD, allowing the activity types.
func (f *fixture) addModel(t *testing.T, name string, typeIDs ...string) Model {
	t.Helper()
	var m Model
	f.write(t, func(ctx context.Context, repo Store) error {
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
func (f *fixture) addCurrency(t *testing.T, code, rate string, active bool) {
	t.Helper()
	r, err := money.ParseRate(rate)
	if err != nil {
		t.Fatalf("ParseRate(%q): %v", rate, err)
	}
	c := Currency{Code: code, Name: code + " name", Rate: r, RateChangedAt: start, RateChangedBy: actor.System, Active: active,
		Meta: record.NewMeta(actor.System, start)}
	f.write(t, func(ctx context.Context, repo Store) error { return repo.PutCurrency(ctx, c) })
}

// addItem stores the finance budget item in directly, with its budget when the budget is new.
func (f *fixture) addItem(t *testing.T, in ItemInput, active bool) FinanceBudgetItem {
	t.Helper()
	var item FinanceBudgetItem
	f.write(t, func(ctx context.Context, repo Store) error {
		budgets, err := repo.ListFinanceBudgets(ctx)
		if err != nil {
			return err
		}
		budget, found := findNamed(budgets, in.BudgetName, "", func(b FinanceBudget) (string, string) { return b.ID, b.Name })
		if !found || budget.Year != *in.Year {
			budget = FinanceBudget{ID: f.next(ctx, ids.PrefixFinanceBudget), Year: *in.Year, Name: in.BudgetName, Meta: record.NewMeta(actor.System, start)}
			if err := repo.PutFinanceBudget(ctx, budget); err != nil {
				return err
			}
		}
		item = FinanceBudgetItem{ID: f.next(ctx, ids.PrefixFinanceBudgetItem), BudgetID: budget.ID, Name: in.Name, Code: in.Code, Active: active,
			Meta: record.NewMeta(actor.System, start)}
		return repo.PutFinanceBudgetItem(ctx, item)
	})
	return item
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
		entries, err = history.NewJSONStore().Entries(ctx)
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
