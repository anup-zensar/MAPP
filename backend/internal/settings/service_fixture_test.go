package settings

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

var (
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
)

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

// fixture's reference source answers the records a test sets in refs.
type fixture struct {
	service *Service
	history *history.Service
	db      *fake.DB
	clock   *testClock
	ids     *ids.Generator
	refs    []Reference
}

func newTestService(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	clock := &testClock{now: start}
	generator := ids.NewGenerator(fake.NewCounters(db))
	hist := history.NewService(db, historytest.NewStore(db), generator, clock.Now)
	f := &fixture{history: hist, db: db, clock: clock, ids: generator}
	f.service = NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}, newFakeStore(db))
	for _, kind := range f.service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	f.service.AddReferenceSource(func(context.Context) ([]Reference, error) { return f.refs, nil })
	return f
}

func (f *fixture) write(t *testing.T, fn func(ctx context.Context, repo Store) error) {
	t.Helper()
	repo := f.service.repo
	if err := f.db.Write(t.Context(), func(ctx context.Context) error { return fn(ctx, repo) }); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func (f *fixture) next(ctx context.Context, prefix string) string {
	id, err := f.ids.Next(ctx, prefix)
	if err != nil {
		panic(err)
	}
	return id
}

func (f *fixture) addType(t *testing.T, name string, active bool) ActivityType {
	t.Helper()
	var at ActivityType
	f.write(t, func(ctx context.Context, repo Store) error {
		at = ActivityType{ID: f.next(ctx, ids.PrefixActivityType), Name: name, Active: active, Meta: record.NewMeta(actor.System, start)}
		return repo.PutActivityType(ctx, at)
	})
	return at
}

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

func version(v int) *int { return &v }

func text(s string) *string { return &s }

func yes(b bool) *bool { return &b }

func (f *fixture) logEntries(t *testing.T) []history.Entry {
	t.Helper()
	entries, err := historytest.NewStore(f.db).Entries(t.Context())
	if err != nil {
		t.Fatalf("reading the log: %v", err)
	}
	return entries
}

func (f *fixture) lastEntry(t *testing.T) history.Entry {
	t.Helper()
	entries := f.logEntries(t)
	if len(entries) == 0 {
		t.Fatal("the activity log is empty")
	}
	return entries[len(entries)-1]
}

func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v\nwant    %#v", err, want)
	}
}

// problems takes field, message pairs.
func problems(pairs ...string) *errs.Error {
	var fields []errs.FieldError
	for i := 0; i+1 < len(pairs); i += 2 {
		fields = append(fields, errs.FieldError{Field: pairs[i], Message: pairs[i+1]})
	}
	return errs.Validation(fields...)
}
