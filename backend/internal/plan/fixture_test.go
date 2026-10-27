package plan

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

type fixture struct {
	service   *Service
	history   *history.Service
	db        *fake.DB
	clock     *testClock
	settings  *fakeSettings
	directory *fakeDirectory
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	clock := &testClock{now: start}
	generator := ids.NewGenerator(fake.NewCounters(db))
	hist := history.NewService(db, historytest.NewStore(db), generator, clock.Now)
	f := &fixture{history: hist, db: db, clock: clock, settings: newFakeSettings(), directory: newFakeDirectory()}
	f.service = NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}, newFakeStore(db), f.settings, f.directory)
	for _, kind := range f.service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return f
}

func (f *fixture) addLevel(t *testing.T, name string, facts map[string]Facts) *fakeLevel {
	t.Helper()
	l := &fakeLevel{name: name, facts: facts}
	if err := f.service.RegisterLevel(l.level()); err != nil {
		t.Fatalf("RegisterLevel(%s): %v", name, err)
	}
	return l
}

// input is a complete plan form: Lubricants 1.0, Lubricants Europe, Priya as owner, business unit Lubricants.
func input(name string) PlanInput {
	return PlanInput{
		Name: name, Year: ptr(2026), ModelID: ptr(lubricantsModel), TeamIDs: []string{teamLubricants},
		OwnerID: ptr(priya.ID), DefaultSpread: string(SpreadEvenly), GoalStage: string(StageINQ),
		Attributes: settings.AttributeValues{businessUnit: json.RawMessage(`["` + lubricants + `"]`)},
	}
}

// updateOf is the form of p as the edit page sends it unchanged.
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

func (f *fixture) create(t *testing.T, a actor.Actor, in PlanInput) PlanView {
	t.Helper()
	view, err := f.service.Create(t.Context(), a, in)
	if err != nil {
		t.Fatalf("Create(%q): %v", in.Name, err)
	}
	return view
}

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

// entries are newest first.
func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

func ptr[T any](v T) *T {
	return &v
}

func usd(t *testing.T, text string) money.Amount {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return amount
}

func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v\nwant    %#v", err, want)
	}
}

// problems returns the validation error of field and message pairs.
func problems(pairs ...string) *errs.Error {
	var fields []errs.FieldError
	for i := 0; i+1 < len(pairs); i += 2 {
		fields = append(fields, errs.FieldError{Field: pairs[i], Message: pairs[i+1]})
	}
	return errs.Validation(fields...)
}
