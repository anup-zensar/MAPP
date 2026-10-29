package dashboard

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"mapp/internal/allwork"
	"mapp/internal/approval"
	"mapp/internal/budget"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
)

// Planned cost by month as Budget & cost spreads it; actuals by posting month of the plans whose finance is
// visible, payments of other years apart; running totals from January (FR-014).
func TestMonths_FR014(t *testing.T) {
	t.Parallel()
	planned := map[string]allwork.Months{
		"A-1": {ByMonth: map[string]money.Amount{"2026-01": amount(t, "100.00"), "2026-03": amount(t, "600.00")}},
		"A-3": {ByMonth: map[string]money.Amount{}, NoDates: amount(t, "100.00")},
		"A-4": {ByMonth: map[string]money.Amount{"2026-03": amount(t, "50.00")}, OtherYears: amount(t, "0.00")},
	}
	paid := map[string]map[string]money.Amount{
		"L-1": {"2026-03": amount(t, "150.00"), "2025-12": amount(t, "50.00")},
		"L-3": {"2026-03": amount(t, "5.00")},
	}

	got := sampleBoard(t, nil).months(planned, paid, 2026, money.DisplayUSD)

	if len(got.Items) != 12 || got.Items[11].Month != "2026-12" {
		t.Fatalf("months = %+v", got.Items)
	}
	march := got.Items[2]
	hidden := sampleBoard(t, func(p *allwork.Node) bool { return p.ID == "P-2" }).months(planned, paid, 2026, money.DisplayUSD)
	same(t, "months", texts("650.00", "150.00", "750.00", "150.00", "100.00", "—", "50.00", "—", "—"),
		texts(march.Planned.Amount, text(march.Actual), march.PlannedToDate.Amount, text(march.ActualToDate), text(got.NoDates),
			text(got.OtherYears), text(got.PaidOutsideYear), text(hidden.Items[2].Actual), text(hidden.PaidOutsideYear)))
}

// The chart counts the activities whose line items exceed their planned cost, the reason its months add up to
// more than the planned cost (038 FR-024, L-2).
func TestMonths_NegativeBalanceCount_038_FR024_L2(t *testing.T) {
	t.Parallel()
	planned := map[string]allwork.Months{
		"A-1": {ByMonth: map[string]money.Amount{"2026-01": amount(t, "100.00"), "2026-03": amount(t, "600.00")},
			OtherYears: amount(t, "50.00")},
		"A-3": {ByMonth: map[string]money.Amount{}, NoDates: amount(t, "100.00")},
		"A-4": {ByMonth: map[string]money.Amount{"2026-03": amount(t, "50.00")}},
	}

	got := sampleBoard(t, nil).months(planned, nil, 2026, money.DisplayUSD)
	exact := sampleBoard(t, func(p *allwork.Node) bool { return p.ID == "P-2" }).months(planned, nil, 2026, money.DisplayUSD)

	same(t, "negative balance counts", texts(1, 0), texts(got.NegativeBalanceCount, exact.NegativeBalanceCount))
}

// The rounds of the board's activities and purchase requests only: waiting since the oldest level start, decided by
// a person, approved automatically, and the average days to decide (FR-016).
func TestApprovals_FR016(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	at := func(hours int) *time.Time { moment := start.Add(time.Duration(hours) * time.Hour); return &moment }
	rounds := []approval.RoundSummary{
		{SubjectID: "A-1", Outcome: approval.OutcomeApproved, SubmittedAt: start, EndedAt: at(24)},
		{SubjectID: "A-3", Outcome: approval.OutcomeDeclined, SubmittedAt: start, EndedAt: at(48)},
		{SubjectID: "A-3", Outcome: approval.OutcomeOpen, SubmittedAt: *at(50), WaitingSince: at(60)},
		{SubjectID: "PR-1", Outcome: approval.OutcomeOpen, SubmittedAt: start, WaitingSince: at(10)},
		{SubjectID: "A-4", Outcome: approval.OutcomeApproved, AutoApproved: true, SubmittedAt: start, EndedAt: &start},
		{SubjectID: "A-9", Outcome: approval.OutcomeOpen, SubmittedAt: start, WaitingSince: &start},
		{SubjectID: "A-1", Outcome: approval.OutcomeRecalled, SubmittedAt: start, EndedAt: at(1)},
	}

	got := sampleBoard(t, nil).approvals(rounds, []purchaserequest.Summary{{ID: "PR-1", PlanID: "P-1"}})

	if got.Waiting != 2 || !got.OldestWaitingSince.Equal(*at(10)) || got.Approved != 1 || got.Declined != 1 ||
		got.AutoApproved != 1 || got.AverageDecisionDays == nil || *got.AverageDecisionDays != "1.5" {
		t.Errorf("approvals = %+v", got)
	}
	if got.ActivitiesByStatus[1] != (StatusCount{Status: "submitted", Count: 2}) {
		t.Errorf("by status = %+v", got.ActivitiesByStatus)
	}
}

// Purchase requests on the board's plans by status, their amounts and those waiting for GSAP (FR-017).
func TestPurchaseRequests_FR017(t *testing.T) {
	t.Parallel()
	summaries := purchaserequest.Summaries{SwitchedOn: true, Items: []purchaserequest.Summary{
		{ID: "PR-1", PlanID: "P-1", Status: purchaserequest.StatusApproved, AmountUSD: amount(t, "100.00"), WaitingForGSAP: true},
		{ID: "PR-2", PlanID: "P-2", Status: purchaserequest.StatusApproved, AmountUSD: amount(t, "50.00")},
		{ID: "PR-3", PlanID: "P-9", Status: purchaserequest.StatusCreated, AmountUSD: amount(t, "999.00")},
	}}

	got := sampleBoard(t, nil).purchaseRequests(summaries, money.DisplayUSD)

	approved := got.ByStatus[4]
	if !got.Available || got.Count != 2 || got.Amount.Amount.String() != "150.00" || got.WaitingForGSAP != 1 ||
		approved.Status != "approved" || approved.Count != 2 || got.ByStatus[0].Count != 0 {
		t.Errorf("purchase requests = %+v", got)
	}
}

func TestGlobalBudgets_ShownInTheCurrency_FR004(t *testing.T) {
	t.Parallel()
	usd := func(text string) money.Money { return money.USD(amount(t, text)) }
	page := budget.Page{Items: []budget.Summary{{ID: "GB-1", Name: "Global", Figures: budget.Figures{
		Amount: usd("100.00"), Allocated: usd("120.00"), Unallocated: usd("-20.00"), PlannedCost: usd("50.00"), OverAllocated: true}}}}

	got := globalBudgets(page, money.DisplayIn("EUR", "0.50"))

	if len(got) != 1 || got[0].Amount.Format() != "50.00 EUR" || got[0].Unallocated.Format() != "-10.00 EUR" || !got[0].OverAllocated {
		t.Errorf("global budgets = %+v", got)
	}
}

func TestParseQuery_FR006(t *testing.T) {
	t.Parallel()
	q, err := ParseQuery(url.Values{"year": {"2027"}, "planType": {"strategic"}, "currency": {"EUR"}, "teamId": {"T-1001"}})
	if err != nil || q != (Query{Year: 2027, PlanType: "strategic", Currency: "EUR", TeamID: "T-1001"}) {
		t.Errorf("ParseQuery = %+v, %v", q, err)
	}
	_, err = ParseQuery(url.Values{"year": {"1999"}, "planType": {"other"}, "currency": {"eur"}})
	var fields []string
	if e, ok := errors.AsType[*errs.Error](err); ok {
		for _, f := range e.Fields {
			fields = append(fields, f.Field)
		}
	}
	same(t, "every wrong filter at once", []string{"year", "planType", "currency"}, fields)
	if _, err := ParseFormat(url.Values{"format": {"docx"}}); err == nil {
		t.Errorf("format docx accepted")
	}
}
