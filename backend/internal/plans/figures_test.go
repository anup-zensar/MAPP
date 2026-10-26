package plans

import (
	"reflect"
	"testing"

	"mapp/internal/people"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// planWithBudget is a plan with a budget in USD; "" for none.
func planWithBudget(t *testing.T, budget string) Plan {
	t.Helper()
	p := Plan{ID: "P-1001", Name: "FY2026 Lubricants Europe"}
	if budget != "" {
		amount := usd(t, budget)
		p.Budget = &amount
	}
	return p
}

// moneyOf returns an optional amount in a currency, for the expected figures.
func moneyOf(t *testing.T, amount, currency string) *money.Money {
	t.Helper()
	return &money.Money{Amount: usd(t, amount), Currency: currency}
}

// Remaining budget = plan budget − campaign budgets, negative when the campaigns got more than the plan
// has: 1,200,000.00 − 1,300,000.00 = −100,000.00, over-allocated by 100,000.00; without a plan budget there
// is no remaining budget (FR-012, D-015, US3 scenario 2).
func TestFigures_RemainingBudgetNegative_FR012(t *testing.T) {
	t.Parallel()
	overAllocated := figuresOf(planWithBudget(t, "1200000.00"), Facts{CampaignBudgets: usd(t, "1300000.00")}, usdConverter)
	within := figuresOf(planWithBudget(t, "1200000.00"), Facts{CampaignBudgets: usd(t, "1000000.00")}, usdConverter)
	noBudget := figuresOf(planWithBudget(t, ""), Facts{CampaignBudgets: usd(t, "1000000.00")}, usdConverter)

	if !reflect.DeepEqual(overAllocated.RemainingBudget, moneyOf(t, "-100000.00", "USD")) ||
		!reflect.DeepEqual(overAllocated.OverAllocatedBy, moneyOf(t, "100000.00", "USD")) {
		t.Errorf("over-allocated: remaining %v, over-allocated by %v", overAllocated.RemainingBudget, overAllocated.OverAllocatedBy)
	}
	if !reflect.DeepEqual(within.RemainingBudget, moneyOf(t, "200000.00", "USD")) || within.OverAllocatedBy != nil {
		t.Errorf("within: remaining %v, over-allocated by %v", within.RemainingBudget, within.OverAllocatedBy)
	}
	if noBudget.RemainingBudget != nil || noBudget.OverAllocatedBy != nil || noBudget.Budget != nil {
		t.Errorf("no budget: %+v, want no budget, remaining or over-allocation", noBudget)
	}
	if !reflect.DeepEqual(noBudget.CampaignBudgets, *moneyOf(t, "1000000.00", "USD")) {
		t.Errorf("campaign budgets = %v", noBudget.CampaignBudgets)
	}
}

// Budget used = planned cost ÷ plan budget as a whole percent: 960,000 of 1,200,000 → 80; 1,300,000 →
// 108 and over budget. Without a budget or with a budget of 0 it is empty; a budget of 0 with a planned
// cost is still over budget (FR-012, D-088, US3 scenario 3).
func TestFigures_BudgetUsed_FR012(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, budget, planned string
		wantPercent           *int
		wantOver              bool
	}{
		{"80 %", "1200000.00", "960000.00", ptr(80), false},
		{"108 %", "1200000.00", "1300000.00", ptr(108), true},
		{"nothing planned yet", "1200000.00", "", ptr(0), false},
		{"no budget", "", "960000.00", nil, false},
		{"budget 0", "0.00", "10.00", nil, true},
		{"budget 0, nothing planned", "0.00", "", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f Facts
			if tt.planned != "" {
				planned := usd(t, tt.planned)
				f.PlannedCost = &planned
			}

			got := figuresOf(planWithBudget(t, tt.budget), f, usdConverter)

			if !reflect.DeepEqual(got.BudgetUsedPercent, tt.wantPercent) || got.OverBudget != tt.wantOver {
				t.Errorf("budget used %v, over budget %v; want %v, %v", deref(got.BudgetUsedPercent), got.OverBudget, deref(tt.wantPercent), tt.wantOver)
			}
		})
	}
}

// Activities approved are "n of m" with the progress: no activities, in progress, all approved (FR-012,
// D-026).
func TestFigures_ApprovalProgress_FR012(t *testing.T) {
	t.Parallel()
	tests := []struct {
		activities, approved int
		want                 ApprovalProgress
	}{
		{0, 0, ProgressNoActivities},
		{7, 4, ProgressInProgress},
		{7, 0, ProgressInProgress},
		{7, 7, ProgressAllApproved},
	}
	for _, tt := range tests {
		got := figuresOf(planWithBudget(t, ""), Facts{Activities: tt.activities, ApprovedActivities: tt.approved}, usdConverter)

		if got.Activities != tt.activities || got.ActivitiesApproved != tt.approved || got.ApprovalProgress != tt.want {
			t.Errorf("%d of %d: %d of %d, %s; want %s", tt.approved, tt.activities, got.ActivitiesApproved, got.Activities, got.ApprovalProgress, tt.want)
		}
	}
}

// Every amount is shown in the reader's display currency, converted from USD at the end and rounded half
// up to the cent: EUR at 0.92 (FR-015, D-079). An unusable rate falls back to USD.
func TestFigures_DisplayCurrency_FR015(t *testing.T) {
	t.Parallel()
	eur := converterOf(people.DisplayRate{Code: "EUR", Rate: "0.92"})
	planned := usd(t, "0.05") // 0.046 EUR → 0.05
	f := Facts{CampaignBudgets: usd(t, "1300000.00"), PlannedCost: &planned}

	got := figuresOf(planWithBudget(t, "1200000.00"), f, eur)

	want := Figures{
		Currency: "EUR", Budget: moneyOf(t, "1104000.00", "EUR"), PlannedCost: moneyOf(t, "0.05", "EUR"),
		CampaignBudgets: *moneyOf(t, "1196000.00", "EUR"), RemainingBudget: moneyOf(t, "-92000.00", "EUR"),
		OverAllocatedBy: moneyOf(t, "92000.00", "EUR"), BudgetUsedPercent: ptr(0), ApprovalProgress: ProgressNoActivities,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("figures = %+v\nwant      %+v", got, want)
	}
	if fallback := converterOf(people.DisplayRate{Code: "EUR", Rate: "not a rate"}); fallback != usdConverter {
		t.Errorf("unusable rate: %+v, want USD", fallback)
	}
}

// The completeness chip lists the plan's own needed fields in form order — Teams, Owner (empty, or in
// none of the plan's teams), the needed Shell attributes, Goal stage — and every record below that is not
// complete (FR-017, D-085).
func TestCompleteness_OwnFieldsAndItemsBelow_FR017(t *testing.T) {
	t.Parallel()
	businessUnitMissing := []settings.MissingField{{Field: "attributes.SA-BU", Label: "Business unit"}}
	teams := settings.MissingField{Field: "teamIds", Label: "Teams"}
	owner := settings.MissingField{Field: "ownerId", Label: "Owner"}
	stage := settings.MissingField{Field: "goalStage", Label: "Goal stage"}
	tests := []struct {
		name       string
		plan       Plan
		ownerOK    bool
		attributes []settings.MissingField
		want       []settings.MissingField
	}{
		{"all filled in", Plan{TeamIDs: []string{teamLubricants}, OwnerID: priya.ID, GoalStage: StageINQ}, true, nil, []settings.MissingField{}},
		{"everything missing, in form order", Plan{}, false, businessUnitMissing, []settings.MissingField{teams, owner, businessUnitMissing[0], stage}},
		{"owner in none of the teams", Plan{TeamIDs: []string{teamMobility}, OwnerID: priya.ID, GoalStage: StageINQ}, false, nil, []settings.MissingField{owner}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ownMissing(tt.plan, tt.ownerOK, tt.attributes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("missing = %+v, want %+v", got, tt.want)
			}
		})
	}

	campaign := AttentionItem{Record: errs.RecordRef{Type: "campaign", ID: "C-1001", Name: "Retail growth"},
		Missing: []settings.MissingField{{Field: "attributes.SA-1003", Label: "Cluster"}}, BlocksPublish: true}
	activity := AttentionItem{Record: errs.RecordRef{Type: "activity", ID: "A-1001", Name: "Launch film"}}
	got := itemsBelow(Facts{NeedsAttention: []AttentionItem{campaign, activity}})
	activity.Missing = []settings.MissingField{} // never null in the answer
	if !reflect.DeepEqual(got, []AttentionItem{campaign, activity}) {
		t.Errorf("items below = %+v", got)
	}
	if none := itemsBelow(Facts{}); none == nil || len(none) != 0 {
		t.Errorf("items below of nothing = %#v, want an empty list", none)
	}
}
