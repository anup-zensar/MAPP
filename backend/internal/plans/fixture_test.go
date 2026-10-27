package plans

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
)

// fixture is the plans service on a temporary store, with its fakes and history.
type fixture struct {
	service   *Service
	history   *history.Service
	db        *store.DB
	clock     *testClock
	settings  *fakeSettings
	directory *fakeDirectory
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONStore(), generator, clock.Now)
	f := &fixture{history: hist, db: db, clock: clock, settings: newFakeSettings(), directory: newFakeDirectory()}
	f.service = NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}, NewJSONStore(), f.settings, f.directory)
	for _, kind := range f.service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return f
}

// addLevel registers a fake level with facts and returns it.
func (f *fixture) addLevel(t *testing.T, name string, facts map[string]Facts) *fakeLevel {
	t.Helper()
	l := &fakeLevel{name: name, facts: facts}
	if err := f.service.RegisterLevel(l.level()); err != nil {
		t.Fatalf("RegisterLevel(%s): %v", name, err)
	}
	return l
}

// input is a complete plan form: FY2026 Lubricants Test with Lubricants 1.0, Lubricants Europe and Priya
// as owner, business unit Lubricants.
func input(name string) PlanInput {
	return PlanInput{
		Name: name, Year: ptr(2026), ModelID: ptr(lubricantsModel), TeamIDs: []string{teamLubricants},
		OwnerID: ptr(priya.ID), DefaultSpread: string(SpreadEvenly), GoalStage: string(StageINQ),
		Attributes: settings.AttributeValues{businessUnit: json.RawMessage(`["` + lubricants + `"]`)},
	}
}

// updateOf is the form of the stored plan p, as the edit page sends it unchanged.
func updateOf(p Plan) PlanUpdate {
	u := PlanUpdate{
		Version: ptr(p.Version), Name: p.Name, ModelID: ptr(p.ModelID), TeamIDs: slices.Clone(p.TeamIDs), AOP: p.AOP,
		Description: p.Description, DefaultSpread: string(p.DefaultSpread), BCPIR: p.BCPIR,
		GoalStage: string(p.GoalStage), Attributes: maps.Clone(p.Attributes),
	}
	if p.OwnerID != "" {
		u.OwnerID = ptr(p.OwnerID)
	}
	if p.Budget != nil {
		u.Budget = &money.Input{Amount: p.Budget.String(), Currency: "USD"}
	}
	if p.DefaultFinanceBudgetID != "" {
		u.DefaultFinanceBudgetID = ptr(p.DefaultFinanceBudgetID)
	}
	if p.GoalAmount != "" {
		u.GoalAmount = ptr(p.GoalAmount)
	}
	return u
}

// create creates a plan as a and stops the test when that fails.
func (f *fixture) create(t *testing.T, a actor.Actor, in PlanInput) PlanView {
	t.Helper()
	view, err := f.service.Create(t.Context(), a, in)
	if err != nil {
		t.Fatalf("Create(%q): %v", in.Name, err)
	}
	return view
}

// stored reads the plan id as stored.
func (f *fixture) stored(t *testing.T, id string) Plan {
	t.Helper()
	var p Plan
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		p, err = f.service.stored(ctx, id)
		return err
	})
	if err != nil {
		t.Fatalf("stored(%s): %v", id, err)
	}
	return p
}

// entries returns the activity-log entries, newest first.
func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T {
	return &v
}

// usd returns the USD amount of text, such as "1200000.00".
func usd(t *testing.T, text string) money.Amount {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return amount
}

// assertError checks that err is exactly want.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v\nwant    %#v", err, want)
	}
}

// problems returns the validation error of the field problems, given as field and message pairs.
func problems(pairs ...string) *errs.Error {
	var fields []errs.FieldError
	for i := 0; i+1 < len(pairs); i += 2 {
		fields = append(fields, errs.FieldError{Field: pairs[i], Message: pairs[i+1]})
	}
	return errs.Validation(fields...)
}
