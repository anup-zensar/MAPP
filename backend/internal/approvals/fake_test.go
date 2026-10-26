package approvals

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
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

// The two workflows of the fixture, as the demo data has them (data model §8).
const (
	lubricantsFlow = "WF-1001" // Lubricants EU approval: level 1 Daniel Okoro
	chinaFlow      = "WF-1002" // China BC approval workflow: level 1 Carlos Mendes, level 2 Alex Morgan
)

// fakeActivities is the Activities of the service tests: activities in memory, a ready-to-submit check that
// lists the problems a test sets, and line items a test sets.
type fakeActivities struct {
	clock      *testClock
	facts      map[string]ActivityFacts
	problems   map[string][]SubmitProblem
	missing    map[string][]MissingFieldRef
	unassigned map[string]*money.Money
	lineItems  map[string][]PreviewLineItem
	order      []string // ids in the order they were added
}

func (f *fakeActivities) Activity(_ context.Context, id string) (ActivityFacts, bool, error) {
	facts, found := f.facts[id]
	return facts, found, nil
}

func (f *fakeActivities) List(context.Context) ([]ActivityFacts, error) {
	list := []ActivityFacts{}
	for _, id := range f.order {
		if facts, found := f.facts[id]; found {
			list = append(list, facts)
		}
	}
	return list, nil
}

// ListInScope applies the scope to the plan and the owner; the fake knows no round's approvers.
func (f *fakeActivities) ListInScope(ctx context.Context, scope authz.Scope) ([]ActivityFacts, error) {
	all, err := f.List(ctx)
	return slices.DeleteFunc(all, func(facts ActivityFacts) bool {
		return !scope.SeesActivity(authz.ActivityFacts{Plan: facts.Access, OwnerID: facts.OwnerID})
	}), err
}

// SetStatus changes the status, and the version goes up by one, as the adapter's Touch does.
func (f *fakeActivities) SetStatus(_ context.Context, a actor.Actor, id, status string) (ActivityFacts, error) {
	facts, found := f.facts[id]
	if !found {
		return ActivityFacts{}, fmt.Errorf("fake: activity %s is not stored", id)
	}
	facts.Status, facts.DisplayStatus = status, status
	facts.Touch(a, f.clock.Now())
	f.facts[id] = facts
	return facts, nil
}

func (f *fakeActivities) ReadyCheck(_ context.Context, _ actor.Actor, id string) (ReadyCheck, error) {
	return ReadyCheck{Problems: f.problems[id], UnassignedPlannedCost: f.unassigned[id], Missing: f.missing[id]}, nil
}

func (f *fakeActivities) LineItems(_ context.Context, _ actor.Actor, activityID string) ([]PreviewLineItem, error) {
	if items, found := f.lineItems[activityID]; found {
		return items, nil
	}
	return []PreviewLineItem{}, nil
}

// fixture is a service on a temporary store with the six demo people, the two demo workflows and a fake
// Activities, at a clock that starts at 12 Mar 2026 09:00 UTC.
type fixture struct {
	service    *Service
	activities *fakeActivities
	history    *history.Service
	db         *store.DB
	clock      *testClock
	access     *fakeAccess
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONRepository(), generator, clock.Now)
	fake := &fakeActivities{
		clock: clock, facts: map[string]ActivityFacts{}, problems: map[string][]SubmitProblem{}, missing: map[string][]MissingFieldRef{},
		unassigned: map[string]*money.Money{}, lineItems: map[string][]PreviewLineItem{},
	}
	peopleRepo := people.NewJSONRepository()
	service := NewService(db, NewJSONRepository(), peopleRepo, fake, generator, hist, clock.Now)
	access := &fakeAccess{rule: true, cannotApprove: map[string]bool{}, holders: map[string][]string{}}
	service.UseAccess(access)
	service.UseRuleChoices(fakeChoices{})
	f := &fixture{service: service, activities: fake, history: hist, db: db, clock: clock, access: access}

	// The activity kind of feature 006, as far as the log needs it: its name.
	err = hist.Register(history.Kind{
		Name: activityKind, Singular: "activity", Plural: "activities",
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			facts, found := fake.facts[id]
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

// addActivity stores a Created activity named name, owned by owner, with the workflow workflowID, in the
// fixture's plan, and returns it.
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

// version returns a pointer to v, for the versions of requests.
func version(v int) *int {
	return &v
}

// submit submits (or re-submits) the activity id as a from its current version, and stops the test when
// that fails.
func (f *fixture) submit(t *testing.T, a actor.Actor, id, comment string) ActivityApproval {
	t.Helper()
	approval, err := f.service.Submit(t.Context(), a, id, SubmitInput{Version: version(f.activities.facts[id].Version), Comment: comment})
	if err != nil {
		t.Fatalf("Submit(%s): %v", id, err)
	}
	return approval
}

// read runs fn in a Read of the fixture's store and stops the test when it fails.
func (f *fixture) read(t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	if err := f.db.Read(t.Context(), fn); err != nil {
		t.Fatalf("read: %v", err)
	}
}

// write runs fn in a Write of the fixture's store and stops the test when it fails.
func (f *fixture) write(t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	if err := f.db.Write(t.Context(), fn); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// rounds returns the stored rounds of the activity id, by number.
func (f *fixture) rounds(t *testing.T, id string) []Round {
	t.Helper()
	var rounds []Round
	f.read(t, func(ctx context.Context) (err error) {
		rounds, err = f.service.repo.RoundsOf(ctx, id)
		return err
	})
	return rounds
}

// actions returns the stored history entries of the activity id, oldest first.
func (f *fixture) actions(t *testing.T, id string) []Action {
	t.Helper()
	var actions []Action
	f.read(t, func(ctx context.Context) (err error) {
		actions, err = f.service.repo.ActionsOf(ctx, id)
		return err
	})
	return actions
}

// notifications returns the stored notifications of the person personID, newest first.
func (f *fixture) notifications(t *testing.T, personID string) []Notification {
	t.Helper()
	var list []Notification
	f.read(t, func(ctx context.Context) (err error) {
		list, err = f.service.repo.NotificationsOf(ctx, personID)
		return err
	})
	return list
}

// log returns the activity-log entries about the record id, newest first.
func (f *fixture) log(t *testing.T, id string) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{RecordID: id, Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

// mustAmount parses a decimal text and stops the test when it is not one.
func mustAmount(t *testing.T, text string) money.Amount {
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
		t.Errorf("error = %#v, want %#v", err, want)
	}
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// The rule choices of the fixture (017): two activity types and two business units.
const (
	eventType       = "AT-1001" // Event
	sponsorshipType = "AT-1002" // Sponsorship
	lubricantsUnit  = "SO-1001" // Lubricants
	mobilityUnit    = "SO-1002" // Mobility
)

// fakeChoices is the RuleChoices of the service tests.
type fakeChoices struct{}

func (fakeChoices) ActivityTypes(context.Context) ([]Choice, error) {
	return []Choice{{ID: eventType, Label: "Event", Active: true}, {ID: sponsorshipType, Label: "Sponsorship", Active: true}}, nil
}

func (fakeChoices) BusinessUnits(context.Context) ([]Choice, error) {
	return []Choice{{ID: lubricantsUnit, Label: "Lubricants", Active: true}, {ID: mobilityUnit, Label: "Mobility", Active: true}}, nil
}

// The roles of the fixture's fake access (014 seed names).
var (
	approverRole = RoleInfo{RoleRef: RoleRef{ID: "RL-1004", Name: "Approver", BuiltIn: "approver"}, Approves: true}
	viewerRole   = RoleInfo{RoleRef: RoleRef{ID: "RL-1005", Name: "Viewer", BuiltIn: "viewer"}, Approves: false}
)

// fakeAccess is the Access of the service tests: everyone may approve except cannotApprove; everyone may
// decide on behalf and manage workflows unless the lists are set; the access rule is on unless a test turns
// it off; two roles and their holders by role id.
type fakeAccess struct {
	rule          bool
	cannotApprove map[string]bool
	onBehalf      []string // nil: everyone may decide on behalf
	managers      []string // nil: everyone holds workflows.manage
	holders       map[string][]string
}

func (f *fakeAccess) CanApprove(_ context.Context, personIDs []string) (map[string]bool, error) {
	can := map[string]bool{}
	for _, id := range personIDs {
		can[id] = !f.cannotApprove[id]
	}
	return can, nil
}

func (f *fakeAccess) Check(_ context.Context, personID string, p Permission, _ *PlanFacts) (Grant, error) {
	var list []string
	switch p {
	case PermissionDecide:
		if f.cannotApprove[personID] {
			return Grant{Refusal: "Approving activities isn't part of your role (Viewer). Ask an administrator if you need it."}, nil
		}
		return Grant{Allowed: true}, nil
	case PermissionDecideOnBehalf:
		list = f.onBehalf
	default:
		list = f.managers
	}
	if list == nil || slices.Contains(list, personID) {
		return Grant{Allowed: true}, nil
	}
	return Grant{Refusal: "Changing approval workflows isn't part of your role (Planner). Ask an administrator if you need it."}, nil
}

func (f *fakeAccess) NoSelfApproval(context.Context) (bool, error) { return f.rule, nil }

func (f *fakeAccess) Roles(context.Context) ([]RoleInfo, error) {
	return []RoleInfo{approverRole, viewerRole}, nil
}

func (f *fakeAccess) RoleHolders(_ context.Context, roleID string, _ PlanFacts) ([]string, error) {
	return f.holders[roleID], nil
}

// setManager makes manager the line manager of the person p in the fixture's store.
func (f *fixture) setManager(t *testing.T, p, manager actor.Actor) {
	t.Helper()
	f.write(t, func(ctx context.Context) error {
		person, _, err := f.service.people.Get(ctx, p.ID)
		if err != nil {
			return err
		}
		person.ManagerID = manager.ID
		return f.service.people.Put(ctx, person)
	})
}

// decideLevel is a round level whose approvers decide, as Submit stores one of named people where any one
// approves.
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
