package plans

import (
	"reflect"
	"testing"

	"northstar/internal/platform/money"
)

// rangePlans stores plans of 2026, 2027 and 2028 of each type: P-1001 2026 Operational, P-1002 2027
// Strategic, P-1003 2027 Operational, P-1004 2028 Cross-market, each with a budget of 100,000.00 USD.
func rangePlans(t *testing.T, f *fixture) {
	t.Helper()
	plan := func(id, name string, year int, planType PlanType) Plan {
		p := seedPlan(t, id, name, year, teamLubricants, priya.ID, "100000.00", StatusDraft)
		p.PlanType = planType
		return p
	}
	f.put(t, plan("P-1001", "A 2026", 2026, TypeOperational), plan("P-1002", "B 2027", 2027, TypeStrategic),
		plan("P-1003", "C 2027", 2027, TypeOperational), plan("P-1004", "D 2028", 2028, TypeCrossMarket))
}

// listIDsOf lists the plans of filter, as IDs.
func listIDsOf(t *testing.T, f *fixture, filter ListFilter) (PlanList, []string) {
	t.Helper()
	list, err := f.service.List(t.Context(), priya, filter)
	if err != nil {
		t.Fatalf("List(%+v): %v", filter, err)
	}
	return list, itemIDs(list)
}

// Years from–to lists the plans of every year of the range, newest first; one bound leaves the other open
// (FR-033).
func TestListPlans_YearRange_FR033(t *testing.T) {
	f := newFixture(t)
	rangePlans(t, f)

	_, both := listIDsOf(t, f, ListFilter{YearFrom: ptr(2026), YearTo: ptr(2027)})
	_, from := listIDsOf(t, f, ListFilter{YearFrom: ptr(2027)})
	_, to := listIDsOf(t, f, ListFilter{YearTo: ptr(2026)})
	_, err := f.service.List(t.Context(), priya, ListFilter{YearFrom: ptr(2028), YearTo: ptr(2026)})

	if !reflect.DeepEqual(both, []string{"P-1002", "P-1003", "P-1001"}) || !reflect.DeepEqual(from, []string{"P-1004", "P-1002", "P-1003"}) ||
		!reflect.DeepEqual(to, []string{"P-1001"}) {
		t.Errorf("2026–2027 %q, from 2027 %q, to 2026 %q", both, from, to)
	}
	assertError(t, err, problems("yearTo", "Choose a last year that is not before the first year."))
}

// year is the alias of a one-year range (D-269).
func TestListPlans_YearAlias(t *testing.T) {
	f := newFixture(t)
	rangePlans(t, f)

	_, alias := listIDsOf(t, f, ListFilter{Year: ptr(2027)})
	_, explicit := listIDsOf(t, f, ListFilter{YearFrom: ptr(2027), YearTo: ptr(2027)})

	if !reflect.DeepEqual(alias, []string{"P-1002", "P-1003"}) || !reflect.DeepEqual(alias, explicit) {
		t.Errorf("year 2027 %q, 2027–2027 %q", alias, explicit)
	}
}

// Plan type narrows the list to the chosen types; an unknown type is refused (FR-033, FR-032).
func TestListPlans_TypeFilter_FR033(t *testing.T) {
	f := newFixture(t)
	rangePlans(t, f)

	_, strategic := listIDsOf(t, f, ListFilter{Types: []string{"strategic"}})
	_, two := listIDsOf(t, f, ListFilter{Types: []string{"operational", "crossMarket"}, YearFrom: ptr(2027)})
	_, err := f.service.List(t.Context(), priya, ListFilter{Types: []string{"tactical"}})

	if !reflect.DeepEqual(strategic, []string{"P-1002"}) || !reflect.DeepEqual(two, []string{"P-1004", "P-1003"}) {
		t.Errorf("strategic %q, operational and cross-market from 2027 %q", strategic, two)
	}
	assertError(t, err, problems("type", "Choose Operational, Strategic or Cross-market."))
}

// The summary cards cover the range and the types, whatever the status and the search (FR-033).
func TestListPlans_SummaryOverRange_FR033(t *testing.T) {
	f := newFixture(t)
	rangePlans(t, f)
	planned := money.FromCents(5_000_000)
	f.addLevel(t, "activity", map[string]Facts{"P-1002": {PlannedCost: &planned}, "P-1004": {PlannedCost: &planned}})

	list, _ := listIDsOf(t, f, ListFilter{YearFrom: ptr(2027), YearTo: ptr(2028), Q: "B 2027"})

	if list.Summary.Plans != 3 || list.Summary.Draft != 3 || list.Summary.Budget.Format() != "300,000.00 USD" ||
		list.Summary.PlannedCost.Format() != "100,000.00 USD" || len(list.Items) != 1 || list.Total != 4 {
		t.Errorf("summary %+v, %d items of %d", list.Summary, len(list.Items), list.Total)
	}
}
