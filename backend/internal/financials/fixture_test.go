package financials

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// The people of these tests.
var (
	alex  = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	priya = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	lena  = actor.Actor{ID: "U-1008", Name: "Lena Vogel"} // shows amounts in EUR
)

// start is the time on the clock when every test begins: 20 Oct 2026 09:00 UTC (plan, "Tests").
var start = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

// node builds a planning node.
func node(level Level, id, parentID, planID, name string, order int, planned string) Node {
	n := Node{Level: level, ID: id, ParentID: parentID, PlanID: planID, Name: name, PlanYear: 2026, Order: order}
	if planned != "" {
		amount, err := money.ParseAmount(planned)
		if err != nil {
			panic(err)
		}
		n.PlannedCost = &amount
	}
	return n
}

// demoTree is the seed's planning tree in the shape the planning adapter answers (data model §6.1): two
// 2026 plans, line items with their USD planned costs.
func demoTree() []Node {
	return []Node{
		node(LevelPlan, "P-1001", "", "P-1001", "FY2026 Lubricants Europe", 0, ""),
		node(LevelCampaign, "C-1001", "P-1001", "P-1001", "Helix launch", 0, ""),
		node(LevelInitiative, "I-1001", "C-1001", "P-1001", "Digital & social", 0, ""),
		node(LevelInitiative, "I-1002", "C-1001", "P-1001", "Print & outdoor", 1, ""),
		node(LevelCampaign, "C-1002", "P-1001", "P-1001", "Rimula fleet", 1, ""),
		node(LevelInitiative, "I-1003", "C-1002", "P-1001", "Events", 0, ""),
		node(LevelInitiative, "I-1004", "C-1002", "P-1001", "Retail events", 1, ""),
		node(LevelActivity, "A-1001", "I-1001", "P-1001", "Launch film", 1, "120000.00"),
		node(LevelActivity, "A-1002", "I-1001", "P-1001", "Social always-on", 0, "80000.00"),
		node(LevelActivity, "A-1003", "I-1002", "P-1001", "Billboard winter wave", 0, "45000.00"),
		node(LevelActivity, "A-1004", "I-1002", "P-1001", "Trade magazine ads", 1, "30000.00"),
		node(LevelActivity, "A-1005", "I-1004", "P-1001", "Fleet roadshow", 0, "60000.00"),
		node(LevelActivity, "A-1006", "I-1004", "P-1001", "Christmas trade gifting", 1, "25000.00"),
		node(LevelActivity, "A-1007", "I-1003", "P-1001", "Partner co-marketing fund", 0, "100000.00"),
		node(LevelLineItem, "L-1001", "A-1001", "P-1001", "Agency fee", 0, "50000.00"),
		node(LevelLineItem, "L-1002", "A-1001", "P-1001", "Film production", 1, "50000.00"),
		node(LevelLineItem, "L-1003", "A-1002", "P-1001", "Venue hire", 2, "20000.00"),
		node(LevelLineItem, "L-1004", "A-1002", "P-1001", "Speaker fees", 3, "5000.00"),
		node(LevelLineItem, "L-1005", "A-1004", "P-1001", "Media booking", 4, "30000.00"),
		node(LevelLineItem, "L-1006", "A-1003", "P-1001", "Stand build", 5, "10000.00"),
		node(LevelPlan, "P-1002", "", "P-1002", "FY2026 Mobility China", 0, ""),
		node(LevelCampaign, "C-1003", "P-1002", "P-1002", "Retail growth", 0, ""),
		node(LevelInitiative, "I-1005", "C-1003", "P-1002", "Station promotions", 0, ""),
		node(LevelActivity, "A-1008", "I-1005", "P-1002", "Grand opening Shanghai", 0, "40000.00"),
		node(LevelLineItem, "L-1007", "A-1008", "P-1002", "Station signage", 6, "1000.00"),
	}
}

// demoRates are the seed's rates: units per 1 USD.
func demoRates() money.Rates {
	return money.Rates{
		"USD": money.One, "EUR": money.RateFromMicros(920_000), "GBP": money.RateFromMicros(790_000),
		"CNY": money.RateFromMicros(7_200_000),
	}
}

// fakePlanning is Planning in memory.
type fakePlanning struct {
	nodes   []Node
	inBin   []string            // line items in the recycle bin
	myTeams map[string][]string // person → plans with one of their teams
}

func (f *fakePlanning) Nodes(context.Context) ([]Node, error) { return slices.Clone(f.nodes), nil }
func (f *fakePlanning) LineItemsInBin(context.Context) ([]string, error) {
	return slices.Clone(f.inBin), nil
}
func (f *fakePlanning) MyTeamPlanIDs(_ context.Context, personID string) ([]string, error) {
	return f.myTeams[personID], nil
}

// moveToBin takes the line item id out of the live tree and puts it in the recycle bin.
func (f *fakePlanning) moveToBin(id string) Node {
	i := slices.IndexFunc(f.nodes, func(n Node) bool { return n.ID == id })
	removed := f.nodes[i]
	f.nodes = slices.Delete(f.nodes, i, i+1)
	f.inBin = append(f.inBin, id)
	return removed
}

// restore puts a line item back from the recycle bin.
func (f *fakePlanning) restore(n Node) {
	f.nodes = append(f.nodes, n)
	f.inBin = slices.DeleteFunc(f.inBin, func(id string) bool { return id == n.ID })
}

// fakeRates is Rates in memory.
type fakeRates struct{ rates money.Rates }

func (f *fakeRates) Rates(context.Context) (money.Rates, error) { return f.rates, nil }

// fakeDisplays shows Lena Vogel's amounts in EUR at 0.92 and everyone else's in USD.
type fakeDisplays struct{}

func (fakeDisplays) Display(_ context.Context, personID string) (money.Display, error) {
	if personID == lena.ID {
		return money.DisplayIn("EUR", "0.92"), nil
	}
	return money.DisplayUSD, nil
}

// fakeAccess refuses what its fields say, with 403 forbidden.
type fakeAccess struct {
	noView, noImport, noLink bool
	plans                    []string // nil: every plan
}

// refusal is the error of a refused check.
var refusal = errs.Forbidden("Seeing financial data isn't part of your role (Viewer). Ask an administrator if you need it.")

func (f *fakeAccess) SeeFinance(context.Context, actor.Actor) error { return refuse(f.noView) }
func (f *fakeAccess) SeeFinanceIn(_ context.Context, _ actor.Actor, planID string) error {
	if !f.noView && f.plans != nil && !slices.Contains(f.plans, planID) {
		return errs.NotFound("This plan no longer exists — someone may have deleted it.")
	}
	return refuse(f.noView)
}
func (f *fakeAccess) Import(context.Context, actor.Actor) error { return refuse(f.noImport) }
func (f *fakeAccess) Link(_ context.Context, _ actor.Actor, planID string) error {
	return refuse(f.noLink || (planID != "" && f.plans != nil && !slices.Contains(f.plans, planID)))
}
func (f *fakeAccess) VisiblePlanIDs(context.Context, actor.Actor) ([]string, error) {
	return f.plans, nil
}

// refuse returns refusal when refused.
func refuse(refused bool) error {
	if refused {
		return refusal
	}
	return nil
}

// fixture is a financials service on a temporary store with fakes for every port, a fixed clock, a real
// history service and the demo data loaded.
type fixture struct {
	service  *Service
	db       *store.DB
	history  *history.Service
	planning *fakePlanning
	rates    *fakeRates
	access   *fakeAccess
	now      time.Time
}

// newFixture builds the fixture with the demo data of data model §6 loaded by the seed's code.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := newEmptyFixture(t)
	f.write(t, func(ctx context.Context) error {
		items := storedSet{}
		for _, n := range f.planning.nodes {
			if n.Level == LevelLineItem {
				items[n.ID] = true
			}
		}
		_, err := LoadDemo(ctx, f.service.repo, f.service.ids, items, f.rates.rates, start.Add(-24*time.Hour))
		return err
	})
	return f
}

// newEmptyFixture builds the fixture without GSAP data.
func newEmptyFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	f := &fixture{
		db: db, planning: &fakePlanning{nodes: demoTree(), myTeams: map[string][]string{priya.ID: {"P-1001"}}},
		rates: &fakeRates{rates: demoRates()}, access: &fakeAccess{}, now: start,
	}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	clock := func() time.Time { return f.now }
	f.history = history.NewService(db, history.NewJSONStore(), generator, clock)
	f.service = NewService(db, NewJSONStore(), generator, f.history, Deps{
		Planning: f.planning, Rates: f.rates, Displays: fakeDisplays{}, Access: f.access,
	}, clock)
	for _, k := range f.service.Kinds() {
		if err := f.history.Register(k); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}
	return f
}

// write runs fn in a Write and stops the test on an error.
func (f *fixture) write(t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	if err := f.db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// record returns the card of recordID for a, stopping the test on an error.
func (f *fixture) record(t *testing.T, a actor.Actor, recordID string) RecordView {
	t.Helper()
	v, err := f.service.Record(t.Context(), a, recordID)
	if err != nil {
		t.Fatalf("Record(%s): %v", recordID, err)
	}
	return v
}

// transactions returns the stored transactions.
func (f *fixture) transactions(t *testing.T) []Transaction {
	t.Helper()
	var list []Transaction
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		list, err = f.service.repo.ListTransactions(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	return list
}

// stored returns the transaction id as stored.
func (f *fixture) stored(t *testing.T, id string) Transaction {
	t.Helper()
	for _, tr := range f.transactions(t) {
		if tr.ID == id {
			return tr
		}
	}
	t.Fatalf("no transaction %s", id)
	return Transaction{}
}

// entries returns the activity-log entries, oldest first.
func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), alex, history.LogFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	slices.Reverse(entries)
	return entries
}

// file is a CSV file as an import request: lines joined with line breaks, as base64.
func file(name string, lines ...string) ImportRequest {
	content := strings.Join(lines, "\n") + "\n"
	return ImportRequest{FileName: name, Content: base64.StdEncoding.EncodeToString([]byte(content))}
}

// header is the header row of the simulated GSAP extract.
var header = strings.Join(csvColumns, ",")

// amountOf parses an amount for the tests.
func amountOf(t *testing.T, text string) money.Amount {
	t.Helper()
	a, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return a
}

// usd is an amount in US dollars.
func usd(t *testing.T, text string) money.Money {
	t.Helper()
	return money.USD(amountOf(t, text))
}

// assertCode checks that err is the API error with code (and status).
func assertCode(t *testing.T, err error, status int, code string) *errs.Error {
	t.Helper()
	var apiErr *errs.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want %d %s", err, status, code)
	}
	if apiErr.Status != status || apiErr.Code != code {
		t.Fatalf("error = %d %s (%s), want %d %s", apiErr.Status, apiErr.Code, apiErr.Message, status, code)
	}
	return apiErr
}

// version returns a pointer to v.
func version(v int) *int {
	return &v
}
