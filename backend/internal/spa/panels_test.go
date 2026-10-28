package spa

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/authz"
)

func TestBudgetPanel_FR020_TheCampaignBudgetAgainstPlannedCost(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("L-1").Linked = true
	f.record("L-2").Linked = true
	f.record("L-1").PlannedCost = ptr(amt(t, "160000.00"))
	f.record("L-1").Spread = []MonthAmount{{Month: "2026-07", USD: amt(t, "160000.00")}}
	b := f.view(t, Query{CampaignID: "C-1"}).Budget
	got := []string{moneyText(b.Budget), b.PlannedCost.Format(), moneyText(b.Committed), textOf(b.PlannedVsBudgetPercent),
		b.Linked.Format(), b.NotLinked.Format()}
	want := []string{"2,200,000.00 USD", "400,000.00 USD", "200,000.00 USD", "-81.8", "300,000.00 USD", "100,000.00 USD"}
	if diff := cmp.Diff(want, got); diff != "" || b.OverBudget || b.NoBudgetReason != nil {
		t.Errorf("budget panel (-want +got):\n%s", diff)
	}
}

func TestBudgetPanel_FR020_AScopeWithoutABudgetSays(t *testing.T) {
	t.Parallel()
	b := newFixture(t).view(t, Query{}).Budget
	if b.Budget != nil || b.PlannedVsBudgetPercent != nil || textOf(b.NoBudgetReason) != msgNoBudget {
		t.Errorf("budget panel = %+v", b)
	}
}

func TestBudgetPanel_FR020_PlannedCostAboveTheBudgetIs(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.planning.plan.Budget = ptr(amt(t, "300000.00"))
	b := f.view(t, Query{}).Budget
	if !b.OverBudget || textOf(b.PlannedVsBudgetPercent) != "33.3" {
		t.Errorf("budget panel = %+v", b)
	}
}

func TestProcurement_FR021_ApprovedPendingApprovalAndDraftAmounts(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.finance.spend["L-1"]["2026-09"] = Spend{Invoiced: amt(t, "120000.00")}
	f.purchases.prs = []PurchaseRequest{
		{ID: "PR-1", Name: "Venue", Vendor: "Northlight", Status: statusApproved, Lines: []PurchaseLine{
			{LineItemID: "L-1", USD: amt(t, "200000.00")}, {LineItemID: "L-9", USD: amt(t, "5.00")}}},
		{ID: "PR-2", Name: "Print", Status: statusSubmitted, Lines: []PurchaseLine{{LineItemID: "L-2", USD: amt(t, "50000.00")}}},
		{ID: "PR-3", Name: "Media", Status: statusCreated, Lines: []PurchaseLine{{LineItemID: "L-3", USD: amt(t, "10000.00")}}},
		{ID: "PR-4", Name: "Old", Status: statusDeclined, Lines: []PurchaseLine{{LineItemID: "L-3", USD: amt(t, "7.00")}}},
	}
	p := f.view(t, Query{CampaignID: "C-1"}).Procurement
	got := []string{p.Approved.Format(), p.PendingApproval.Format(), p.Draft.Format(), moneyText(p.Invoiced)}
	want := []string{"200,000.00 USD", "50,000.00 USD", "10,000.00 USD", "120,000.00 USD"}
	if diff := cmp.Diff(want, got); diff != "" || p.Count != 4 {
		t.Errorf("procurement (-want +got):\n%s count %d", diff, p.Count)
	}
	if p.Items[0].Amount != usd(t, "200000.00") || p.Items[3].Counted || !p.Items[0].Counted {
		t.Errorf("items = %+v", p.Items)
	}
}

func TestRaise_FR022_AnActivityWithLineItemsOffers(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.purchases.raise = authz.Unavailable("Raising purchase requests isn't part of your role.")
	row := f.view(t, Query{}).Activities[0]
	if row.RaisePurchaseRequest != f.purchases.raise || len(row.LineItems) != 2 {
		t.Errorf("row = %+v", row)
	}
}

func TestRaise_FR022_AnActivityWithoutLineItemsSays(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.planning.records = f.planning.records[:len(f.planning.records)-1]
	row := f.view(t, Query{}).Activities[1]
	if row.RaisePurchaseRequest != authz.Unavailable(msgAddLineItemFirst) {
		t.Errorf("raise = %+v", row.RaisePurchaseRequest)
	}
}
