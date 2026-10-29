package bulk_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"mapp/internal/bulk"
	"mapp/internal/bulk/bulktest"
	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
)

var (
	priya = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
)

const recordKind = "record"

// stored is a record of the fake planning areas: what the actions read and change.
type stored struct {
	bulk.Record
	OwnerID string
	Value   string
}

// fakePlanning keeps records in a fake table, so a discarded Write puts them back, and logs each change.
type fakePlanning struct {
	records    *fake.Table[stored]
	log        *history.Service
	refusals   map[string]error
	people     map[string]bool
	fields     []bulk.Field
	below      map[string]bulk.Below
	targets    []bulk.Target
	effects    bulk.Consequences
	restarted  []string
	submitted  []string
	brokenNext bool
}

func (f *fakePlanning) Find(_ context.Context, _ actor.Actor, id string) (bulk.Record, error) {
	s, found, err := f.records.Get(id)
	if err != nil || !found {
		return bulk.Record{}, errors.Join(err, errs.NotFound("This record no longer exists."))
	}
	return s.Record, nil
}

func (f *fakePlanning) Below(_ context.Context, r bulk.Record) (bulk.Below, error) {
	return f.below[r.ID], nil
}

func (f *fakePlanning) Fields(context.Context, bulk.Level) ([]bulk.Field, error) {
	return f.fields, nil
}

func (f *fakePlanning) Destinations(context.Context, actor.Actor, bulk.Kind, bulk.Record) ([]bulk.Target, error) {
	return f.targets, nil
}

func (f *fakePlanning) Consequences(context.Context, actor.Actor, bulk.Record, string) (bulk.Consequences, error) {
	return f.effects, nil
}

func (f *fakePlanning) Person(_ context.Context, id string) (string, bool, bool, error) {
	active, found := f.people[id]
	return "Person " + id, active, found, nil
}

func (f *fakePlanning) change(ctx context.Context, a actor.Actor, id string, edit func(*stored) bool) (bool, error) {
	if f.brokenNext {
		return false, errors.New("the database went away")
	}
	if err := f.refusals[id]; err != nil {
		return false, err
	}
	s, _, err := f.records.Get(id)
	if err != nil || !edit(&s) {
		return false, err
	}
	if err = f.records.Put(ctx, s); err != nil {
		return false, err
	}
	_, err = f.log.Log(ctx, a, history.LogInput{Kind: recordKind, RecordID: id, RecordName: s.Name, Action: history.ActionChanged})
	return err == nil, err
}

func (f *fakePlanning) SetOwner(ctx context.Context, a actor.Actor, r bulk.Record, ownerID string) (bool, error) {
	return f.change(ctx, a, r.ID, func(s *stored) bool {
		changed := s.OwnerID != ownerID
		s.OwnerID = ownerID
		return changed
	})
}

func (f *fakePlanning) SetField(ctx context.Context, a actor.Actor, r bulk.Record, c bulk.SetField) (bool, error) {
	if c.RestartApproval {
		f.restarted = append(f.restarted, r.ID)
	}
	return f.change(ctx, a, r.ID, func(s *stored) bool {
		value, _ := json.Marshal(c.Value)
		changed := s.Value != string(value)
		s.Value = string(value)
		return changed
	})
}

func (f *fakePlanning) Submit(ctx context.Context, a actor.Actor, r bulk.Record, _ string) error {
	_, err := f.change(ctx, a, r.ID, func(s *stored) bool { s.Status = bulk.StatusSubmitted; return true })
	if err == nil {
		f.submitted = append(f.submitted, r.ID)
	}
	return err
}

func (f *fakePlanning) Copy(ctx context.Context, a actor.Actor, r bulk.Record, _ string) (string, error) {
	_, err := f.change(ctx, a, r.ID, func(*stored) bool { return true })
	return "copy-of-" + r.ID, err
}

func (f *fakePlanning) Move(ctx context.Context, a actor.Actor, r bulk.Record, targetID string) error {
	_, err := f.change(ctx, a, r.ID, func(s *stored) bool { s.Ancestors = []string{targetID}; return true })
	return err
}

func (f *fakePlanning) Delete(ctx context.Context, a actor.Actor, r bulk.Record) (history.BinEntry, error) {
	if _, err := f.change(ctx, a, r.ID, func(*stored) bool { return true }); err != nil {
		return history.BinEntry{}, err
	}
	return history.BinEntry{ID: "BIN-" + r.ID, RecordID: r.ID, RecordName: r.Name, DeletedAt: start, DeletedBy: a}, f.records.Delete(ctx, r.ID)
}

type fixture struct {
	db       *fake.DB
	service  *bulk.Service
	store    bulktest.Store
	planning *fakePlanning
	history  *history.Service
}

func newFixture(t *testing.T, records ...stored) *fixture {
	t.Helper()
	db := fake.NewDB()
	now := func() time.Time { return start }
	generator := ids.NewGenerator(fake.NewCounters(db))
	f := &fixture{db: db, store: bulktest.NewStore(db)}
	f.history = history.NewService(db, historytest.NewStore(db), generator, now)
	f.planning = &fakePlanning{records: fake.TableOf(db, "records", func(s stored) string { return s.ID }), log: f.history,
		refusals: map[string]error{}, people: map[string]bool{"U-1003": true, "U-1009": false}, below: map[string]bulk.Below{}}
	if err := f.history.Register(f.planning.kind()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	f.service = bulk.NewService(history.Tools{Tx: db, IDs: generator, Log: f.history, Now: now}, f.store, f.planning)
	if err := db.Write(t.Context(), func(ctx context.Context) error { return f.planning.records.Put(ctx, records...) }); err != nil {
		t.Fatalf("Put records: %v", err)
	}
	return f
}

func (f *fakePlanning) kind() history.Kind {
	return history.Kind{Name: recordKind, Singular: "record", Plural: "records",
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			s, found, err := f.records.Get(id)
			return history.Snapshot{ID: s.ID, Name: s.Name}, found, err
		},
		Remove: func(context.Context, string) error { return nil },
		Put:    func(context.Context, history.Snapshot) error { return nil },
	}
}

func (f *fixture) stored(t *testing.T, id string) stored {
	t.Helper()
	s, _, err := f.planning.records.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *fixture) log(t *testing.T) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

func activity(id, name, status string) stored {
	return stored{Record: bulk.Record{ID: id, Level: bulk.LevelActivity, Name: name, PlanID: "P-1001", PlanName: "FY2026",
		Year: 2026, Ancestors: []string{"I-1001", "C-1001"}, Status: status, Shown: map[string]string{
			bulk.StatusCreated: "Created", bulk.StatusSubmitted: "Submitted", bulk.StatusApproved: "Approved"}[status]}}
}

func container(id, name string, level bulk.Level, ancestors ...string) stored {
	return stored{Record: bulk.Record{ID: id, Level: level, Name: name, PlanID: "P-1001", PlanName: "FY2026", Year: 2026,
		Ancestors: ancestors}}
}

func assertValidation(t *testing.T, err error, want errs.FieldError) {
	t.Helper()
	got, ok := errors.AsType[*errs.Error](err)
	if !ok || got.Code != errs.CodeValidationFailed || len(got.Fields) != 1 || got.Fields[0] != want {
		t.Errorf("error = %+v, want validation_failed %+v", err, want)
	}
}
