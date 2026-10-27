package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

var (
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	meiLin = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
	sam    = actor.Actor{ID: "U-1005", Name: "Sam Inactive"} // inactive
)

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The fake Settings' attributes: a campaign's Cluster and Region (needed to publish), an initiative's FIT4.
const (
	clusterID = "SA-1003"
	regionID  = "SA-1004"
	fit4ID    = "SA-1011"
)

// fixture is a campaigns service on in-memory stores and a real history service, with fakes for the other
// features.
type fixture struct {
	service    *Service
	db         *fake.DB
	history    *history.Service
	plans      *fakePlans
	settings   *fakeSettings
	activities *fakeActivities
	now        time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
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
	f.history = history.NewService(db, historytest.NewStore(db), ids.NewGenerator(fake.NewCounters(db)), f.clock)
	tools := history.Tools{Tx: db, IDs: ids.NewGenerator(fake.NewCounters(db)), Log: f.history, Now: f.clock}
	f.service = NewService(tools, newFakeStore(db), Deps{
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

func (f *fixture) clock() time.Time { return f.now }

func (f *fixture) advance(d time.Duration) { f.now = f.now.Add(d) }

func (f *fixture) createCampaign(t *testing.T, a actor.Actor, planID string, in CampaignInput) CampaignView {
	t.Helper()
	view, err := f.service.CreateCampaign(t.Context(), a, planID, in)
	if err != nil {
		t.Fatalf("CreateCampaign(%q): %v", in.Name, err)
	}
	return view
}

func (f *fixture) createInitiative(t *testing.T, a actor.Actor, campaignID string, in InitiativeInput) InitiativeView {
	t.Helper()
	view, err := f.service.CreateInitiative(t.Context(), a, campaignID, in)
	if err != nil {
		t.Fatalf("CreateInitiative(%q): %v", in.Name, err)
	}
	return view
}

func (f *fixture) list(t *testing.T, planID string) PlanCampaigns {
	t.Helper()
	tree, err := f.service.ListPlanCampaigns(t.Context(), priya, planID)
	if err != nil {
		t.Fatalf("ListPlanCampaigns(%s): %v", planID, err)
	}
	return tree
}

func (f *fixture) storedCampaign(t *testing.T, id string) (Campaign, bool) {
	t.Helper()
	c, found, err := newFakeStore(f.db).GetCampaign(t.Context(), id)
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}
	return c, found
}

func (f *fixture) storedInitiative(t *testing.T, id string) (Initiative, bool) {
	t.Helper()
	i, found, err := newFakeStore(f.db).GetInitiative(t.Context(), id)
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

// problems returns the validation error of field, message pairs.
func problems(fieldsAndMessages ...string) *errs.Error {
	var p errs.Problems
	for i := 0; i+1 < len(fieldsAndMessages); i += 2 {
		p.Add(fieldsAndMessages[i], fieldsAndMessages[i+1])
	}
	return errs.Validation(p...)
}

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

func input(amount string) *money.Input {
	return &money.Input{Amount: amount, Currency: "USD"}
}

func moneyInput(amount, currency string) money.Input {
	return money.Input{Amount: amount, Currency: currency}
}

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

// updateOf returns the update that saves campaign c's form unchanged, from version.
func updateOf(c CampaignView, version int) CampaignUpdate {
	var budget *money.Input
	if c.Budget != nil {
		budget = input(c.Budget.Amount.String())
	}
	return CampaignUpdate{
		Name: c.Name, Description: c.Description, OwnerID: c.OwnerID, StartDate: c.StartDate, EndDate: c.EndDate,
		Budget: budget, Attributes: c.Attributes, Version: &version,
	}
}

func initiativeUpdateOf(i InitiativeView, version int) InitiativeUpdate {
	var budget *money.Input
	if i.Budget != nil {
		budget = input(i.Budget.Amount.String())
	}
	return InitiativeUpdate{
		Name: i.Name, Description: i.Description, OwnerID: i.OwnerID, StartDate: i.StartDate, EndDate: i.EndDate,
		Budget: budget, Attributes: i.Attributes, Version: &version,
	}
}

// putStored writes records straight into the store, for states the service would refuse to create.
func (f *fixture) putStored(t *testing.T, campaigns []Campaign, initiatives []Initiative) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		repo := newFakeStore(f.db)
		for _, c := range campaigns {
			if err := repo.PutCampaign(ctx, c); err != nil {
				return err
			}
		}
		for _, i := range initiatives {
			if err := repo.PutInitiative(ctx, i); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
}
