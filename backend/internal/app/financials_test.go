package app

import (
	"context"
	"slices"
	"testing"

	"northstar/internal/financials"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/money"
)

// The planning adapter of feature 015 answers the seed's tree with the planned costs of data model §6.1:
// activities as typed, line items in USD at the current rate, containers left to the roll-up (T010).
func TestFinancialPlanning_SeedTree_FR012(t *testing.T) {
	built, db := newSeededApp(t)
	var nodes []financials.Node
	var mine []string
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		if nodes, err = built.finance.Nodes(ctx); err != nil {
			return err
		}
		mine, err = built.finance.MyTeamPlanIDs(ctx, "U-1002") // Priya Raman
		return err
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	byName := map[string]financials.Node{}
	for _, n := range nodes {
		byName[n.Name] = n
	}
	planned := func(name, want string) {
		t.Helper()
		n, found := byName[name]
		if !found {
			t.Fatalf("%s is not in the tree", name)
		}
		if want == "" {
			if n.PlannedCost != nil {
				t.Errorf("%s planned %s, want none", name, n.PlannedCost.Format())
			}
			return
		}
		amount, _ := money.ParseAmount(want)
		if n.PlannedCost == nil || *n.PlannedCost != amount {
			t.Errorf("%s planned %v, want %s", name, n.PlannedCost, want)
		}
	}
	planned("Venue hire", "20000.00")  // 15,800.00 GBP
	planned("Stand build", "10000.00") // 9,200.00 EUR
	planned("Speaker fees", "5000.00")
	planned("Social always-on", "80000.00")
	planned("Billboard winter wave", "45000.00")
	planned("Helix launch", "")

	venue, social := byName["Venue hire"], byName["Social always-on"]
	if venue.Level != financials.LevelLineItem || venue.ParentID != social.ID || venue.PlanID != social.PlanID ||
		venue.PlanYear != 2026 || social.Level != financials.LevelActivity {
		t.Errorf("Venue hire = %+v under %+v", venue, social)
	}
	plan := byName["FY2026 Lubricants Europe"]
	if plan.Level != financials.LevelPlan || plan.ID != social.PlanID || len(mine) != 1 || mine[0] != plan.ID {
		t.Errorf("plan %+v, Priya's team plans %v", plan, mine)
	}
}

// A line item in the recycle bin leaves the live tree and is reported as in the bin, so matching still
// finds it (data model §4.2, T010).
func TestFinancialPlanning_LineItemsInBin_FR007(t *testing.T) {
	built, db := newSeededApp(t)
	adapter := built.finance.(financialPlanning)
	var inBin []string
	var nodes []financials.Node
	err := db.Write(t.Context(), func(ctx context.Context) error {
		if _, err := adapter.history.MoveToBin(ctx, actor.System, "lineItem", "L-1005"); err != nil {
			return err
		}
		var err error
		if inBin, err = adapter.LineItemsInBin(ctx); err != nil {
			return err
		}
		nodes, err = adapter.Nodes(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !slices.Equal(inBin, []string{"L-1005"}) {
		t.Errorf("in the bin = %v, want L-1005", inBin)
	}
	if slices.ContainsFunc(nodes, func(n financials.Node) bool { return n.ID == "L-1005" }) {
		t.Errorf("L-1005 is still in the live tree")
	}
}

// The display adapter gives the person's display currency with its current rate.
func TestFinancialDisplays_PersonCurrency_FR011(t *testing.T) {
	built, db := newSeededApp(t)
	displays := financialDisplays{people: built.finance.(financialPlanning).people}
	var d money.Display
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		d, err = displays.Display(ctx, "U-1002")
		return err
	})
	if err != nil || d.Currency() != "USD" {
		t.Errorf("Priya's display = %v, %v", d.Currency(), err)
	}
}
