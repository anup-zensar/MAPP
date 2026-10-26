package activities_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/activities"
	"mapp/internal/approvals"
	"mapp/internal/campaigns"
	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
)

// The service tests run on the demo data of the seed (006 plan "Demo data"): plan P-1001 "FY2026 Lubricants
// Europe" (business unit Lubricants, model "Lubricants 1.0" — 2.00 %, 5.00 %, 10,000.00 USD), its
// initiatives I-1001 "Digital & social" (1 Jan – 30 Jun 2026, owner Priya Raman), I-1002 "Print & outdoor"
// (1 Jan – 31 Dec 2026) and I-1004 "Retail events" (1 Jun – 31 Dec 2026); plan P-1002 "FY2026 Mobility
// China" without a business unit; activities A-1001 … A-1010.

// The people of these tests.
var (
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The demo records the tests use by ID.
const (
	digitalAndSocial = "I-1001"
	printAndOutdoor  = "I-1002"
	retailEvents     = "I-1004"
	fleetCards       = "I-1006" // Mobility China, no dates
	contentType      = "AT-1004"
	eventType        = "AT-1002"
	mobilityType     = "AT-1006" // not allowed by Lubricants 1.0
	lubricantsFlow   = "WF-1001"
	chinaFlow        = "WF-1002"
	countrySplit     = "SA-1013"
	companyCode      = "SA-1014"
	brands           = "SA-1022" // Lubricants group
)

// fixture is the activities service on a temporary store holding the demo data, with fake line items
// (007) and approval hooks (008).
type fixture struct {
	service   *activities.Service
	history   *history.Service
	db        *store.DB
	clock     *testClock
	lineItems *fakeLineItems
	hooks     *fakeHooks
}

// newFixture copies the demo data into a temporary store and builds the service as internal/app does.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	apitest.CopyDemoData(t, dir, true)
	db, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONRepository(), generator, clock.Now)
	campaignsRepo := campaigns.NewJSONRepository()
	service := activities.NewService(db, activities.NewJSONRepository(), generator, hist, clock.Now, activities.Deps{
		Plans: plans.NewJSONRepository(), Hierarchy: campaignsRepo,
		Settings: settings.NewService(db, settings.NewJSONRepository(), generator, hist, clock.Now), People: people.NewJSONRepository(),
		Workflows: activities.NewSeededWorkflows(approvals.NewJSONRepository()),
	})
	for _, kind := range append([]history.Kind{initiativeKind(campaignsRepo)}, service.Kinds()...) {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	f := &fixture{service: service, history: hist, db: db, clock: clock,
		lineItems: &fakeLineItems{facts: map[string]activities.LineItemFacts{}}, hooks: &fakeHooks{}}
	service.UseLineItems(f.lineItems)
	service.UseApprovalHooks(f.hooks)
	return f
}

// initiativeKind is feature 005's initiative kind as the tests need it: the parent of activities.
func initiativeKind(repo campaigns.Repository) history.Kind {
	unused := errors.New("not used by these tests")
	return history.Kind{
		Name: "initiative", Singular: "initiative", Plural: "initiatives",
		Snapshot: func(ctx context.Context, id string) (history.Snapshot, bool, error) {
			i, found, err := repo.GetInitiative(ctx, id)
			return history.Snapshot{ID: i.ID, Name: i.Name}, found, err
		},
		Remove: func(context.Context, string) error { return unused },
		Put:    func(context.Context, history.Snapshot) error { return unused },
	}
}

// create creates an activity as priya and stops the test when that fails.
func (f *fixture) create(t *testing.T, initiativeID string, in activities.Input) activities.View {
	t.Helper()
	view, err := f.service.Create(t.Context(), priya, initiativeID, in)
	if err != nil {
		t.Fatalf("Create(%s, %q): %v", initiativeID, in.Name, err)
	}
	return view
}

// get reads an activity as priya and stops the test when that fails.
func (f *fixture) get(t *testing.T, id string) activities.View {
	t.Helper()
	view, err := f.service.Get(t.Context(), priya, id)
	if err != nil {
		t.Fatalf("Get(%s): %v", id, err)
	}
	return view
}

// edit reads the activity id, lets change edit its form, and saves it as a, from the version it read.
func (f *fixture) edit(t *testing.T, a actor.Actor, id string, change func(u *activities.Update)) (activities.View, error) {
	t.Helper()
	update := formOf(f.get(t, id))
	change(&update)
	return f.service.Update(t.Context(), a, id, update)
}

// mustEdit is edit that stops the test when the save fails.
func (f *fixture) mustEdit(t *testing.T, id string, change func(u *activities.Update)) activities.View {
	t.Helper()
	view, err := f.edit(t, priya, id, change)
	if err != nil {
		t.Fatalf("Update(%s): %v", id, err)
	}
	return view
}

// formOf is the whole editable state of an activity, as the edit form sends it (D-107).
func formOf(v activities.View) activities.Update {
	update := activities.Update{
		Name: v.Name, ExternalName: v.ExternalName, ActivityTypeID: v.ActivityType.ID, Ongoing: v.Ongoing,
		Description: v.Description, Attributes: settings.AttributeValues{}, Version: &v.Version,
		Goals: activities.GoalsInput{INQ: v.Goals.INQ, SetLaterStagesMyself: v.Goals.SetLaterStagesMyself},
	}
	for id, value := range v.Attributes {
		update.Attributes[id] = value
	}
	if v.Owner != nil {
		update.OwnerID = &v.Owner.ID
	}
	if v.StartDate != nil {
		text := v.StartDate.String()
		update.StartDate = &text
	}
	if v.EndDate != nil {
		text := v.EndDate.String()
		update.EndDate = &text
	}
	if v.PlannedCost != nil {
		update.PlannedCost = &money.Input{Amount: v.PlannedCost.Amount.String(), Currency: v.PlannedCost.Currency}
	}
	if v.ApprovalWorkflow != nil {
		update.ApprovalWorkflowID = &v.ApprovalWorkflow.ID
	}
	if v.Goals.SetLaterStagesMyself {
		update.Goals.MQL = v.Goals.MQL
		if v.Goals.Revenue != nil {
			update.Goals.Revenue = &money.Input{Amount: v.Goals.Revenue.Amount.String(), Currency: "USD"}
		}
	}
	return update
}

// entries returns the activity-log entries about activities, newest first.
func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	list, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{RecordType: "activity", Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return list
}

// stored reads the activity id as stored, and false when it is not; its attribute values come compacted, as
// a save writes them, because the copied demo files hold them indented.
func (f *fixture) stored(t *testing.T, id string) (activities.Activity, bool) {
	t.Helper()
	var activity activities.Activity
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		activity, found, err = activities.NewJSONRepository().Get(ctx, id)
		return err
	})
	if err != nil {
		t.Fatalf("Get(%s): %v", id, err)
	}
	for attribute, value := range activity.Attributes {
		activity.Attributes[attribute], _ = json.Marshal(value)
	}
	return activity, found
}

// change changes demo data directly, as other features would (deactivate a type, move an initiative's
// dates, …), and stops the test when that fails.
func (f *fixture) change(t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	if err := f.db.Write(t.Context(), fn); err != nil {
		t.Fatalf("changing the demo data: %v", err)
	}
}

// deactivateType makes the activity type id inactive.
func (f *fixture) deactivateType(t *testing.T, id string) {
	t.Helper()
	repo := settings.NewJSONRepository()
	f.change(t, func(ctx context.Context) error {
		activityType, _, err := repo.GetActivityType(ctx, id)
		if err != nil {
			return err
		}
		activityType.Active = false
		return repo.PutActivityType(ctx, activityType)
	})
}

// deactivateOption makes the Shell attribute option id inactive.
func (f *fixture) deactivateOption(t *testing.T, id string) {
	t.Helper()
	repo := settings.NewJSONRepository()
	f.change(t, func(ctx context.Context) error {
		option, _, err := repo.GetOption(ctx, id)
		if err != nil {
			return err
		}
		option.Active = false
		return repo.PutOption(ctx, option)
	})
}

// deactivateWorkflow makes the approval workflow id inactive.
func (f *fixture) deactivateWorkflow(t *testing.T, id string) {
	t.Helper()
	repo := approvals.NewJSONRepository()
	f.change(t, func(ctx context.Context) error {
		workflow, _, err := repo.GetWorkflow(ctx, id)
		if err != nil {
			return err
		}
		workflow.Active = false
		return repo.PutWorkflow(ctx, workflow)
	})
}

// deactivatePerson makes the person id inactive.
func (f *fixture) deactivatePerson(t *testing.T, id string) {
	t.Helper()
	repo := people.NewJSONRepository()
	f.change(t, func(ctx context.Context) error {
		person, _, err := repo.Get(ctx, id)
		if err != nil {
			return err
		}
		person.Active = false
		return repo.Put(ctx, person)
	})
}

// fakeLineItems are feature 007's line items as the tests set them.
type fakeLineItems struct {
	facts        map[string]activities.LineItemFacts
	datesChanged []string                              // the activities whose dates changed, in order
	copies       []string                              // the CopyAll calls, "<from> → <to> +<years>", in order
	items        map[string][]activities.LineItemBrief // 016: the line items of each activity, for Split
	moves        []string                              // 016: the MoveToActivity calls, "<ids> → <to>"
}

func (l *fakeLineItems) Facts(_ context.Context, activityIDs []string) (map[string]activities.LineItemFacts, error) {
	facts := map[string]activities.LineItemFacts{}
	for _, id := range activityIDs {
		if f, found := l.facts[id]; found {
			facts[id] = f
		}
	}
	return facts, nil
}

func (l *fakeLineItems) CopyMany(ctx context.Context, by actor.Actor, pairs []activities.CopyPair, years int) (int, error) {
	copied := 0
	for _, p := range pairs {
		n, _ := l.CopyAll(ctx, by, p.FromID, p.ToID, years)
		copied += n
	}
	return copied, nil
}

// CopyAll records the call and answers the activity's line-item count of facts.
func (l *fakeLineItems) CopyAll(_ context.Context, _ actor.Actor, from, to string, years int) (int, error) {
	l.copies = append(l.copies, fmt.Sprintf("%s → %s +%d", from, to, years))
	return l.facts[from].Count, nil
}

func (l *fakeLineItems) DatesChanged(_ context.Context, _ actor.Actor, _, after activities.Activity) error {
	l.datesChanged = append(l.datesChanged, after.ID)
	return nil
}

// Items answers the line items the test set for the activity (016 Split).
func (l *fakeLineItems) Items(_ context.Context, activityID string) ([]activities.LineItemBrief, error) {
	return append([]activities.LineItemBrief{}, l.items[activityID]...), nil
}

// MoveToActivity records the move, "<ids> → <activity>", and moves the line items between the activities'
// facts and items as the real line items would.
func (l *fakeLineItems) MoveToActivity(_ context.Context, _ actor.Actor, ids []string, to string) error {
	l.moves = append(l.moves, fmt.Sprintf("%v → %s", ids, to))
	if l.items == nil {
		return nil
	}
	for from, items := range l.items {
		kept := []activities.LineItemBrief{}
		for _, li := range items {
			if slices.Contains(ids, li.ID) && from != to {
				l.items[to] = append(l.items[to], li)
				l.shift(from, to, li.PlannedCostPlan)
				continue
			}
			kept = append(kept, li)
		}
		l.items[from] = kept
	}
	return nil
}

// shift moves one line item's count and plan-currency amount from one activity's facts to another's.
func (l *fakeLineItems) shift(from, to string, amount money.Money) {
	if l.facts == nil {
		l.facts = map[string]activities.LineItemFacts{}
	}
	f := l.facts[from]
	f.Count--
	f.PlannedCostPlan = f.PlannedCostPlan.Sub(amount.Amount)
	l.facts[from] = f
	t := l.facts[to]
	t.Count++
	t.PlannedCostPlan = t.PlannedCostPlan.Add(amount.Amount)
	t.Currency = amount.Currency
	l.facts[to] = t
}

// fakeHooks record what feature 008 is told.
type fakeHooks struct {
	restarted []string // "<id>: <before status> → <after status>"
	changes   []string // the changes of every restart, "<label>: <old text> → <new text>"
	deleting  []string
	restored  []string
}

func (h *fakeHooks) ApprovalRestarted(_ context.Context, _ actor.Actor, before, after activities.Activity, changes []history.Change) error {
	h.restarted = append(h.restarted, after.ID+": "+string(before.Status)+" → "+string(after.Status))
	for _, c := range changes {
		h.changes = append(h.changes, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	return nil
}

func (h *fakeHooks) ActivityDeleting(_ context.Context, _ actor.Actor, a activities.Activity) error {
	h.deleting = append(h.deleting, a.ID)
	return nil
}

func (h *fakeHooks) ActivityRestored(_ context.Context, _ history.RestoreContext, id string) ([]string, error) {
	h.restored = append(h.restored, id)
	return nil, nil
}

// assertError checks that err is the API error want.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %v\n  %#v\nwant %#v", err, got, want)
	}
}

// problems returns the API error with the field problems given as field, message, field, message, ….
func problems(fieldsAndMessages ...string) *errs.Error {
	var list []errs.FieldError
	for i := 0; i+1 < len(fieldsAndMessages); i += 2 {
		list = append(list, errs.FieldError{Field: fieldsAndMessages[i], Message: fieldsAndMessages[i+1]})
	}
	return errs.Validation(list...)
}

// text returns a pointer to s, for the optional fields of requests.
func text(s string) *string {
	return &s
}

// usdInput is an amount in US dollars as a request sends it.
func usdInput(amount string) *money.Input {
	return &money.Input{Amount: amount, Currency: "USD"}
}

// split is a % split value: option id, share, option id, share, ….
func split(optionsAndShares ...string) json.RawMessage {
	var shares []map[string]string
	for i := 0; i+1 < len(optionsAndShares); i += 2 {
		shares = append(shares, map[string]string{"optionId": optionsAndShares[i], "share": optionsAndShares[i+1]})
	}
	encoded, _ := json.Marshal(shares)
	return encoded
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }
