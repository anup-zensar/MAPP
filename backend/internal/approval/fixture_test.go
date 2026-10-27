package approval

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/people"
	"mapp/internal/people/peopletest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The demo people of these tests (demo-users-and-data.md), with the ids the seed gives them.
var (
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	mei    = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
	carlos = actor.Actor{ID: "U-1005", Name: "Carlos Mendes"}
	sara   = actor.Actor{ID: "U-1006", Name: "Sara Novak"}
)

const (
	lubricantsFlow = "WF-1001" // Lubricants EU approval: level 1 Daniel Okoro
	chinaFlow      = "WF-1002" // China BC approval workflow: level 1 Carlos Mendes, level 2 Alex Morgan
)

type fixture struct {
	service    *Service
	activities *fakeActivities
	history    *history.Service
	db         *fake.DB
	clock      *testClock
	access     *fakeAccess
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	clock := &testClock{now: start}
	generator := ids.NewGenerator(fake.NewCounters(db))
	hist := history.NewService(db, historytest.NewStore(db), generator, clock.Now)
	activities := &fakeActivities{
		clock: clock, facts: map[string]ActivityFacts{}, problems: map[string][]SubmitProblem{}, missing: map[string][]MissingFieldRef{},
		unassigned: map[string]*money.Money{}, lineItems: map[string][]PreviewLineItem{},
	}
	peopleRepo := peopletest.NewStore(db)
	service := NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}, newFakeStore(db), peopleRepo, activities)
	access := &fakeAccess{rule: true, cannotApprove: map[string]bool{}, holders: map[string][]string{}}
	service.UseAccess(access)
	service.UseRuleChoices(fakeChoices{})
	f := &fixture{service: service, activities: activities, history: hist, db: db, clock: clock, access: access}

	// The activity kind of feature 006, as far as the log needs it: its name.
	err := hist.Register(history.Kind{
		Name: activityKind, Singular: "activity", Plural: "activities",
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			facts, found := activities.facts[id]
			return history.Snapshot{ID: id, Name: facts.Name}, found, nil
		},
		Remove: func(context.Context, string) error { return nil },
		Put:    func(context.Context, history.Snapshot) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register(activity): %v", err)
	}
	for _, k := range service.Kinds() {
		if err := hist.Register(k); err != nil {
			t.Fatalf("Register(%s): %v", k.Name, err)
		}
	}

	f.write(t, func(ctx context.Context) error {
		for _, p := range []actor.Actor{alex, priya, daniel, mei, carlos, sara} {
			person := people.Person{ID: p.ID, Name: p.Name, Email: p.ID + "@shell.com", Active: true, Meta: record.NewMeta(actor.System, start)}
			if err := peopleRepo.Put(ctx, person); err != nil {
				return err
			}
		}
		for _, w := range []Workflow{
			{ID: lubricantsFlow, Name: "Lubricants EU approval", Active: true,
				Levels: []Level{{ApproverIDs: []string{daniel.ID}}}, Meta: record.NewMeta(actor.System, start)},
			{ID: chinaFlow, Name: "China BC approval workflow", Active: true,
				Levels: []Level{{ApproverIDs: []string{carlos.ID}}, {ApproverIDs: []string{alex.ID}}}, Meta: record.NewMeta(actor.System, start)},
		} {
			if err := service.repo.PutWorkflow(ctx, w); err != nil {
				return err
			}
		}
		// The demo workflows took WF-1001 and WF-1002: new ones start at WF-1003.
		for range 2 {
			if _, err := generator.Next(ctx, ids.PrefixApprovalWorkflow); err != nil {
				return err
			}
		}
		return nil
	})
	return f
}

func (f *fixture) addActivity(t *testing.T, id, name, owner, workflowID string) ActivityFacts {
	t.Helper()
	startDate, endDate := date.MustParse("2026-04-01"), date.MustParse("2026-06-30")
	cost := money.USD(mustAmount(t, "50000.00"))
	facts := ActivityFacts{
		ID: id, Name: name, Status: StatusCreated, DisplayStatus: StatusCreated,
		Path: []errs.RecordRef{
			{Type: "plan", ID: "P-1001", Name: "FY2026 Lubricants Europe"},
			{Type: "campaign", ID: "C-1001", Name: "Helix launch"},
			{Type: "initiative", ID: "I-1001", Name: "Digital & social"},
		},
		OwnerID: owner, StartDate: &startDate, EndDate: &endDate, PlannedCost: &cost, UnassignedPlannedCost: &cost,
		TypeName: "Event", WorkflowID: workflowID, Meta: record.NewMeta(actor.System, start),
		Plan: PlanFacts{ID: "P-1001", TeamIDs: []string{"T-1001"}}, ActivityTypeID: eventType,
		BusinessUnits: []Choice{{ID: lubricantsUnit, Label: "Lubricants", Active: true}},
	}
	f.activities.facts[id] = facts
	f.activities.order = append(f.activities.order, id)
	return facts
}

func version(v int) *int {
	return &v
}

func (f *fixture) submit(t *testing.T, a actor.Actor, id, comment string) ActivityApproval {
	t.Helper()
	approval, err := f.service.Submit(t.Context(), a, id, SubmitInput{Version: version(f.activities.facts[id].Version), Comment: comment})
	if err != nil {
		t.Fatalf("Submit(%s): %v", id, err)
	}
	return approval
}

func (f *fixture) read(t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	if err := f.db.Read(t.Context(), fn); err != nil {
		t.Fatalf("read: %v", err)
	}
}

func (f *fixture) write(t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	if err := f.db.Write(t.Context(), fn); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func (f *fixture) rounds(t *testing.T, id string) []Round {
	t.Helper()
	var rounds []Round
	f.read(t, func(ctx context.Context) (err error) {
		rounds, err = f.service.repo.RoundsOf(ctx, id)
		return err
	})
	return rounds
}

func (f *fixture) actions(t *testing.T, id string) []Action {
	t.Helper()
	var actions []Action
	f.read(t, func(ctx context.Context) (err error) {
		actions, err = f.service.repo.ActionsOf(ctx, id)
		return err
	})
	return actions
}

func (f *fixture) notifications(t *testing.T, personID string) []Notification {
	t.Helper()
	var list []Notification
	f.read(t, func(ctx context.Context) (err error) {
		list, err = f.service.repo.NotificationsOf(ctx, personID)
		return err
	})
	return list
}

func (f *fixture) log(t *testing.T, id string) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{RecordID: id, Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

func mustAmount(t *testing.T, text string) money.Amount {
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
		t.Errorf("error = %#v, want %#v", err, want)
	}
}

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func (f *fixture) setManager(t *testing.T, p, manager actor.Actor) {
	t.Helper()
	repo := peopletest.NewStore(f.db)
	f.write(t, func(ctx context.Context) error {
		person, _, err := repo.Get(ctx, p.ID)
		if err != nil {
			return err
		}
		person.ManagerID = manager.ID
		return repo.Put(ctx, person)
	})
}

// decideLevel is a level of named people where any one approves, as Submit stores it.
func decideLevel(n int, approvers ...actor.Actor) RoundLevel {
	if approvers == nil {
		approvers = []actor.Actor{}
	}
	return RoundLevel{Number: n, Approvers: approvers, ApproverType: SourcePeople, Mode: ModeAny, Resolution: ResolutionDecide, Excluded: []Excluded{}}
}

// fixtureFacts are the rule facts of an activity of addActivity: 50,000.00 USD, Event, Lubricants.
func fixtureFacts(t *testing.T) *RuleFacts {
	t.Helper()
	cost := money.USD(mustAmount(t, "50000.00"))
	return &RuleFacts{PlannedCost: &cost, ActivityType: &FactRef{ID: eventType, Name: "Event"},
		BusinessUnits: []FactOption{{ID: lubricantsUnit, Label: "Lubricants"}}}
}
