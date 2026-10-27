package campaign

import (
	"reflect"
	"testing"

	"mapp/internal/platform/money"
)

// US4 — Budget, cost and progress in the plan tree.

// count returns a pointer to n, for goal values.
func count(n int64) *int64 { return &n }

// Initiatives with planned costs of 100,000 and 50,000 under a campaign budget of 120,000: the campaign's
// planned cost is 150,000.00, budget used 125 %, over budget, and the plan's planned cost includes it
// (FR-011, US4 scenario 1, domain model §4.2).
func TestCampaignView_PlannedCostRollsUp_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", Budget: input("120000.00")})
	first := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})
	second := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Print & outdoor"})
	f.activities.add(
		ActivitySummary{ID: "A-1001", InitiativeID: first.ID, PlannedCost: usd("60000.00")},
		ActivitySummary{ID: "A-1002", InitiativeID: first.ID, PlannedCost: usd("40000.00")},
		ActivitySummary{ID: "A-1003", InitiativeID: first.ID}, // no planned cost yet: skipped
		ActivitySummary{ID: "A-1004", InitiativeID: second.ID, PlannedCost: usd("50000.00")},
	)

	tree := f.list(t, "P-1001")

	helix := tree.Items[0]
	bc := helix.BudgetAndCost
	if amountText(bc.PlannedCost) != "150000.00" || bc.BudgetUsedPercent == nil || *bc.BudgetUsedPercent != 125 || !bc.OverBudget ||
		amountText(bc.LeftToPlan) != "-30000.00" {
		t.Errorf("campaign budget and cost = %+v", bc)
	}
	if got := amountText(helix.Initiatives[0].BudgetAndCost.PlannedCost); got != "100000.00" {
		t.Errorf("Digital & social planned cost = %s, want 100000.00", got)
	}
	if got := amountText(tree.Totals.PlannedCost); got != "150000.00" {
		t.Errorf("plan planned cost = %s, want 150000.00", got)
	}
}

// With nothing below, planned cost and budget used are "—" (nil), Approval progress is "No activities"
// and goals are nil (FR-011, FR-013).
func TestViews_NothingBelow_DashAndNoActivities_FR011_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", Budget: input("100.00")})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social", Budget: input("10.00")})

	view, _ := f.service.GetCampaign(t.Context(), priya, c.ID)

	for name, v := range map[string]struct {
		bc       BudgetAndCost
		progress ApprovalSummary
		goals    GoalSums
	}{
		"campaign":   {view.BudgetAndCost, view.ApprovalProgress, view.Goals},
		"initiative": {view.Initiatives[0].BudgetAndCost, view.Initiatives[0].ApprovalProgress, view.Initiatives[0].Goals},
	} {
		if v.bc.PlannedCost != nil || v.bc.BudgetUsedPercent != nil || v.bc.OverBudget {
			t.Errorf("%s: budget and cost = %+v, want no planned cost or budget used", name, v.bc)
		}
		if v.progress != (ApprovalSummary{State: "noActivities"}) {
			t.Errorf("%s: progress = %+v, want No activities", name, v.progress)
		}
		if v.goals != (GoalSums{}) {
			t.Errorf("%s: goals = %+v, want none", name, v.goals)
		}
	}
	if view.Initiatives[0].ID != i.ID || f.list(t, "P-1001").Totals.PlannedCost != nil {
		t.Error("the plan's planned cost is not nil with nothing below")
	}
}

// A campaign shows what it allocated to its initiatives and what remains for them (FR-010): 700,000 −
// (300,000 + 250,000) = 150,000.00.
func TestCampaignView_RemainingForInitiatives_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", Budget: input("700000.00")})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social", Budget: input("300000.00")})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Print & outdoor", Budget: input("250000.00")})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Without budget"})

	view, _ := f.service.GetCampaign(t.Context(), priya, c.ID)

	bc := view.BudgetAndCost
	if bc.AllocatedToChildren.Amount.String() != "550000.00" || amountText(bc.RemainingForChildren) != "150000.00" || bc.OverAllocatedBy != nil {
		t.Errorf("budget and cost = %+v", bc)
	}
	initiative := view.Initiatives[0].BudgetAndCost
	if initiative.AllocatedToChildren != money.USD(money.Amount{}) {
		t.Errorf("an initiative allocates %+v, want 0.00 USD (activities have no budget)", initiative.AllocatedToChildren)
	}
}

// The parent's remaining budget counts every child's budget, the record's own included (FR-010): the plan
// has 1,200,000 − (700,000 + 450,000) = 50,000.00 left; the campaign 450,000 − (200,000 + 150,000).
func TestViews_ParentRemainingIncludesOwnBudget_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", Budget: input("700000.00")})
	rimula := f.createCampaign(t, daniel, "P-1001", CampaignInput{Name: "Rimula fleet", Budget: input("450000.00")})
	events := f.createInitiative(t, daniel, rimula.ID, InitiativeInput{Name: "Events", Budget: input("200000.00")})
	f.createInitiative(t, daniel, rimula.ID, InitiativeInput{Name: "Retail events", Budget: input("150000.00")})

	tree := f.list(t, "P-1001")

	for _, c := range tree.Items {
		if amountText(c.ParentRemainingBudget) != "50000.00" {
			t.Errorf("%s: remaining plan budget = %s, want 50000.00", c.Name, amountText(c.ParentRemainingBudget))
		}
	}
	got, _ := f.service.GetInitiative(t.Context(), daniel, events.ID)
	if amountText(got.ParentRemainingBudget) != "100000.00" {
		t.Errorf("remaining campaign budget = %s, want 100000.00", amountText(got.ParentRemainingBudget))
	}
}

// Without a parent budget there is nothing to allocate from: the remaining budget is nil, shown "—" (spec
// edge case).
func TestViews_ParentWithoutBudget_Nil_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, alex, "P-1003", CampaignInput{Name: "Brand platform"})
	i := f.createInitiative(t, alex, c.ID, InitiativeInput{Name: "Brand film", Budget: input("250000.00")})

	tree := f.list(t, "P-1003")

	if tree.Items[0].ParentRemainingBudget != nil || tree.Totals.RemainingPlanBudget != nil || tree.Totals.OverAllocatedBy != nil ||
		tree.Totals.PlanBudget != nil {
		t.Errorf("plan without budget: campaign %+v, totals %+v", tree.Items[0].ParentRemainingBudget, tree.Totals)
	}
	if got := tree.Items[0].Initiatives[0]; got.ID != i.ID || got.ParentRemainingBudget != nil || tree.Items[0].BudgetAndCost.RemainingForChildren != nil {
		t.Errorf("campaign without budget: initiative remaining %+v", got.ParentRemainingBudget)
	}
}

// Approval progress counts the activities whose status is approved (shown Approved, In progress or
// Complete): "0 of 5", "3 of 5", All approved (FR-013, US4 scenario 2).
func TestViews_ApprovalProgressCountsApproved_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})
	for _, id := range []string{"A-1", "A-2", "A-3", "A-4", "A-5"} {
		f.activities.add(ActivitySummary{ID: id, InitiativeID: i.ID})
	}
	approve := func(n int) {
		for k := range f.activities.byInitiative[i.ID] {
			f.activities.byInitiative[i.ID][k].Approved = k < n
		}
	}
	tests := []struct {
		approved int
		want     ApprovalSummary
	}{
		{0, ApprovalSummary{State: "inProgress", Approved: 0, Total: 5}},
		{3, ApprovalSummary{State: "inProgress", Approved: 3, Total: 5}},
		{5, ApprovalSummary{State: "allApproved", Approved: 5, Total: 5}},
	}
	for _, tt := range tests {
		approve(tt.approved)

		view, _ := f.service.GetCampaign(t.Context(), priya, c.ID)

		if view.ApprovalProgress != tt.want || view.Initiatives[0].ApprovalProgress != tt.want || view.Counts.ApprovedActivities != tt.approved {
			t.Errorf("%d approved: campaign %+v, initiative %+v; want %+v", tt.approved, view.ApprovalProgress, view.Initiatives[0].ApprovalProgress, tt.want)
		}
	}
}

// Timing compares the dates with today, 12 Mar 2026: Not started, Running, Finished, or nil without dates
// (FR-013, US4 scenario 3).
func TestViews_TimingAgainstToday_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tests := []struct {
		name, start, end string
		want             *string
	}{
		{"Later", "2026-03-13", "2026-12-31", ptr("notStarted")},
		{"Today", "2026-03-12", "2026-03-12", ptr("running")},
		{"Earlier", "2026-01-01", "2026-03-11", ptr("finished")},
		{"Undated", "", "", nil},
	}
	for _, tt := range tests {
		in := CampaignInput{Name: tt.name}
		if tt.start != "" {
			in.StartDate, in.EndDate = day(tt.start), day(tt.end)
		}

		view := f.createCampaign(t, priya, "P-1001", in)

		if !reflect.DeepEqual(view.Timing, tt.want) {
			t.Errorf("%s: timing = %v, want %v", tt.name, deref(view.Timing), deref(tt.want))
		}
	}
}

// deref returns *s, or "<nil>".
func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// Goals are the sums of the activities' INQ, MQL and revenue below; a goal no activity has stays nil
// (FR-016).
func TestViews_GoalSums_FR016(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	first := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})
	second := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Print & outdoor"})
	f.activities.add(
		ActivitySummary{ID: "A-1", InitiativeID: first.ID, INQ: count(1000), MQL: count(250), Revenue: usd("12500.50")},
		ActivitySummary{ID: "A-2", InitiativeID: first.ID, INQ: count(500)},
		ActivitySummary{ID: "A-3", InitiativeID: second.ID, INQ: count(200)},
	)

	view, _ := f.service.GetCampaign(t.Context(), priya, c.ID)

	if *view.Goals.INQ != 1700 || *view.Goals.MQL != 250 || amountText(view.Goals.Revenue) != "12500.50" {
		t.Errorf("campaign goals = %d, %d, %s", *view.Goals.INQ, *view.Goals.MQL, amountText(view.Goals.Revenue))
	}
	print := view.Initiatives[1].Goals
	if *print.INQ != 200 || print.MQL != nil || print.Revenue != nil {
		t.Errorf("Print & outdoor goals = %+v, want INQ 200 only", print)
	}
}

// Nothing derived is stored: when an activity's planned cost or status changes below, the next read shows
// the new planned cost, budget used and progress (FR-012, FR-015, D-090).
func TestViews_FollowChangesBelow_FR012_FR015(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", Budget: input("1000.00")})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})
	f.activities.add(ActivitySummary{ID: "A-1", InitiativeID: i.ID, PlannedCost: usd("500.00")})
	before := f.list(t, "P-1001").Items[0]

	f.activities.byInitiative[i.ID][0].PlannedCost = usd("1100.00")
	f.activities.byInitiative[i.ID][0].Approved = true
	after := f.list(t, "P-1001").Items[0]

	if *before.BudgetAndCost.BudgetUsedPercent != 50 || before.ApprovalProgress.State != "inProgress" {
		t.Errorf("before = %+v, %+v", before.BudgetAndCost, before.ApprovalProgress)
	}
	if amountText(after.BudgetAndCost.PlannedCost) != "1100.00" || *after.BudgetAndCost.BudgetUsedPercent != 110 ||
		!after.BudgetAndCost.OverBudget || after.ApprovalProgress.State != "allApproved" {
		t.Errorf("after = %+v, %+v", after.BudgetAndCost, after.ApprovalProgress)
	}
}
