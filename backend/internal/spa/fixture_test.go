package spa

import (
	"context"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

var planner = actor.Actor{ID: "U-1", Name: "Priya Raman"}

type fakeTx struct{}

func (fakeTx) Read(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type fakePlanning struct {
	plan    Plan
	found   bool
	records []Record
	dims    []Dimension
}

func (f *fakePlanning) Plan(_ context.Context, planID string) (Plan, bool, error) {
	return f.plan, f.found && planID == f.plan.ID, nil
}

func (f *fakePlanning) Records(context.Context, Plan) ([]Record, error) {
	return append([]Record{}, f.records...), nil
}

func (f *fakePlanning) Dimensions(context.Context) ([]Dimension, error) { return f.dims, nil }

type fakeFinance struct {
	sees  bool
	spend map[string]map[string]Spend
	asOf  *time.Time
}

func (f *fakeFinance) SeesFinance(context.Context, actor.Actor, string) (bool, error) {
	return f.sees, nil
}

func (f *fakeFinance) SpendByMonth(context.Context) (map[string]map[string]Spend, error) {
	return f.spend, nil
}

func (f *fakeFinance) DataAsOf(context.Context) (*time.Time, error) { return f.asOf, nil }

type fakePurchases struct {
	prs   []PurchaseRequest
	raise authz.Availability
}

func (f *fakePurchases) Drawn(_ context.Context, ids []string) ([]PurchaseRequest, error) {
	var list []PurchaseRequest
	for _, pr := range f.prs {
		kept := pr
		kept.Lines = nil
		for _, l := range pr.Lines {
			for _, id := range ids {
				if l.LineItemID == id {
					kept.Lines = append(kept.Lines, l)
				}
			}
		}
		if len(kept.Lines) > 0 {
			list = append(list, kept)
		}
	}
	return list, nil
}

func (f *fakePurchases) CanRaise(context.Context, Plan) (authz.Availability, error) {
	return f.raise, nil
}

type fakeDisplays struct{ display money.Display }

func (f fakeDisplays) Display(context.Context, string) (money.Display, error) { return f.display, nil }

// fixture is 006's worked example: campaign C-1 with Activity A (340,000.00 on two line items spread to Q3 and
// Q4) and Activity B (60,000.00 spread to Q4), and 150,000.00 paid on A in August.
type fixture struct {
	planning  *fakePlanning
	finance   *fakeFinance
	purchases *fakePurchases
	display   money.Display
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	return &fixture{
		planning: &fakePlanning{found: true, plan: Plan{ID: "P-1", Name: "FY2026 Lubricants Europe", Year: 2026,
			Status: "published", BCPIR: true}, records: exampleRecords(t), dims: exampleDimensions()},
		finance: &fakeFinance{sees: true, spend: map[string]map[string]Spend{
			"L-1": {"2026-08": {Committed: amt(t, "200000.00"), Actual: amt(t, "150000.00")}},
		}},
		purchases: &fakePurchases{raise: authz.Allowed},
		display:   money.DisplayUSD,
	}
}

func (f *fixture) service() *Service {
	return NewService(fakeTx{}, Deps{Planning: f.planning, Finance: f.finance, Purchases: f.purchases,
		Displays: fakeDisplays{display: f.display}})
}

func (f *fixture) view(t *testing.T, q Query) View {
	t.Helper()
	v, err := f.service().View(t.Context(), planner, "P-1", q)
	if err != nil {
		t.Fatalf("View(%+v): %v", q, err)
	}
	return v
}

func (f *fixture) record(id string) *Record {
	for i := range f.planning.records {
		if f.planning.records[i].ID == id {
			return &f.planning.records[i]
		}
	}
	return nil
}

func exampleRecords(t *testing.T) []Record {
	t.Helper()
	start := date.New(2026, time.July, 1)
	months := []string{"2026-07", "2026-08", "2026-09", "2026-10", "2026-11", "2026-12"}
	return []Record{
		{Level: LevelCampaign, ID: "C-1", ParentID: "P-1", Name: "Brand", Budget: ptr(amt(t, "2200000.00"))},
		{Level: LevelCampaign, ID: "C-2", ParentID: "P-1", Order: 1, Name: "Empty campaign"},
		{Level: LevelInitiative, ID: "I-1", ParentID: "C-1", Name: "Awareness"},
		{Level: LevelActivity, ID: "A-1", ParentID: "I-1", Name: "Activity A", Owner: "Priya Raman", Status: "approved",
			ActivityType: "Event", StartDate: &start, Months: months, PlannedCost: ptr(amt(t, "340000.00")),
			Values: map[string][]Value{
				"channel": {{Label: "External"}},
				"country": {{Label: "China", Share: pct(t, "60.00")}, {Label: "India", Share: pct(t, "40.00")}},
			}},
		{Level: LevelActivity, ID: "A-2", ParentID: "I-1", Order: 1, Name: "Activity B", Status: "created",
			ActivityType: "Digital", StartDate: &start, Months: months, PlannedCost: ptr(amt(t, "60000.00")),
			Values: map[string][]Value{"country": {{Label: "China", Share: pct(t, "100.00")}}}},
		lineItemOf("L-1", "A-1", 200000, "2026-07"),
		lineItemOf("L-2", "A-1", 140000, "2026-10"),
		lineItemOf("L-3", "A-2", 60000, "2026-11"),
	}
}

func lineItemOf(id, activityID string, dollars int64, month string) Record {
	amount := money.FromCents(dollars * 100)
	return Record{Level: LevelLineItem, ID: id, ParentID: activityID, Name: "Line " + id, PlannedCost: &amount,
		Spread: []MonthAmount{{Month: month, USD: amount}}, Values: map[string][]Value{}}
}

func exampleDimensions() []Dimension {
	return []Dimension{
		{Key: "channel", Label: "Channel / audience", Level: LevelActivity, Channel: true},
		{Key: "activityType", Label: "Activity type", Level: LevelActivity},
		{Key: "country", Label: "Country / OU", Level: LevelActivity},
		{Key: "spendType", Label: "Spend type", Level: LevelLineItem},
	}
}

func amt(t *testing.T, s string) money.Amount {
	t.Helper()
	a, err := money.ParseAmount(s)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", s, err)
	}
	return a
}

func usd(t *testing.T, s string) money.Money {
	t.Helper()
	return money.USD(amt(t, s))
}

func pct(t *testing.T, s string) *money.Percent {
	t.Helper()
	p, err := money.ParsePercent(s)
	if err != nil {
		t.Fatalf("ParsePercent(%q): %v", s, err)
	}
	return &p
}

func ptr[T any](v T) *T { return &v }

func textOf(s *string) string {
	if s == nil {
		return "—"
	}
	return *s
}

func moneyText(m *money.Money) string {
	if m == nil {
		return "—"
	}
	return m.Format()
}
