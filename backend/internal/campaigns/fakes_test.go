package campaigns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
)

// The people of these tests (Sam is inactive).
var (
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	meiLin = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
	sam    = actor.Actor{ID: "U-1005", Name: "Sam Inactive"}
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC (today is 12 Mar 2026).
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The attribute ids of the fake Settings: a campaign's Cluster and Region (needed to publish) and an
// initiative's FIT4 flag.
const (
	clusterID = "SA-1003"
	regionID  = "SA-1004"
	fit4ID    = "SA-1011"
)

// fixture is a campaigns service on a real JSON store and a real history service, with fakes for the other
// features.
type fixture struct {
	service    *Service
	db         *store.DB
	history    *history.Service
	plans      *fakePlans
	settings   *fakeSettings
	activities *fakeActivities
	now        time.Time
}

// newFixture builds the service with the plans P-1001 (2026, budget 1,200,000.00, Priya), P-1002 (2026,
// budget 800,000.00, another model and business unit) and P-1003 (2027, no budget), and the people above.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	f := &fixture{
		db: db,
		plans: &fakePlans{plans: map[string]PlanFacts{
			"P-1001": {ID: "P-1001", Name: "FY2026 Lubricants Europe", Year: 2026, Status: "published", OwnerID: ptr("U-1002"),
				Budget: usd("1200000.00"), Currency: "USD", ModelID: "MOD-1001", Attributes: settings.AttributeValues{"SA-1001": json.RawMessage(`["SO-1002"]`)}},
			"P-1002": {ID: "P-1002", Name: "FY2026 Mobility China", Year: 2026, Status: "draft", OwnerID: ptr("U-1004"),
				Budget: usd("800000.00"), Currency: "USD", ModelID: "MOD-1002", Attributes: settings.AttributeValues{"SA-1001": json.RawMessage(`["SO-1001"]`)}},
			"P-1003": {ID: "P-1003", Name: "FY2027 Global Brand", Year: 2027, Status: "draft", OwnerID: ptr("U-1001"),
				Currency: "USD", ModelID: "MOD-1001", Attributes: settings.AttributeValues{}},
		}},
		settings:   &fakeSettings{problems: map[settings.Level][]errs.FieldError{}},
		activities: &fakeActivities{byInitiative: map[string][]ActivitySummary{}},
		now:        start,
	}
	f.history = history.NewService(db, history.NewJSONStore(), ids.NewGenerator(ids.NewJSONCounters()), f.clock)
	tools := history.Tools{Tx: db, IDs: ids.NewGenerator(ids.NewJSONCounters()), Log: f.history, Now: f.clock}
	f.service = NewService(tools, NewJSONStore(), Deps{
		Plans: f.plans, People: fakePeople{}, Settings: f.settings, Activities: f.activities,
	})
	kinds := append([]history.Kind{f.plans.kind()}, f.service.Kinds()...)
	kinds = append(kinds, f.activities.kind())
	for _, k := range kinds {
		if err := f.history.Register(k); err != nil {
			t.Fatalf("Register %s: %v", k.Name, err)
		}
	}
	return f
}

// clock is the services' clock: the time the test has reached.
func (f *fixture) clock() time.Time { return f.now }

// advance moves the clock forward by d.
func (f *fixture) advance(d time.Duration) { f.now = f.now.Add(d) }

// createCampaign creates a campaign in planID or stops the test.
func (f *fixture) createCampaign(t *testing.T, a actor.Actor, planID string, in CampaignInput) CampaignView {
	t.Helper()
	view, err := f.service.CreateCampaign(t.Context(), a, planID, in)
	if err != nil {
		t.Fatalf("CreateCampaign(%q): %v", in.Name, err)
	}
	return view
}

// createInitiative creates an initiative in campaignID or stops the test.
func (f *fixture) createInitiative(t *testing.T, a actor.Actor, campaignID string, in InitiativeInput) InitiativeView {
	t.Helper()
	view, err := f.service.CreateInitiative(t.Context(), a, campaignID, in)
	if err != nil {
		t.Fatalf("CreateInitiative(%q): %v", in.Name, err)
	}
	return view
}

// list returns the campaign tree of planID or stops the test.
func (f *fixture) list(t *testing.T, planID string) PlanCampaigns {
	t.Helper()
	tree, err := f.service.ListPlanCampaigns(t.Context(), priya, planID)
	if err != nil {
		t.Fatalf("ListPlanCampaigns(%s): %v", planID, err)
	}
	return tree
}

// storedCampaign reads the campaign id as it is stored.
func (f *fixture) storedCampaign(t *testing.T, id string) (Campaign, bool) {
	t.Helper()
	var c Campaign
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		c, found, err = NewJSONStore().GetCampaign(ctx, id)
		return err
	})
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}
	return c, found
}

// storedInitiative reads the initiative id as it is stored.
func (f *fixture) storedInitiative(t *testing.T, id string) (Initiative, bool) {
	t.Helper()
	var i Initiative
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		i, found, err = NewJSONStore().GetInitiative(ctx, id)
		return err
	})
	if err != nil {
		t.Fatalf("GetInitiative: %v", err)
	}
	return i, found
}

// log returns the activity-log entries about the record id, newest first.
func (f *fixture) log(t *testing.T, recordType, id string) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{RecordType: recordType, RecordID: id, Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

// assertError fails the test unless err is an *errs.Error equal to want.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) {
		t.Fatalf("error = %v, want %+v", err, want)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("error =\n%+v %+v\nwant\n%+v %+v", got, got.Fields, want, want.Fields)
	}
}

// problems returns the validation error of the field problems given as field, message pairs.
func problems(fieldsAndMessages ...string) *errs.Error {
	var p errs.Problems
	for i := 0; i+1 < len(fieldsAndMessages); i += 2 {
		p.Add(fieldsAndMessages[i], fieldsAndMessages[i+1])
	}
	return errs.Validation(p...)
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }

// usd returns the USD amount text as money; "" gives nil.
func usd(text string) *money.Money {
	if text == "" {
		return nil
	}
	amount, err := money.ParseAmount(text)
	if err != nil {
		panic(err)
	}
	m := money.USD(amount)
	return &m
}

// input returns a USD money input.
func input(amount string) *money.Input {
	return &money.Input{Amount: amount, Currency: "USD"}
}

// moneyInput returns a money input in currency.
func moneyInput(amount, currency string) money.Input {
	return money.Input{Amount: amount, Currency: currency}
}

// day returns a pointer to the date "YYYY-MM-DD".
func day(text string) *date.Date {
	d := date.MustParse(text)
	return &d
}

// amountText writes an optional amount as its digits, "" for nil.
func amountText(m *money.Money) string {
	if m == nil {
		return ""
	}
	return m.Amount.String()
}

// fakePlans is the PlanSource of the tests, and the plan kind history needs as the parent of campaigns.
type fakePlans struct {
	plans map[string]PlanFacts
}

func (p *fakePlans) Plan(_ context.Context, id string) (PlanFacts, bool, error) {
	plan, found := p.plans[id]
	return plan, found, nil
}

func (p *fakePlans) PlansOfYear(_ context.Context, year int) ([]PlanFacts, error) {
	list := []PlanFacts{}
	for _, plan := range p.plans {
		if plan.Year == year {
			list = append(list, plan)
		}
	}
	slices.SortFunc(list, func(a, b PlanFacts) int { return strings.Compare(a.ID, b.ID) })
	return list, nil
}

// kind is a minimal history kind "plan": plans are never deleted in these tests.
func (p *fakePlans) kind() history.Kind {
	return history.Kind{
		Name: planKind, Singular: "plan", Plural: "plans",
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			plan, found := p.plans[id]
			return history.Snapshot{ID: plan.ID, Name: plan.Name, Data: json.RawMessage(`{}`)}, found, nil
		},
		Remove: func(context.Context, string) error { return nil },
		Put:    func(context.Context, history.Snapshot) error { return nil },
	}
}

// fakePeople knows the five people of these tests; Sam is inactive.
type fakePeople struct{}

func (fakePeople) Get(_ context.Context, id string) (people.Person, bool, error) {
	for _, a := range []actor.Actor{alex, priya, daniel, meiLin, sam} {
		if a.ID == id {
			return people.Person{ID: a.ID, Name: a.Name, JobTitle: "Marketer", Active: a != sam}, true, nil
		}
	}
	return people.Person{}, false, nil
}

// fakeSettings is the Settings of the tests: the campaign level has Cluster and Region (needed to publish,
// Region with no options for the cluster "SO-NONE"), the problems to report per level are set by a test,
// and attributes are described as "Label <id>".
type fakeSettings struct {
	problems map[settings.Level][]errs.FieldError
}

func (s *fakeSettings) CheckAttributes(_ context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error) {
	return settings.AttributeResult{Problems: s.problems[level], Values: in.Values}, nil
}

func (s *fakeSettings) MissingAttributes(_ context.Context, level settings.Level, values settings.AttributeValues, _ []string) ([]settings.MissingField, error) {
	missing := []settings.MissingField{}
	if level != settings.LevelCampaign {
		return missing, nil
	}
	if _, set := values[clusterID]; !set {
		missing = append(missing, settings.MissingField{Field: "attributes." + clusterID, Label: "Cluster"})
	}
	_, regionSet := values[regionID]
	if !regionSet && string(values[clusterID]) != `["SO-NONE"]` { // a cluster without regions: nothing to choose (D-034)
		missing = append(missing, settings.MissingField{Field: "attributes." + regionID, Label: "Region"})
	}
	return missing, nil
}

func (s *fakeSettings) BusinessUnit(_ context.Context, planValues settings.AttributeValues) ([]string, error) {
	var ids []string
	_ = json.Unmarshal(planValues["SA-1001"], &ids)
	return ids, nil
}

func (s *fakeSettings) DescribeAttribute(_ context.Context, field string, value any) (string, string, error) {
	id, found := strings.CutPrefix(field, "attributes.")
	if !found {
		return "", "", nil
	}
	if list, ok := value.([]any); ok {
		texts := make([]string, 0, len(list))
		for _, v := range list {
			texts = append(texts, "Option "+v.(string))
		}
		return "Label " + id, strings.Join(texts, ", "), nil
	}
	return "Label " + id, "", nil
}

// Rates are the rates of the 016 examples: EUR 0.92, GBP 0.79 per USD.
func (s *fakeSettings) Rates(context.Context) (money.Rates, error) {
	return money.Rates{"USD": money.One, "EUR": money.RateFromMicros(920_000), "GBP": money.RateFromMicros(790_000)}, nil
}

// Model knows Lubricants 1.0 (MOD-1001: Content production and Event) and Mobility 1.0 (MOD-1002: Event and
// Fleet promotion).
func (s *fakeSettings) Model(_ context.Context, id string) (settings.Model, bool, error) {
	models := map[string]settings.Model{
		"MOD-1001": {ID: "MOD-1001", Name: "Lubricants 1.0", ActivityTypeIDs: []string{"AT-1001", "AT-1002"}, Active: true},
		"MOD-1002": {ID: "MOD-1002", Name: "Mobility 1.0", ActivityTypeIDs: []string{"AT-1002", "AT-1003"}, Active: true},
	}
	m, found := models[id]
	return m, found, nil
}

// ActivityType knows the three activity types of the models.
func (s *fakeSettings) ActivityType(_ context.Context, id string) (settings.ActivityType, bool, error) {
	names := map[string]string{"AT-1001": "Content production", "AT-1002": "Event", "AT-1003": "Fleet promotion"}
	name, found := names[id]
	return settings.ActivityType{ID: id, Name: name, Active: true}, found, nil
}

// Attributes are the Shell attributes of the levels below a plan that the business unit shows: Brands
// (activities) and Spend type (line items) with Lubricants (SO-1002), Fleet segment (activities) with
// Mobility (SO-1001), an inactive Lubricants one, and Cluster and Region, always shown.
func (s *fakeSettings) Attributes(_ context.Context, level settings.Level) ([]settings.Attribute, error) {
	all := []settings.Attribute{
		{ID: clusterID, Label: "Cluster", Level: settings.LevelCampaign, Active: true},
		{ID: regionID, Label: "Region", Level: settings.LevelCampaign, Active: true},
		{ID: "SA-1022", Label: "Brands", Level: settings.LevelActivity, Condition: []string{"SO-1002"}, Active: true},
		{ID: "SA-1023", Label: "Fleet segment", Level: settings.LevelActivity, Condition: []string{"SO-1001"}, Active: true},
		{ID: "SA-1024", Label: "Old brand code", Level: settings.LevelActivity, Condition: []string{"SO-1002"}, Active: false},
		{ID: "SA-1031", Label: "Spend type", Level: settings.LevelLineItem, Condition: []string{"SO-1002"}, Active: true},
	}
	ofLevel := []settings.Attribute{}
	for _, a := range all {
		if a.Level == level {
			ofLevel = append(ofLevel, a)
		}
	}
	return ofLevel, nil
}

// Option knows the Business unit options Mobility (SO-1001) and Lubricants (SO-1002), in that order.
func (s *fakeSettings) Option(_ context.Context, id string) (settings.Option, bool, error) {
	options := map[string]settings.Option{
		"SO-1001": {ID: "SO-1001", AttributeID: "SA-1001", Label: "Mobility", Position: 1, Active: true},
		"SO-1002": {ID: "SO-1002", AttributeID: "SA-1001", Label: "Lubricants", Position: 2, Active: true},
	}
	o, found := options[id]
	return o, found, nil
}

// fakeActivities holds activity summaries per initiative; tests change them to see the roll-ups follow.
// Its history kind lets deletes take activities with their initiative.
type fakeActivities struct {
	byInitiative map[string][]ActivitySummary
	copies       []string // the CopyAll calls, "<from> → <to> +<years>"
	copyErr      error    // what CopyAll answers, when set
}

func (a *fakeActivities) Summaries(_ context.Context, initiativeIDs []string) ([]ActivitySummary, error) {
	list := []ActivitySummary{}
	for _, id := range initiativeIDs {
		list = append(list, a.byInitiative[id]...)
	}
	return list, nil
}

func (a *fakeActivities) CopyMany(ctx context.Context, by actor.Actor, pairs []CopyPair, years int) (ActivityCopyCounts, error) {
	var counts ActivityCopyCounts
	for _, p := range pairs {
		copied, err := a.CopyAll(ctx, by, p.FromID, p.ToID, years)
		if err != nil {
			return ActivityCopyCounts{}, err
		}
		counts.Activities += copied.Activities
		counts.LineItems += copied.LineItems
	}
	return counts, nil
}

// CopyAll copies the summaries of fromInitiativeID into toInitiativeID with new IDs "<id>-copy", records the
// call and counts the copies and their line items; copyErr, when set, makes it fail instead.
func (a *fakeActivities) CopyAll(_ context.Context, _ actor.Actor, from, to string, years int) (ActivityCopyCounts, error) {
	if a.copyErr != nil {
		return ActivityCopyCounts{}, a.copyErr
	}
	a.copies = append(a.copies, fmt.Sprintf("%s → %s +%d", from, to, years))
	var counts ActivityCopyCounts
	for _, s := range slices.Clone(a.byInitiative[from]) {
		s.ID, s.InitiativeID = s.ID+"-copy", to
		a.add(s)
		counts.Activities++
		counts.LineItems += s.LineItems
	}
	return counts, nil
}

// add puts activity summaries under their initiatives.
func (a *fakeActivities) add(summaries ...ActivitySummary) {
	for _, s := range summaries {
		a.byInitiative[s.InitiativeID] = append(a.byInitiative[s.InitiativeID], s)
	}
}

// find returns the activity id.
func (a *fakeActivities) find(id string) (ActivitySummary, bool) {
	for _, list := range a.byInitiative {
		for _, s := range list {
			if s.ID == id {
				return s, true
			}
		}
	}
	return ActivitySummary{}, false
}

// kind is a fake history kind "activity" under the initiative.
func (a *fakeActivities) kind() history.Kind {
	return history.Kind{
		Name: "activity", Singular: "activity", Plural: "activities", Parent: initiativeKind,
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			s, found := a.find(id)
			if !found {
				return history.Snapshot{}, false, nil
			}
			data, err := json.Marshal(s)
			return history.Snapshot{ID: s.ID, Name: s.Name, ParentID: s.InitiativeID, Data: data}, true, err
		},
		ChildIDs: func(_ context.Context, initiativeID string) ([]string, error) {
			list := []string{}
			for _, s := range a.byInitiative[initiativeID] {
				list = append(list, s.ID)
			}
			return list, nil
		},
		Remove: func(_ context.Context, id string) error {
			s, _ := a.find(id)
			a.byInitiative[s.InitiativeID] = slices.DeleteFunc(a.byInitiative[s.InitiativeID], func(x ActivitySummary) bool { return x.ID == id })
			return nil
		},
		Put: func(_ context.Context, snapshot history.Snapshot) error {
			var s ActivitySummary
			if err := json.Unmarshal(snapshot.Data, &s); err != nil {
				return err
			}
			a.add(s)
			return nil
		},
	}
}
