package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/activities"
	"mapp/internal/financials"
	"mapp/internal/platform/actor"
)

// The planning nodes, read once for every plan, keep the order the plan pages show: campaigns and
// initiatives as the plan tree lists them, activities as the plan's list of activities (028 FR-001).
func TestFinancialPlanning_NodesKeepThePlanPagesOrder_028_FR001(t *testing.T) {
	built, db := newSeededApp(t)
	adapter := built.finance.(financialPlanning)
	activityService := built.allWork.(allWorkSource).activities
	want, got := map[string]int{}, map[string]int{}
	err := db.Read(t.Context(), func(ctx context.Context) error {
		nodes, err := adapter.Nodes(ctx)
		for _, n := range nodes {
			if n.Level != financials.LevelLineItem && n.Level != financials.LevelPlan {
				got[n.ID] = n.Order
			}
		}
		return errors.Join(err, pageOrders(ctx, adapter, activityService, want))
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("order of the planning nodes (-plan pages +nodes):\n%s", diff)
	}
}

// A record's card reads only its plan: the plan of any live record is found from the record, and the nodes
// of a plan are exactly its part of every node (028 FR-002).
func TestFinancialPlanning_PlanOfAndPlanNodes_028_FR002(t *testing.T) {
	built, db := newSeededApp(t)
	adapter := built.finance.(financialPlanning)
	got := map[string]string{}
	var every, ofPlan []financials.Node
	err := db.Read(t.Context(), func(ctx context.Context) error {
		for _, id := range []string{"L-1005", "A-1002", "I-1001", "C-1002", "P-1004", "L-1011", "X-1"} {
			planID, live, err := adapter.PlanOf(ctx, id)
			if err != nil {
				return err
			}
			if live {
				got[id] = planID
			}
		}
		var err error
		if every, err = adapter.Nodes(ctx); err != nil {
			return err
		}
		ofPlan, err = adapter.PlanNodes(ctx, []string{"P-1002"})
		return err
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := map[string]string{"L-1005": "P-1001", "A-1002": "P-1001", "I-1001": "P-1001", "C-1002": "P-1001", "P-1004": "P-1004"}
	if diff := cmp.Diff(want, got); diff != "" { // L-1011 is in the recycle bin
		t.Errorf("plans of the records (-want +got):\n%s", diff)
	}
	inP1002 := slices.DeleteFunc(every, func(n financials.Node) bool { return n.PlanID != "P-1002" })
	if diff := cmp.Diff(described(inP1002), described(ofPlan)); diff != "" || len(ofPlan) == 0 {
		t.Errorf("PlanNodes(P-1002) (-part of Nodes +got):\n%s", diff)
	}
}

// pageOrders puts each campaign's, initiative's and activity's place among its siblings on the plan pages into
// orders.
func pageOrders(ctx context.Context, adapter financialPlanning, activityService *activities.Service, orders map[string]int) error {
	all, err := adapter.plans.List(ctx)
	if err != nil {
		return err
	}
	for _, plan := range all {
		tree, err := adapter.campaigns.ListPlanCampaigns(ctx, actor.System, plan.ID)
		if err != nil {
			return err
		}
		for ci, c := range tree.Items {
			orders[c.ID] = ci
			for ii, i := range c.Initiatives {
				orders[i.ID] = ii
			}
		}
		listed, err := activityService.ListForPlan(ctx, actor.System, plan.ID)
		if err != nil {
			return err
		}
		for order, a := range listed {
			orders[a.ID] = order
		}
	}
	return nil
}

// described lists the nodes as text, with each node's place among its siblings: Order only orders siblings.
func described(nodes []financials.Node) []string {
	sorted := slices.Clone(nodes)
	slices.SortStableFunc(sorted, func(x, y financials.Node) int { return x.Order - y.Order })
	place := map[string]int{}
	list := make([]string, 0, len(nodes))
	for _, n := range sorted {
		planned := "none"
		if n.PlannedCost != nil {
			planned = n.PlannedCost.Format()
		}
		list = append(list, fmt.Sprintf("%s %s %s %s %s %d %d %s", n.Level, n.ID, n.ParentID, n.PlanID, n.Name, n.PlanYear,
			place[n.ParentID], planned))
		place[n.ParentID]++
	}
	slices.Sort(list)
	return list
}
