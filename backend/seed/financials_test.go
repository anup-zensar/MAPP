package seed_test

import (
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"mapp/internal/financials"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

// The demo GSAP data tells the story of data model §6.1 (FR-031): every figure of its table, read through
// the API as the cards read it, the counts, and Priya Raman's Home card. Every record's card equals its row
// in its parent's breakdown and the totals of the Financials page filtered on it (SC-002).
func TestSeed_FinancialsStory_015(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	get := func(path string, v any) {
		t.Helper()
		response := priya.Do(t, http.MethodGet, path, nil)
		if response.Status != http.StatusOK {
			t.Fatalf("GET %s = %d %s", path, response.Status, response.Body)
		}
		response.Decode(t, v)
	}
	card := func(id string) financials.RecordView {
		t.Helper()
		var v financials.RecordView
		get("/api/v1/financials/records/"+id, &v)
		return v
	}

	// name → planned, committed, invoiced, paid, accrued, not yet committed, %, line items above planned
	// ("yes" for the line item itself).
	table := map[string][8]string{
		"Venue hire":               {"20000.00", "20000.00", "20000.00", "20000.00", "0.00", "0.00", "100", "0"},
		"Speaker fees":             {"5000.00", "5000.00", "2500.00", "0.00", "2500.00", "0.00", "100", "0"},
		"Stand build":              {"10000.00", "10652.17", "10652.17", "10652.17", "0.00", "-652.17", "107", "yes"},
		"Social always-on":         {"80000.00", "25000.00", "22500.00", "20000.00", "2500.00", "55000.00", "31", "0"},
		"Billboard winter wave":    {"45000.00", "10652.17", "10652.17", "10652.17", "0.00", "34347.83", "24", "1"},
		"Launch film":              {"120000.00", "0.00", "0.00", "0.00", "0.00", "120000.00", "0", "0"},
		"Digital & social":         {"200000.00", "25000.00", "22500.00", "20000.00", "2500.00", "175000.00", "13", "0"},
		"Print & outdoor":          {"75000.00", "10652.17", "10652.17", "10652.17", "0.00", "64347.83", "14", "1"},
		"Helix launch":             {"275000.00", "35652.17", "33152.17", "30652.17", "2500.00", "239347.83", "13", "1"},
		"Rimula fleet":             {"185000.00", "0.00", "0.00", "0.00", "0.00", "185000.00", "0", "0"},
		"FY2026 Lubricants Europe": {"460000.00", "35652.17", "33152.17", "30652.17", "2500.00", "424347.83", "8", "1"},
	}
	usd := func(text string) money.Money {
		amount, err := money.ParseAmount(text)
		if err != nil {
			t.Fatalf("ParseAmount(%q): %v", text, err)
		}
		return money.USD(amount)
	}

	seen := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		v := card(id)
		if w, listed := table[v.Record.Name]; listed {
			seen[v.Record.Name] = true
			f := v.Figures
			above := strconv.Itoa(f.LineItemsAbovePlanned)
			if f.CommittedAbovePlanned {
				above = "yes"
			}
			if f.PlannedCost == nil || *f.PlannedCost != usd(w[0]) || f.Committed != usd(w[1]) || f.Invoiced != usd(w[2]) ||
				f.Paid != usd(w[3]) || f.Accrued != usd(w[4]) || f.NotYetCommitted == nil || *f.NotYetCommitted != usd(w[5]) ||
				f.CommittedPercent == nil || strconv.Itoa(*f.CommittedPercent) != w[6] || above != w[7] {
				t.Errorf("%s = %+v, want %v", v.Record.Name, f, w)
			}
		}
		var page financials.TransactionPage
		get("/api/v1/financials/transactions?recordId="+id, &page)
		if page.Totals != (financials.Totals{Committed: v.Figures.Committed, Invoiced: v.Figures.Invoiced, Paid: v.Figures.Paid,
			Accrued: v.Figures.Accrued}) || page.Total != v.Figures.TransactionCount {
			t.Errorf("%s: the Financials page totals %+v (%d), the card %+v", v.Record.Name, page.Totals, page.Total, v.Figures)
		}
		for _, row := range v.Breakdown {
			if child := card(row.Record.ID); !reflect.DeepEqual(child.Figures, row.Figures) {
				t.Errorf("%s's row %s = %+v, card %+v", v.Record.Name, row.Record.Name, row.Figures, child.Figures)
			}
			if row.Figures.TransactionCount > 0 || table[row.Record.Name] != [8]string{} {
				walk(row.Record.ID)
			}
		}
	}
	walk("P-1001")
	for name := range table {
		if !seen[name] {
			t.Errorf("%s is not in the plan's tree", name)
		}
	}

	var all financials.TransactionPage
	get("/api/v1/financials/transactions?year=2026", &all)
	// Priya does not see the not-linked PO: not-linked transactions show only to people who may import (014).
	if all.Counts != (financials.TypeCounts{All: 10, PurchaseOrder: 3, Invoice: 3, Payment: 2, Accrual: 2, NotLinked: 0}) ||
		all.Totals.Committed != usd("35652.17") || all.Totals.Invoiced != usd("33152.17") ||
		all.Totals.Paid != usd("30652.17") || all.Totals.Accrued != usd("2500.00") {
		t.Errorf("Financials page 2026: counts %+v, totals %+v", all.Counts, all.Totals)
	}
	if all.Source.FeedName == nil || *all.Source.FeedName != "Demo data (simulated GSAP)" || all.Source.AccrualMonth == nil ||
		*all.Source.AccrualMonth != "2026-09" {
		t.Errorf("source = %+v", all.Source)
	}

	var overview financials.Overview
	get("/api/v1/financials/overview?scope=myTeams", &overview)
	o := overview.Figures
	if o.Committed != usd("35652.17") || o.PlannedCost == nil || *o.PlannedCost != usd("460000.00") || o.CommittedPercent == nil ||
		*o.CommittedPercent != 8 || overview.ActivitiesAbovePlanned != 1 || overview.NotLinked != 0 {
		t.Errorf("Priya's Home card = %+v", overview)
	}
}

// The demo data links only to Venue hire, Speaker fees and Stand build, and leaves one PO not linked
// (D-306): the other line items have no transactions.
func TestSeed_FinancialsLinkOnlyThreeLineItems_D306(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, "alex.morgan@shell.com")
	response := alex.Do(t, http.MethodGet, "/api/v1/financials/transactions?pageSize=50", nil)
	var page financials.TransactionPage
	response.Decode(t, &page)
	linkedTo := map[string]int{}
	for _, row := range page.Items {
		if row.Link.LineItemID != nil {
			linkedTo[*row.Link.LineItemID]++
		}
	}
	if !reflect.DeepEqual(linkedTo, map[string]int{"L-1003": 3, "L-1004": 4, "L-1006": 3}) {
		t.Errorf("linked to %v", linkedTo)
	}
}
