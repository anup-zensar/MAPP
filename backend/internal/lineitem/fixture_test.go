package lineitem

import (
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
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	lena   = actor.Actor{ID: "U-1008", Name: "Lena Vogel"} // shows amounts in EUR
)

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

const (
	launchFilm  = "A-1001"
	socialOnAir = "A-1002"
	tradeAds    = "A-1003"
	partnerFund = "A-1004"
	lunarPromo  = "A-1005"
	retailEvent = "A-1006"
)

const (
	europeDigital = "FBI-1001"
	europeEvents  = "FBI-1002"
	europeBrand   = "FBI-1003"
	europeLegacy  = "FBI-1004" // inactive
	onlyItem      = "FBI-1005"
	lastYearItem  = "FBI-1006"
)

// Spend type and GL code are needed to submit.
const (
	spendType = "SA-1031"
	glCode    = "SA-1032"
)

type fixture struct {
	service    *Service
	db         *fake.DB
	history    *history.Service
	activities *fakeActivities
	currencies *fakeCurrencies
	attributes *fakeAttributes
}

// newTestService registers the fake kind "activity" before "lineItem", as internal/app registers 006's.
func newTestService(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	now := func() time.Time { return start }
	generator := ids.NewGenerator(fake.NewCounters(db))
	hist := history.NewService(db, historytest.NewStore(db), generator, now)
	f := &fixture{
		db: db, history: hist, activities: newFakeActivities(), currencies: newFakeCurrencies(), attributes: &fakeAttributes{},
	}
	deps := Deps{
		Activities: f.activities, Currencies: f.currencies, FinanceBudgets: fakeBudgets{},
		Attributes: f.attributes, People: fakePeople{},
	}
	f.service = NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: now}, newFakeStore(db), deps)
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

func (f *fixture) create(t *testing.T, a actor.Actor, activityID string, in Input) View {
	t.Helper()
	view, err := f.service.Create(t.Context(), a, activityID, in)
	if err != nil {
		t.Fatalf("Create(%s): %v", in.Name, err)
	}
	return view
}

func (f *fixture) stored(t *testing.T) []LineItem {
	t.Helper()
	items, err := newFakeStore(f.db).List(t.Context())
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

// input is a form owned by Priya, spread evenly, in finance budget year 2026, Request SWBS code from GSAP on.
func input(name, currency, cost string) Input {
	year := 2026
	return Input{
		Name: name, OwnerID: text(priya.ID), Currency: currency, PlannedCost: text(cost),
		Spread: SpreadInput{Method: SpreadEvenly, Months: []SpreadMonthInput{}}, Attributes: settings.AttributeValues{},
		RequestSwbs: true, FinanceBudgetYear: &year,
	}
}

// complete adds a spend type, a GL code and a finance budget item: nothing is missing.
func complete(in Input) Input {
	in.Attributes = settings.AttributeValues{spendType: json.RawMessage(`"SO-1101"`), glCode: json.RawMessage(`"SO-1201"`)}
	in.FinanceBudgetItemID = text(europeDigital)
	return in
}

func text(s string) *string {
	return &s
}

func amount(s string) money.Amount {
	a, err := money.ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return a
}

func day(s string) *date.Date {
	d := date.MustParse(s)
	return &d
}

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

func usd(s string) money.Money {
	return money.USD(amount(s))
}

// launchFilmWithTwoItems leaves 20,000.00 USD unassigned: 46,000.00 EUR is 50,000.00 USD at 0.92.
func launchFilmWithTwoItems(t *testing.T, f *fixture) (fee, film View) {
	t.Helper()
	fee = f.create(t, priya, launchFilm, complete(input("Agency fee", "USD", "50000.00")))
	film = f.create(t, priya, launchFilm, input("Film production", "EUR", "46000.00"))
	return fee, film
}

func confirmed(in Input) Input {
	in.RestartApproval = true
	return in
}
