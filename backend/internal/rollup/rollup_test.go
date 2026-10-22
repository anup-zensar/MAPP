package rollup

import (
	"testing"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// usd returns a pointer to the USD amount text ("150000.00"); "" gives nil (no value).
func usd(text string) *money.Money {
	if text == "" {
		return nil
	}
	amount, err := money.ParseAmount(text)
	if err != nil {
		panic(err)
	}
	m := money.USD(amount)
	return &m
}

// text writes an optional amount as its digits, "" for nil.
func text(m *money.Money) string {
	if m == nil {
		return ""
	}
	return m.Amount.String()
}

// percentValue returns an optional percent, -1 for nil.
func percentValue(p *int) int {
	if p == nil {
		return -1
	}
	return *p
}

// Sum adds only the amounts that are present; with none present it is nil, shown "—"; 0.00 is a real
// value (FR-011).
func TestSum_SkipsEmpty_NilWhenNothing_FR011(t *testing.T) {
	if got := text(Sum([]*money.Money{usd("100000.00"), nil, usd("50000.00")})); got != "150000.00" {
		t.Errorf("Sum = %q, want 150000.00", got)
	}
	if got := Sum([]*money.Money{nil, nil}); got != nil {
		t.Errorf("Sum of nothing = %v, want nil", got)
	}
	if got := Sum(nil); got != nil {
		t.Errorf("Sum of no list = %v, want nil", got)
	}
	if got := text(Sum([]*money.Money{usd("0.00")})); got != "0.00" {
		t.Errorf("Sum of 0.00 = %q, want 0.00 (a real value)", got)
	}
	if got := Sum([]*money.Money{usd("1.00")}); got.Currency != "USD" {
		t.Errorf("currency = %q, want USD", got.Currency)
	}
}

// Total is 0.00 USD when nothing is allocated: the totals row always has an amount (FR-022).
func TestTotal_ZeroWhenNothing_FR022(t *testing.T) {
	if got := Total(nil); got != money.USD(money.Amount{}) {
		t.Errorf("Total(nil) = %+v, want 0.00 USD", got)
	}
	if got := Total([]*money.Money{usd("700000.00"), nil, usd("450000.00")}); got.Amount.String() != "1150000.00" {
		t.Errorf("Total = %s, want 1150000.00", got.Amount)
	}
}

// Over-allocating is allowed and shown: 800,000 − (600,000 + 300,000) = −100,000.00, over by 100,000.00
// (US1 scenario 3, D-015).
func TestRemaining_NegativeAllowed_FR010(t *testing.T) {
	remaining := Remaining(usd("800000.00"), Total([]*money.Money{usd("600000.00"), usd("300000.00")}))
	if got := text(remaining); got != "-100000.00" {
		t.Errorf("Remaining = %q, want -100000.00", got)
	}
	if got := text(OverBy(remaining)); got != "100000.00" {
		t.Errorf("OverBy = %q, want 100000.00", got)
	}
	if got := OverBy(usd("0.00")); got != nil {
		t.Errorf("OverBy(0.00) = %v, want nil", got)
	}
	if got := OverBy(usd("5.00")); got != nil {
		t.Errorf("OverBy(5.00) = %v, want nil", got)
	}
}

// Without a budget there is nothing to allocate from: remaining is nil, shown "—" (FR-010).
func TestRemaining_NilWithoutBudget_FR010(t *testing.T) {
	if got := Remaining(nil, *usd("100.00")); got != nil {
		t.Errorf("Remaining without budget = %v, want nil", got)
	}
	if got := OverBy(nil); got != nil {
		t.Errorf("OverBy(nil) = %v, want nil", got)
	}
}

// Budget used is a whole percent, half up; nil without a budget, with a budget of 0.00 or without a planned
// cost (FR-011).
func TestBudgetUsedPercent_RoundsHalfUp_FR011(t *testing.T) {
	tests := []struct {
		name, planned, budget string
		want                  int
	}{
		{"150,000 of 120,000", "150000.00", "120000.00", 125},
		{"124.5 % rounds up", "1245.00", "1000.00", 125},
		{"124.4 % rounds down", "1244.00", "1000.00", 124},
		{"99.4 % rounds down", "994.00", "1000.00", 99},
		{"99.5 % rounds up", "995.00", "1000.00", 100},
		{"nothing used", "0.00", "1000.00", 0},
		{"no budget", "100.00", "", -1},
		{"budget 0.00", "100.00", "0.00", -1},
		{"no planned cost", "", "1000.00", -1},
		{"huge amounts do not overflow", "999999999999999.99", "1000000.00", 100000000000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentValue(BudgetUsedPercent(usd(tt.planned), usd(tt.budget))); got != tt.want {
				t.Errorf("BudgetUsedPercent(%s, %s) = %d, want %d", tt.planned, tt.budget, got, tt.want)
			}
		})
	}
}

// Over budget compares exactly: 100.4 % shows "100 %" but in red; equal is not over (FR-011).
func TestOverBudget_ExactComparison_FR011(t *testing.T) {
	planned, budget := usd("1004.00"), usd("1000.00")
	if got := percentValue(BudgetUsedPercent(planned, budget)); got != 100 || !OverBudget(planned, budget) {
		t.Errorf("100.4 %%: percent %d, over %v; want 100 and over", got, OverBudget(planned, budget))
	}
	if OverBudget(usd("1000.00"), usd("1000.00")) {
		t.Error("planned = budget counts as over budget, want not over")
	}
	if !OverBudget(usd("0.01"), usd("0.00")) {
		t.Error("planned 0.01 with budget 0.00 is not over budget, want over")
	}
	if OverBudget(nil, usd("1.00")) || OverBudget(usd("1.00"), nil) {
		t.Error("over budget without both values, want not over")
	}
}

// Left to plan is the budget minus the planned cost (none counts as 0.00); nil without a budget; negative
// when over budget (FR-011).
func TestLeftToPlan_FR011(t *testing.T) {
	tests := []struct{ budget, planned, want string }{
		{"1000.00", "250.00", "750.00"},
		{"1000.00", "", "1000.00"},
		{"1000.00", "1250.50", "-250.50"},
		{"", "250.00", ""},
	}
	for _, tt := range tests {
		if got := text(LeftToPlan(usd(tt.budget), usd(tt.planned))); got != tt.want {
			t.Errorf("LeftToPlan(%q, %q) = %q, want %q", tt.budget, tt.planned, got, tt.want)
		}
	}
}

// Approval progress: No activities, In progress (also 0 of 5), All approved (FR-013, US4 scenario 2).
func TestApprovalProgress_States_FR013(t *testing.T) {
	tests := []struct {
		approved, total int
		want            string
	}{
		{0, 0, NoActivities},
		{0, 5, InProgress},
		{3, 5, InProgress},
		{5, 5, AllApproved},
	}
	for _, tt := range tests {
		got := ApprovalProgress(tt.approved, tt.total)
		if got != (Progress{State: tt.want, Approved: tt.approved, Total: tt.total}) {
			t.Errorf("ApprovalProgress(%d, %d) = %+v, want state %s", tt.approved, tt.total, got, tt.want)
		}
	}
}

// Timing against today: Not started before the start, Running from the start to the end included,
// Finished after; "" without dates (FR-013, US4 scenario 3).
func TestTiming_BeforeOnStartWithinOnEndAfter_FR013(t *testing.T) {
	start, end := date.MustParse("2026-03-01"), date.MustParse("2026-12-31")
	tests := []struct {
		today string
		want  string
	}{
		{"2026-02-28", NotStarted},
		{"2026-03-01", Running},
		{"2026-06-15", Running},
		{"2026-12-31", Running},
		{"2027-01-01", Finished},
	}
	for _, tt := range tests {
		if got := Timing(&start, &end, date.MustParse(tt.today)); got != tt.want {
			t.Errorf("Timing on %s = %q, want %q", tt.today, got, tt.want)
		}
	}
	if got := Timing(nil, nil, start); got != "" {
		t.Errorf("Timing without dates = %q, want \"\"", got)
	}
	if got := Timing(&start, nil, start); got != "" {
		t.Errorf("Timing with one date = %q, want \"\"", got)
	}
}

// Goal sums add what is there; a goal nothing below has stays nil, shown "—" (FR-016).
func TestAddGoals_NilWhenNothing_FR016(t *testing.T) {
	inq, mql := int64(1000), int64(250)
	more := int64(500)
	got := AddGoals([]Goals{
		{INQ: &inq, MQL: &mql, Revenue: usd("10000.00")},
		{INQ: &more},
		{},
	})
	if got.INQ == nil || *got.INQ != 1500 || got.MQL == nil || *got.MQL != 250 || text(got.Revenue) != "10000.00" {
		t.Errorf("AddGoals = %+v, want INQ 1500, MQL 250, revenue 10000.00", got)
	}
	if none := AddGoals([]Goals{{}, {}}); none.INQ != nil || none.MQL != nil || none.Revenue != nil {
		t.Errorf("AddGoals of empty goals = %+v, want all nil", none)
	}
}

// Every container figure equals the hand-calculated value to the cent (SC-004).
func TestContainer_HandCalculated_SC004(t *testing.T) {
	tests := []struct {
		name       string
		budget     string
		children   []string
		planned    string
		allocated  string
		remaining  string
		over       string
		percent    int
		overBudget bool
		leftToPlan string
	}{
		{"US4-1: 150,000 under 120,000", "120000.00", nil, "150000.00", "0.00", "120000.00", "", 125, true, "-30000.00"},
		{"Helix launch", "700000.00", []string{"300000.00", "250000.00"}, "", "550000.00", "150000.00", "", -1, false, "700000.00"},
		{"Lubricants Europe plan", "1200000.00", []string{"700000.00", "450000.00"}, "", "1150000.00", "50000.00", "", -1, false, "1200000.00"},
		{"Sponsorships over-allocated", "300000.00", []string{"250000.00", "100000.00"}, "", "350000.00", "-50000.00", "50000.00", -1, false, "300000.00"},
		{"US1-3: 600,000 + 300,000 of 800,000", "800000.00", []string{"600000.00", "300000.00"}, "", "900000.00", "-100000.00", "100000.00", -1, false, "800000.00"},
		{"no budget", "", []string{"100.00"}, "50.00", "100.00", "", "", -1, false, ""},
		{"nothing at all", "", nil, "", "0.00", "", "", -1, false, ""},
		{"budget 0.00 with cost", "0.00", nil, "0.01", "0.00", "0.00", "", -1, true, "-0.01"},
		{"cents", "1000.10", []string{"333.33", "333.33", "333.34"}, "999.99", "1000.00", "0.10", "", 100, false, "0.11"},
		{"empty children skipped", "500.00", []string{"", "125.25"}, "0.00", "125.25", "374.75", "", 0, false, "500.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			children := make([]*money.Money, 0, len(tt.children))
			for _, c := range tt.children {
				children = append(children, usd(c))
			}

			got := Container(usd(tt.budget), children, usd(tt.planned))

			if got.AllocatedToChildren.Amount.String() != tt.allocated || got.AllocatedToChildren.Currency != "USD" {
				t.Errorf("allocated = %+v, want %s USD", got.AllocatedToChildren, tt.allocated)
			}
			if text(got.RemainingForChildren) != tt.remaining {
				t.Errorf("remaining = %q, want %q", text(got.RemainingForChildren), tt.remaining)
			}
			if text(got.OverAllocatedBy) != tt.over {
				t.Errorf("over-allocated by = %q, want %q", text(got.OverAllocatedBy), tt.over)
			}
			if text(got.PlannedCost) != tt.planned {
				t.Errorf("planned cost = %q, want %q", text(got.PlannedCost), tt.planned)
			}
			if percentValue(got.BudgetUsedPercent) != tt.percent {
				t.Errorf("budget used = %d, want %d", percentValue(got.BudgetUsedPercent), tt.percent)
			}
			if got.OverBudget != tt.overBudget {
				t.Errorf("over budget = %v, want %v", got.OverBudget, tt.overBudget)
			}
			if text(got.LeftToPlan) != tt.leftToPlan {
				t.Errorf("left to plan = %q, want %q", text(got.LeftToPlan), tt.leftToPlan)
			}
		})
	}
}
