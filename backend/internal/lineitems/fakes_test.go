package lineitems

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
)

// The people of these tests.
var (
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	lena   = actor.Actor{ID: "U-1008", Name: "Lena Vogel"} // shows amounts in EUR (003 FR-023)
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The activities of these tests, in the shape of the seed (006): Launch film is the spec's example.
const (
	launchFilm  = "A-1001" // Created, 120,000.00 USD, 1 Mar – 31 May 2026, owner Priya Raman
	socialOnAir = "A-1002" // Approved, 80,000.00 USD, 1 Jan – 30 Jun 2026
	tradeAds    = "A-1003" // Submitted, 30,000.00 USD, 1 Apr – 30 Sep 2026
	partnerFund = "A-1004" // Approved, ongoing from 1 Mar 2026, 100,000.00 USD
	lunarPromo  = "A-1005" // Created, no dates, no planned cost, no owner
	retailEvent = "A-1006" // Created, its plan's default finance budget has a single active item
)

// The finance budget items of these tests (002).
const (
	europeDigital = "FBI-1001" // 2026 Lubricants Budget
	europeEvents  = "FBI-1002"
	europeBrand   = "FBI-1003"
	europeLegacy  = "FBI-1004" // inactive
	onlyItem      = "FBI-1005" // 2026 Retail Budget: its only item
	lastYearItem  = "FBI-1006" // 2025 Lubricants Budget › Europe – Digital
)

// The line-item Shell attributes of these tests: Spend type and GL code are needed to submit.
const (
	spendType = "SA-1031"
	glCode    = "SA-1032"
)

// fixture is a line-items service on a temporary store with fakes of the other features, a fixed clock and a
// real history service.
type fixture struct {
	service    *Service
	db         *store.DB
	history    *history.Service
	activities *fakeActivities
	currencies *fakeCurrencies
	attributes *fakeAttributes
}

// newTestService builds the fixture: the fake kind "activity" is registered before "lineItem", as
// internal/app registers 006's.
func newTestService(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	now := func() time.Time { return start }
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONRepository(), generator, now)
	f := &fixture{
		db: db, history: hist, activities: newFakeActivities(), currencies: newFakeCurrencies(), attributes: &fakeAttributes{},
	}
	deps := Deps{
		Activities: f.activities, Currencies: f.currencies, FinanceBudgets: fakeBudgets{},
		Attributes: f.attributes, People: fakePeople{},
	}
	f.service = NewService(db, NewJSONRepository(), generator, hist, deps, now)
	if err := hist.Register(f.activities.kind()); err != nil {
		t.Fatalf("Register(activity): %v", err)
	}
	for _, kind := range f.service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return f
}

// create adds a line item and stops the test when that fails.
func (f *fixture) create(t *testing.T, a actor.Actor, activityID string, in Input) View {
	t.Helper()
	view, err := f.service.Create(t.Context(), a, activityID, in)
	if err != nil {
		t.Fatalf("Create(%s): %v", in.Name, err)
	}
	return view
}

// stored returns every stored line item.
func (f *fixture) stored(t *testing.T) []LineItem {
	t.Helper()
	var items []LineItem
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		items, err = NewJSONRepository().List(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return items
}

// log returns the activity-log entries, newest first.
func (f *fixture) log(t *testing.T) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

// input is a line-item form with a name, a currency and a planned cost, the activity's owner Priya, spread
// evenly, the finance budget year 2026 and Request SWBS code from GSAP on.
func input(name, currency, cost string) Input {
	year := 2026
	return Input{
		Name: name, OwnerID: text(priya.ID), Currency: currency, PlannedCost: text(cost),
		Spread: SpreadInput{Method: SpreadEvenly, Months: []SpreadMonthInput{}}, Attributes: settings.AttributeValues{},
		RequestSwbs: true, FinanceBudgetYear: &year,
	}
}

// complete returns in with a spend type, a GL code and a finance budget item: nothing is missing.
func complete(in Input) Input {
	in.Attributes = settings.AttributeValues{spendType: json.RawMessage(`"SO-1101"`), glCode: json.RawMessage(`"SO-1201"`)}
	in.FinanceBudgetItemID = text(europeDigital)
	return in
}

// text returns a pointer to s, for the optional fields of a form.
func text(s string) *string {
	return &s
}

// amount returns the amount of a text known to be right, such as "46000.00".
func amount(s string) money.Amount {
	a, err := money.ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return a
}

// day returns a pointer to the date of a text known to be right.
func day(s string) *date.Date {
	d := date.MustParse(s)
	return &d
}

// assertError checks that err is exactly want: the same code, message and field problems.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if got.Code != want.Code || got.Message != want.Message || !reflect.DeepEqual(got.Fields, want.Fields) {
		t.Errorf("error = %s %q %+v\nwant    %s %q %+v", got.Code, got.Message, got.Fields, want.Code, want.Message, want.Fields)
	}
}

// fakeActivities are the activities of these tests; RestartApproval sets Created and records the call.
type fakeActivities struct {
	facts    map[string]ActivityFacts
	restarts []restartCall
}

// restartCall is one call of RestartApproval.
type restartCall struct {
	activityID string
	reasons    []string
}

// newFakeActivities returns the activities of these tests.
func newFakeActivities() *fakeActivities {
	planned := func(s string) *money.Amount { a := amount(s); return &a }
	path := []errs.RecordRef{
		{Type: "plan", ID: "P-1001", Name: "FY2026 Lubricants Europe"},
		{Type: "campaign", ID: "C-1001", Name: "Helix launch"},
		{Type: "initiative", ID: "I-1001", Name: "Digital & social"},
	}
	lubricants := func(f ActivityFacts) ActivityFacts {
		f.Path, f.PlanYear, f.DefaultSpread, f.DefaultFinanceBudget = path, 2026, SpreadEvenly, "2026 Lubricants Budget"
		return f
	}
	return &fakeActivities{facts: map[string]ActivityFacts{
		launchFilm: lubricants(ActivityFacts{ID: launchFilm, Name: "Launch film", OwnerID: priya.ID, Status: "created",
			StartDate: day("2026-03-01"), EndDate: day("2026-05-31"), PlannedCost: planned("120000.00")}),
		socialOnAir: lubricants(ActivityFacts{ID: socialOnAir, Name: "Social always-on", OwnerID: priya.ID, Status: "approved",
			StartDate: day("2026-01-01"), EndDate: day("2026-06-30"), PlannedCost: planned("80000.00")}),
		tradeAds: lubricants(ActivityFacts{ID: tradeAds, Name: "Trade magazine ads", OwnerID: priya.ID, Status: "submitted",
			StartDate: day("2026-04-01"), EndDate: day("2026-09-30"), PlannedCost: planned("30000.00")}),
		partnerFund: lubricants(ActivityFacts{ID: partnerFund, Name: "Partner co-marketing fund", OwnerID: priya.ID,
			Status: "approved", StartDate: day("2026-03-01"), Ongoing: true, PlannedCost: planned("100000.00")}),
		lunarPromo: lubricants(ActivityFacts{ID: lunarPromo, Name: "Lunar new year promo", Status: "created"}),
		retailEvent: {ID: retailEvent, Name: "Retail event", OwnerID: daniel.ID, Status: "created", Path: path,
			StartDate: day("2026-06-01"), EndDate: day("2026-06-30"), PlannedCost: planned("5000.00"),
			PlanYear: 2026, DefaultSpread: SpreadLastMonth, DefaultFinanceBudget: "2026 Retail Budget"},
	}}
}

// Facts returns the facts of the activity id.
func (f *fakeActivities) Facts(_ context.Context, id string) (ActivityFacts, bool, error) {
	facts, found := f.facts[id]
	return facts, found, nil
}

// RestartApproval records the call and sets the activity to Created.
func (f *fakeActivities) RestartApproval(_ context.Context, _ actor.Actor, id string, reasons []string) error {
	f.restarts = append(f.restarts, restartCall{activityID: id, reasons: reasons})
	facts := f.facts[id]
	facts.Status = "created"
	f.facts[id] = facts
	return nil
}

// kind is a history kind "activity" over the fake activities, top level, as 006's would be.
func (f *fakeActivities) kind() history.Kind {
	return history.Kind{
		Name: activityKind, Singular: "activity", Plural: "activities",
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			facts, found := f.facts[id]
			if !found {
				return history.Snapshot{}, false, nil
			}
			data, err := json.Marshal(facts)
			return history.Snapshot{ID: id, Name: facts.Name, Data: data}, true, err
		},
		Remove: func(_ context.Context, id string) error {
			delete(f.facts, id)
			return nil
		},
		Put: func(_ context.Context, s history.Snapshot) error {
			var facts ActivityFacts
			if err := json.Unmarshal(s.Data, &facts); err != nil {
				return err
			}
			f.facts[s.ID] = facts
			return nil
		},
	}
}

// fakeCurrencies are the seeded currencies: USD 1, EUR 0.92, GBP 0.79, CNY 7.10, INR 83.50 (inactive).
type fakeCurrencies struct {
	byCode map[string]settings.Currency
}

// newFakeCurrencies returns the currencies of these tests.
func newFakeCurrencies() *fakeCurrencies {
	c := &fakeCurrencies{byCode: map[string]settings.Currency{}}
	for _, x := range []struct {
		code, name, symbol, rate string
		active                   bool
	}{
		{"USD", "US dollar", "$", "1", true}, {"EUR", "Euro", "€", "0.92", true}, {"GBP", "Pound sterling", "£", "0.79", true},
		{"CNY", "Chinese yuan", "¥", "7.10", true}, {"INR", "Indian rupee", "₹", "83.50", false},
	} {
		rate, err := money.ParseRate(x.rate)
		if err != nil {
			panic(err)
		}
		c.byCode[x.code] = settings.Currency{Code: x.code, Name: x.name, Symbol: x.symbol, Rate: rate, Active: x.active}
	}
	return c
}

// setRate changes the rate of code, as an administrator does in Settings.
func (c *fakeCurrencies) setRate(code, rate string) {
	parsed, err := money.ParseRate(rate)
	if err != nil {
		panic(err)
	}
	currency := c.byCode[code]
	currency.Rate = parsed
	c.byCode[code] = currency
}

// Currency returns the currency with code.
func (c *fakeCurrencies) Currency(_ context.Context, code string) (settings.Currency, bool, error) {
	currency, found := c.byCode[code]
	return currency, found, nil
}

// Currencies returns every currency, by code.
func (c *fakeCurrencies) Currencies(_ context.Context) ([]settings.Currency, error) {
	list := []settings.Currency{}
	for _, code := range []string{"CNY", "EUR", "GBP", "INR", "USD"} {
		list = append(list, c.byCode[code])
	}
	return list, nil
}

// Rates returns every currency's rate (016).
func (c *fakeCurrencies) Rates(_ context.Context) (money.Rates, error) {
	rates := money.Rates{}
	for code, currency := range c.byCode {
		rates[code] = currency.Rate
	}
	return rates, nil
}

// CurrentRates returns every currency's rate, dated 19 Oct 2026 and imported from GSAP but USD (016 FR-009).
func (c *fakeCurrencies) CurrentRates(_ context.Context) ([]settings.RateInfo, error) {
	infos := []settings.RateInfo{}
	day, source := date.MustParse("2026-10-19"), settings.RateGSAPImport
	for _, code := range []string{"USD", "CNY", "EUR", "GBP", "INR"} {
		info := settings.RateInfo{Currency: code, Rate: c.byCode[code].Rate}
		if code != "USD" {
			info.RateDate, info.Source = &day, &source
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// budgetItems are the finance budget items of these tests; 2027 has none.
var budgetItems = []BudgetItem{
	{ID: europeDigital, Year: 2026, BudgetName: "2026 Lubricants Budget", Name: "Europe – Digital", Code: "LUB-EU-DIG", Active: true},
	{ID: europeEvents, Year: 2026, BudgetName: "2026 Lubricants Budget", Name: "Europe – Events", Code: "LUB-EU-EVT", Active: true},
	{ID: europeBrand, Year: 2026, BudgetName: "2026 Lubricants Budget", Name: "Europe – Brand", Code: "LUB-EU-BRD", Active: true},
	{ID: europeLegacy, Year: 2026, BudgetName: "2026 Lubricants Budget", Name: "Europe – Legacy", Code: "LUB-EU-LEG", Active: false},
	{ID: onlyItem, Year: 2026, BudgetName: "2026 Retail Budget", Name: "Retail – All", Code: "RET-ALL", Active: true},
	{ID: lastYearItem, Year: 2025, BudgetName: "2025 Lubricants Budget", Name: "Europe – Digital", Code: "LUB-EU-DIG", Active: true},
}

// fakeBudgets are the finance budget items of these tests.
type fakeBudgets struct{}

// Item returns the item with id.
func (fakeBudgets) Item(_ context.Context, id string) (BudgetItem, bool, error) {
	i := slices.IndexFunc(budgetItems, func(item BudgetItem) bool { return item.ID == id })
	if i < 0 {
		return BudgetItem{}, false, nil
	}
	return budgetItems[i], true, nil
}

// Items returns every item.
func (fakeBudgets) Items(context.Context) ([]BudgetItem, error) {
	return slices.Clone(budgetItems), nil
}

// fakeAttributes stand for feature 002's attribute rules: Spend type and GL code are needed to submit; the
// GL code "SO-1299" never fits the spend type; problems adds any other problems.
type fakeAttributes struct {
	problems []errs.FieldError
}

// glCodeMismatch is 002's message for a GL code that doesn't fit the spend type (R11).
const glCodeMismatch = "'7250080 – Advertising digital' doesn't belong to the chosen spend type — choose another GL code."

// Check returns the values without empty ones, and the problems.
func (x *fakeAttributes) Check(_ context.Context, in settings.AttributeInput) (settings.AttributeResult, error) {
	result := settings.AttributeResult{Problems: slices.Clone(x.problems), Values: settings.AttributeValues{}}
	for id, value := range in.Values {
		if string(value) == `"SO-1299"` {
			result.Problems = append(result.Problems, errs.FieldError{Field: "attributes." + id, Message: glCodeMismatch})
		}
		if len(value) > 0 && string(value) != "null" && string(value) != `""` {
			result.Values[id] = value
		}
	}
	return result, nil
}

// Missing lists Spend type and GL code when they have no value.
func (x *fakeAttributes) Missing(_ context.Context, values settings.AttributeValues, _ []string) ([]settings.MissingField, error) {
	missing := []settings.MissingField{}
	for _, attr := range []struct{ id, label string }{{spendType, "Spend type"}, {glCode, "GL code"}} {
		if _, filled := values[attr.id]; !filled {
			missing = append(missing, settings.MissingField{Field: "attributes." + attr.id, Label: attr.label})
		}
	}
	return missing, nil
}

// Describe gives the attribute's label and the option's id in words.
func (x *fakeAttributes) Describe(_ context.Context, field string, value any) (string, string, error) {
	labels := map[string]string{"attributes." + spendType: "Spend type", "attributes." + glCode: "GL code"}
	options := map[any]string{"SO-1101": "Digital – Platform development", "SO-1201": "7250080 – Advertising digital",
		"SO-1202": "7250040 – Events and sponsorship"}
	return labels[field], options[value], nil
}

// fakePeople are the people of these tests; Sam Former is inactive.
type fakePeople struct{}

// Person returns the person with id.
func (fakePeople) Person(_ context.Context, id string) (people.PersonRef, bool, error) {
	all := map[string]people.PersonRef{
		priya.ID:  {ID: priya.ID, Name: priya.Name, Initials: "PR", JobTitle: "Marketer", Active: true},
		daniel.ID: {ID: daniel.ID, Name: daniel.Name, Initials: "DO", JobTitle: "Marketing team leader", Active: true},
		"U-1009":  {ID: "U-1009", Name: "Sam Former", Initials: "SF", JobTitle: "Marketer", Active: false},
	}
	person, found := all[id]
	return person, found, nil
}

// DisplayRateOf returns Lena's display currency, EUR at 0.92; everyone else shows USD.
func (fakePeople) DisplayRateOf(_ context.Context, personID string) (people.DisplayRate, error) {
	if personID == lena.ID {
		return people.DisplayRate{Code: "EUR", Rate: "0.92"}, nil
	}
	return people.DisplayRate{Code: "USD", Rate: "1"}, nil
}
