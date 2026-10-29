package financials

import (
	"net/http"
	"reflect"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// User story 3 — the Financials page (FR-021).

func count(n int) *int {
	return &n
}

func (f *fixture) list(t *testing.T, a actor.Actor, q Query) TransactionPage {
	t.Helper()
	if q.Type == "" {
		q.Type = filterAll
	}
	if q.Link == "" {
		q.Link = filterAll
	}
	if q.Sort == "" {
		q.Sort, q.Desc = sortPostingDate, true
	}
	if q.Page == 0 {
		q.Page, q.PageSize = 1, 50
	}
	page, err := f.service.Transactions(t.Context(), a, q)
	if err != nil {
		t.Fatalf("Transactions(%+v): %v", q, err)
	}
	return page
}

// The tab counts and totals of the seed, and every filter (FR-021, US3 scenario 1).
func TestTransactions_FiltersCountsAndTotals_FR021(t *testing.T) { //nolint:gocyclo,tparallel // 10: the seed's counts and totals, then every filter; the subtests share one fixture
	t.Parallel()
	f := newFixture(t)
	all := f.list(t, alex, Query{})
	wantCounts := TypeCounts{All: 11, PurchaseOrder: 4, Invoice: 3, Payment: 2, Accrual: 2, NotLinked: count(1)}
	if !reflect.DeepEqual(all.Counts, wantCounts) || all.Total != 11 || all.Year != 2026 || all.Items[0].ID != "FT-1011" {
		t.Errorf("counts %+v, total %d, year %d", all.Counts, all.Total, all.Year)
	}
	wantTotals := Totals{Committed: usd(t, "47652.17"), Invoiced: usd(t, "33152.17"), Paid: usd(t, "30652.17"), Accrued: usd(t, "2500.00")}
	if all.Totals != wantTotals {
		t.Errorf("totals = %+v", all.Totals)
	}
	checkSeedChoices(t, all.Choices)

	cases := []struct {
		name  string
		q     Query
		total int
		first string
	}{
		{"type", Query{Type: string(TypeInvoice)}, 3, "FT-1006"},
		{"type by amount", Query{Type: string(TypePurchaseOrder), Sort: sortAmount, Desc: true}, 4, "FT-1001"},
		{"linked", Query{Link: filterLinked}, 10, "FT-1011"},
		{"not linked", Query{Link: filterNotLinked}, 1, "FT-1004"},
		{"plan", Query{PlanID: "P-1001"}, 10, "FT-1011"},
		{"other plan", Query{PlanID: "P-1002"}, 0, ""},
		{"record", Query{RecordID: "A-1003"}, 3, "FT-1009"},
		{"year", Query{Year: 2027}, 0, ""},
		{"search vendor", Query{Search: "brightside"}, 1, "FT-1004"},
		{"search HX key", Query{Search: "hx1003"}, 1, "FT-1001"},
		{"search SWBS code", Query{Search: "SIM-GB01"}, 3, "FT-1009"},
		{"search line item", Query{Search: "Venue Hire"}, 3, "FT-1008"},
		{"search line item ID", Query{Search: "L-1004"}, 4, "FT-1011"},
		{"search activity ID", Query{Search: "a-1003"}, 3, "FT-1009"},
		{"search activity name", Query{Search: "social ALWAYS"}, 7, "FT-1011"},
		{"search document number", Query{Search: "4500001003"}, 2, "FT-1007"},
		{"sort document number", Query{Sort: sortDocumentNumber}, 11, "FT-1008"},
		{"sort vendor", Query{Sort: sortVendor}, 11, "FT-1004"},
		{"sort type", Query{Sort: sortType}, 11, "FT-1004"},
		{"sort linked to", Query{Sort: sortLinkedTo}, 11, "FT-1004"},
		{"sort amount", Query{Sort: sortAmount, Desc: true}, 11, "FT-1008"},
		{"page 2", Query{Page: 2, PageSize: 50}, 11, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			page := f.list(t, alex, c.q)
			first := ""
			if len(page.Items) > 0 {
				first = page.Items[0].ID
			}
			if page.Total != c.total || first != c.first {
				t.Errorf("total %d, first %q; want %d, %q (%v)", page.Total, first, c.total, c.first, rowIDs(page.Items))
			}
		})
	}

	f.checkTabsKeepCountsAndTotals(t, wantCounts, wantTotals)
	f.checkRecordAndYearFilters(t)
}

// checkSeedChoices checks the year and plan choices of the seed's transactions.
func checkSeedChoices(t *testing.T, choices Choices) {
	t.Helper()
	if len(choices.Years) != 1 || choices.Years[0] != 2026 || len(choices.Plans) != 2 ||
		choices.Plans[0].Name != "FY2026 Lubricants Europe" || choices.CurrentYear != 2026 {
		t.Errorf("choices = %+v", choices)
	}
}

// checkTabsKeepCountsAndTotals checks that the type and link filters leave the tab counts and totals as
// they are, and that the not-linked tab names the reason.
func (f *fixture) checkTabsKeepCountsAndTotals(t *testing.T, wantCounts TypeCounts, wantTotals Totals) {
	t.Helper()
	invoices := f.list(t, alex, Query{Type: string(TypeInvoice)})
	if !reflect.DeepEqual(invoices.Counts, wantCounts) || invoices.Totals != wantTotals {
		t.Errorf("the tab changed the counts or totals: %+v %+v", invoices.Counts, invoices.Totals)
	}
	notLinked := f.list(t, alex, Query{Link: filterNotLinked})
	if notLinked.Totals.Committed != usd(t, "12000.00") || !reflect.DeepEqual(notLinked.Counts, wantCounts) || notLinked.Items[0].Link.Reason == nil ||
		*notLinked.Items[0].Link.Reason != "No line item for HX key HX1099 — check the tracking key in GSAP." {
		t.Errorf("not linked: totals %+v, counts %+v, first %+v", notLinked.Totals, notLinked.Counts, notLinked.Items[0].Link)
	}
}

// checkRecordAndYearFilters checks the counts of a record filter, which names its record, and of a year
// without transactions.
func (f *fixture) checkRecordAndYearFilters(t *testing.T) {
	t.Helper()
	record := f.list(t, alex, Query{RecordID: "L-1004"})
	if record.RecordFilter == nil || record.RecordFilter.Name != "Speaker fees" || record.Counts.All != 4 || *record.Counts.NotLinked != 0 {
		t.Errorf("record filter %+v, counts %+v", record.RecordFilter, record.Counts)
	}
	if other := f.list(t, alex, Query{Year: 2027}); other.Counts.All != 0 || other.Totals.Committed != usd(t, "0.00") {
		t.Errorf("2027: counts %+v, totals %+v", other.Counts, other.Totals)
	}
}

func TestTransactions_UnknownRecordAndPlan_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	_, err := f.service.Transactions(t.Context(), alex, Query{Type: filterAll, Link: filterAll, Page: 1, PageSize: 50, RecordID: "L-9999"})
	if e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed); e.Fields[0].Field != "recordId" {
		t.Errorf("fields = %+v", e.Fields)
	}
	_, err = f.service.Transactions(t.Context(), alex, Query{Type: filterAll, Link: filterAll, Page: 1, PageSize: 50, PlanID: "P-9999"})
	if e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed); e.Fields[0].Field != "planId" {
		t.Errorf("fields = %+v", e.Fields)
	}
}

func TestParseQuery_DefaultsAndEveryProblem_FR021(t *testing.T) { //nolint:gocyclo // 10: the defaults, every parameter read, then every problem at once
	t.Parallel()
	q, err := ParseQuery(map[string][]string{})
	if err != nil || q != (Query{Type: filterAll, Link: filterAll, Sort: sortPostingDate, Desc: true, Page: 1, PageSize: 50}) {
		t.Errorf("defaults = %+v, %v", q, err)
	}
	q, err = ParseQuery(map[string][]string{"sort": {"vendor:asc"}, "q": {"  helix "}, "year": {"2025"}, "pageSize": {"200"}})
	if err != nil || q.Sort != sortVendor || q.Desc || q.Search != "helix" || q.Year != 2025 || q.PageSize != 200 {
		t.Errorf("query = %+v, %v", q, err)
	}
	_, err = ParseQuery(map[string][]string{"type": {"x"}, "year": {"abc"}, "page": {"-1"}})
	if e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed); len(e.Fields) != 3 {
		t.Errorf("fields = %+v, want type, year and page", e.Fields)
	}
}
