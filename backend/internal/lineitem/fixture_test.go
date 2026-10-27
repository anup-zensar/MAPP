package lineitem

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
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
	hist := history.NewService(db, history.NewJSONStore(), generator, now)
	f := &fixture{
		db: db, history: hist, activities: newFakeActivities(), currencies: newFakeCurrencies(), attributes: &fakeAttributes{},
	}
	deps := Deps{
		Activities: f.activities, Currencies: f.currencies, FinanceBudgets: fakeBudgets{},
		Attributes: f.attributes, People: fakePeople{},
	}
	f.service = NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: now}, NewJSONStore(), deps)
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
		items, err = NewJSONStore().List(ctx)
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

// usd returns the money of a USD amount text.
func usd(s string) money.Money {
	return money.USD(amount(s))
}

// launchFilmWithTwoItems adds the spec's two line items to Launch film: Agency fee 50,000.00 USD and Film
// production 46,000.00 EUR (= 50,000.00 USD at 0.92), leaving 20,000.00 USD unassigned.
func launchFilmWithTwoItems(t *testing.T, f *fixture) (fee, film View) {
	t.Helper()
	fee = f.create(t, priya, launchFilm, complete(input("Agency fee", "USD", "50000.00")))
	film = f.create(t, priya, launchFilm, input("Film production", "EUR", "46000.00"))
	return fee, film
}

// confirmed returns in with the person's confirmation that the activity needs approval again.
func confirmed(in Input) Input {
	in.RestartApproval = true
	return in
}
