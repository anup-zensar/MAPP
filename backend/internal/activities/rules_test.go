package activities

import (
	"encoding/json"
	"reflect"
	"testing"

	"northstar/internal/campaigns"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// The pure rules of activities, without storage: dates, names, goals, display status and the
// ready-to-submit check. The service tests (package activities_test) prove the same rules through the
// service on the demo data.

// digitalAndSocial is an initiative running 1 Jan – 30 Jun 2026.
var digitalAndSocial = campaigns.Initiative{
	ID: "I-1001", Name: "Digital & social", StartDate: day("2026-01-01"), EndDate: day("2026-06-30"),
}

// day returns the date of text, YYYY-MM-DD.
func day(text string) *date.Date {
	d := date.MustParse(text)
	return &d
}

// usd returns the US-dollar money of text, such as "120000.00".
func usd(text string) *money.Money {
	amount, err := money.ParseAmount(text)
	if err != nil {
		panic(err)
	}
	m := money.USD(amount)
	return &m
}

// The end on or after the start; both inside the initiative's dates when it has both (R3), only the start
// for an ongoing activity; the end at most 31 Dec of the plan's year + 10 (R5). Every problem at once.
func TestCheckDates_FR009_FR010(t *testing.T) {
	latest := latestEndDate(2026)
	noDates := campaigns.Initiative{ID: "I-1006", Name: "Fleet cards"}
	tests := []struct {
		name       string
		start, end *date.Date
		ongoing    bool
		initiative campaigns.Initiative
		want       errs.Problems
	}{
		{"inside", day("2026-03-01"), day("2026-05-31"), false, digitalAndSocial, nil},
		{"the initiative's own dates", day("2026-01-01"), day("2026-06-30"), false, digitalAndSocial, nil},
		{"end before start", day("2026-05-31"), day("2026-03-01"), false, digitalAndSocial, errs.Problems{
			{Field: "endDate", Message: "End date must be on or after the start date (31 May 2026)."}}},
		{"outside the initiative", day("2025-12-31"), day("2026-07-01"), false, digitalAndSocial, errs.Problems{
			{Field: "startDate", Message: "Dates must be inside the initiative's dates (01 Jan 2026 – 30 Jun 2026)."},
			{Field: "endDate", Message: "Dates must be inside the initiative's dates (01 Jan 2026 – 30 Jun 2026)."}}},
		{"ongoing: only the start counts", day("2026-03-01"), day("2027-01-01"), true, digitalAndSocial, nil},
		{"ongoing, start outside", day("2025-12-01"), nil, true, digitalAndSocial, errs.Problems{
			{Field: "startDate", Message: "Dates must be inside the initiative's dates (01 Jan 2026 – 30 Jun 2026)."}}},
		{"initiative without dates", day("2020-01-01"), day("2036-12-31"), false, noDates, nil},
		{"after the latest end (R5)", day("2026-01-01"), day("2037-01-01"), false, noDates, errs.Problems{
			{Field: "endDate", Message: "The end date can't be after 31 Dec 2036 — 10 years after the plan's year."}}},
		{"only a start", day("2026-02-01"), nil, false, digitalAndSocial, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var problems errs.Problems
			checkDates(&problems, tt.start, tt.end, tt.ongoing, tt.initiative, latest)
			if !reflect.DeepEqual(problems, tt.want) {
				t.Errorf("problems = %+v, want %+v", problems, tt.want)
			}
		})
	}
}

// Names are the same ignoring letter case and surrounding spaces (FR-006); a restored activity whose name is
// taken gets " (restored)", " (restored 2)", … (FR-031).
func TestNames_FR006_FR031(t *testing.T) {
	if !sameName(" launch FILM ", "Launch film") || sameName("Launch film", "Launch films") {
		t.Error("sameName must ignore case and surrounding spaces only")
	}
	siblings := []Activity{{ID: "A-1", Name: "Launch film"}, {ID: "A-2", Name: "Launch film (restored)"}}
	if got := restoredName(siblings, "Launch film", "A-9"); got != "Launch film (restored 2)" {
		t.Errorf("restoredName = %q, want %q", got, "Launch film (restored 2)")
	}
	if got := restoredName(siblings[:1], "Launch film", "A-9"); got != "Launch film (restored)" {
		t.Errorf("restoredName = %q, want %q", got, "Launch film (restored)")
	}
}

// lubricants is the model "Lubricants 1.0": 2.00 %, 5.00 %, 10,000.00 USD.
var lubricants = settings.Model{
	ID: "MOD-1001", Name: "Lubricants 1.0", InqToMqlRate: money.PercentFromHundredths(200),
	MqlToDealRate: money.PercentFromHundredths(500), AverageDealSize: *usd("10000.00"), Active: true,
}

// MQL, deals and revenue come from INQ and the model's values (domain model §4.5): 250,000 → 5,000 → 250 →
// 2,500,000.00 USD; without INQ they are empty (FR-018).
func TestGoals_CalculatedFromModel_FR018(t *testing.T) {
	inq := int64(250_000)
	mql, deals := int64(5_000), int64(250)
	got := goalsOf(Activity{INQ: &inq}, lubricants)
	want := Goals{INQ: &inq, MQL: &mql, Deals: &deals, Revenue: usd("2500000.00"), Calculated: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("goals = %+v, want %+v", got, want)
	}
	if got := goalsOf(Activity{}, lubricants); !reflect.DeepEqual(got, Goals{Calculated: true}) {
		t.Errorf("goals without INQ = %+v, want all empty and calculated", got)
	}
	small := int64(25) // 2 % of 25 = 0.5 → 1 MQL (half-up); 5 % of 1 = 0.05 → 0 deals
	got = goalsOf(Activity{INQ: &small}, lubricants)
	if *got.MQL != 1 || *got.Deals != 0 || got.Revenue.Amount.Sign() != 0 {
		t.Errorf("goals of INQ 25 = %d MQL, %d deals, %s; want 1, 0, 0.00", *got.MQL, *got.Deals, got.Revenue.Format())
	}
}

// With "Set later stages myself" the typed MQL and revenue are kept and nothing is calculated; without it,
// typed values are not stored (FR-018).
func TestGoals_TypedValuesKept_FR018(t *testing.T) {
	inq, mql := int64(60_000), int64(2_000)
	typed := Activity{INQ: &inq, SetLaterStagesMyself: true, MQL: &mql, Revenue: usd("400000.00")}
	want := Goals{INQ: &inq, SetLaterStagesMyself: true, MQL: &mql, Revenue: usd("400000.00")}
	if got := goalsOf(typed, lubricants); !reflect.DeepEqual(got, want) {
		t.Errorf("typed goals = %+v, want %+v", got, want)
	}

	var problems errs.Problems
	stored := parseGoals(&problems, GoalsInput{INQ: &inq, MQL: &mql, Revenue: &money.Input{Amount: "1.00", Currency: "USD"}})
	if stored.MQL != nil || stored.Revenue != nil || len(problems) != 0 {
		t.Errorf("not typed: stored %+v, problems %v; want MQL and revenue not stored", stored, problems)
	}
}

// An approved activity shows In progress from its start date and Complete after its end date, never
// Complete when ongoing; other statuses show as stored (FR-021, D-091).
func TestDisplayStatus_FR021(t *testing.T) {
	today := date.MustParse("2026-03-12")
	tests := []struct {
		name     string
		activity Activity
		want     DisplayStatus
	}{
		{"approved, before the start", Activity{Status: StatusApproved, StartDate: day("2026-03-13"), EndDate: day("2026-05-31")}, DisplayApproved},
		{"approved, on the start", Activity{Status: StatusApproved, StartDate: day("2026-03-12"), EndDate: day("2026-05-31")}, DisplayInProgress},
		{"approved, on the end", Activity{Status: StatusApproved, StartDate: day("2026-01-01"), EndDate: day("2026-03-12")}, DisplayInProgress},
		{"approved, after the end", Activity{Status: StatusApproved, StartDate: day("2026-01-01"), EndDate: day("2026-03-11")}, DisplayComplete},
		{"approved, ongoing", Activity{Status: StatusApproved, StartDate: day("2025-01-01"), Ongoing: true}, DisplayInProgress},
		{"approved, no dates", Activity{Status: StatusApproved}, DisplayApproved},
		{"submitted, after the end", Activity{Status: StatusSubmitted, EndDate: day("2026-01-01")}, DisplaySubmitted},
		{"created", Activity{Status: StatusCreated}, DisplayCreated},
		{"declined", Activity{Status: StatusDeclined}, DisplayDeclined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayStatus(tt.activity, today); got != tt.want {
				t.Errorf("displayStatus = %s, want %s", got, tt.want)
			}
		})
	}
}

// complete is an activity with every fixed needed-to-submit field filled.
func complete() Activity {
	return Activity{
		ID: "A-1001", Name: "Launch film", OwnerID: "U-1002", StartDate: day("2026-03-01"), EndDate: day("2026-05-31"),
		PlannedCost: usd("1000.00"), ApprovalWorkflowID: "WF-1001", Attributes: settings.AttributeValues{},
	}
}

// readyInput is the check input of a complete activity whose line items use all of its planned cost.
func readyInput() checkInput {
	return checkInput{
		activity: complete(), initiative: digitalAndSocial, typeName: "Event", typeAllowed: true, modelName: "Lubricants 1.0",
		workflow:  &WorkflowFacts{ID: "WF-1001", Name: "Lubricants EU approval", Active: true},
		lineItems: LineItemFacts{Count: 1, PlannedCostPlan: usd("1000.00").Amount},
	}
}

// Every problem that stops submitting is listed at once, grouped and in order (FR-023): the empty cost
// centre and the 90.00 % split under "This activity"; the negative balance, then the line items' problems.
func TestCheck_ListsEveryProblemAtOnce_FR023(t *testing.T) {
	in := readyInput()
	in.activity.Attributes = settings.AttributeValues{
		"SA-1013": json.RawMessage(`[{"optionId":"SO-1030","share":"60.00"},{"optionId":"SO-1031","share":"30.00"}]`),
	}
	in.missing = []settings.MissingField{
		{Field: "attributes.SA-1013", Label: "Country / OU", Reason: "Shares add up to 90.00 % — they must add up to 100.00 %."},
		{Field: "attributes.SA-1015", Label: "Cost centre"},
	}
	in.lineItems = LineItemFacts{Count: 2, PlannedCostPlan: usd("1500.00").Amount, Problems: []LineItemProblem{
		{LineItemID: "L-1001", LineItemName: "Film shoot", Code: "missingField", Field: "attributes.SA-1033", Message: "GL code — needed to submit"},
		{LineItemID: "L-1002", LineItemName: "Edit", Code: "customSpreadMismatch", Field: "spread", Message: "The months add up to 900.00 of 1,000.00"},
	}}

	got := check(in)

	l1001, film, l1002, edit := "L-1001", "Film shoot", "L-1002", "Edit"
	want := SubmitCheck{
		Ready: false, ProblemCount: 5,
		Activity: []SubmitProblem{
			{Code: ProblemSplitIncomplete, Field: "attributes.SA-1013", Message: "Country / OU shares add up to 90.00 % — they must add up to 100.00 %"},
			{Code: ProblemMissingField, Field: "attributes.SA-1015", Message: "Cost centre — needed to submit"},
		},
		LineItems: []SubmitProblem{
			{Code: ProblemNegativeUnassigned, Field: "lineItems", Message: "Line items exceed the activity's planned cost by 500.00 USD"},
			{Code: "missingField", Field: "attributes.SA-1033", LineItemID: &l1001, LineItemName: &film,
				Message: "Line item 'Film shoot' (L-1001): GL code — needed to submit"},
			{Code: "customSpreadMismatch", Field: "spread", LineItemID: &l1002, LineItemName: &edit,
				Message: "Line item 'Edit' (L-1002): The months add up to 900.00 of 1,000.00"},
		},
		Advice: []SubmitProblem{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("check =\n%+v\nwant\n%+v", got, want)
	}
	if c := completeness(got, in.lineItems); c != (Completeness{MissingFields: 2, LineItemsNeedingAttention: 2}) {
		t.Errorf("completeness = %+v, want 2 fields and 2 line items (FR-022)", c)
	}
}

// With nothing missing the check is ready, with empty lists (FR-023).
func TestCheck_ReadyWhenNothingMissing_FR023(t *testing.T) {
	got := check(readyInput())
	want := SubmitCheck{Ready: true, Activity: []SubmitProblem{}, LineItems: []SubmitProblem{}, Advice: []SubmitProblem{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("check = %+v, want ready", got)
	}
}

// Every empty fixed needed-to-submit field is listed in form order; 0.00 is a planned cost (FR-011), and
// an ongoing activity needs no end date (FR-010).
func TestCheck_EmptyFieldsAndZeroCost_FR011_FR023(t *testing.T) {
	in := readyInput()
	in.activity = Activity{ID: "A-1010", Name: "Lunar new year promo", Attributes: settings.AttributeValues{}}
	in.workflow, in.lineItems = nil, LineItemFacts{}
	got := check(in)
	var fields []string
	for _, p := range got.Activity {
		fields = append(fields, p.Field+": "+p.Message)
	}
	want := []string{
		"ownerId: Owner — needed to submit", "startDate: Start date — needed to submit", "endDate: End date — needed to submit",
		"plannedCost: Planned cost — needed to submit", "approvalWorkflowId: Approval workflow — needed to submit",
	}
	if !reflect.DeepEqual(fields, want) || got.Ready {
		t.Errorf("problems = %q, want %q", fields, want)
	}

	in = readyInput()
	in.activity.PlannedCost, in.activity.EndDate, in.activity.Ongoing = usd("0.00"), nil, true
	in.lineItems = LineItemFacts{}
	if got := check(in); !got.Ready || len(got.Advice) != 0 {
		t.Errorf("0.00 planned cost, ongoing: check = %+v, want ready without advice", got)
	}
}

// Dates outside the initiative's dates (after the initiative changed) and a type the model no longer
// allows are listed, each with the field that fixes it (edge cases).
func TestCheck_DatesOutsideAndTypeNotAllowed_EdgeCase(t *testing.T) {
	in := readyInput()
	in.activity.StartDate = day("2025-12-01")
	in.typeAllowed, in.typeName = false, "MOBILITY – Long-term contract"
	got := check(in)
	want := []SubmitProblem{
		{Code: ProblemDatesOutsideInitiative, Field: "startDate", Message: "Dates are outside the initiative's dates (01 Jan 2026 – 30 Jun 2026)"},
		{Code: ProblemTypeNotAllowed, Field: "activityTypeId",
			Message: "Activity type 'MOBILITY – Long-term contract' isn't allowed by the plan's model 'Lubricants 1.0' — choose another type"},
	}
	if !reflect.DeepEqual(got.Activity, want) || got.ProblemCount != 2 {
		t.Errorf("activity problems = %+v, want %+v", got.Activity, want)
	}
}

// A chosen workflow that is no longer active is listed (FR-023, D-109).
func TestCheck_InactiveWorkflow_FR023(t *testing.T) {
	in := readyInput()
	in.workflow.Active = false
	got := check(in)
	want := []SubmitProblem{{Code: ProblemInactiveWorkflow, Field: "approvalWorkflowId",
		Message: "Approval workflow 'Lubricants EU approval' is no longer active — choose an active one"}}
	if !reflect.DeepEqual(got.Activity, want) {
		t.Errorf("activity problems = %+v, want %+v", got.Activity, want)
	}
}

// A positive unassigned planned cost is advice that never blocks; a negative one blocks (FR-024, D-204).
func TestCheck_UnassignedPlannedCost_FR024(t *testing.T) {
	in := readyInput()
	in.lineItems = LineItemFacts{Count: 1, PlannedCostPlan: usd("800.00").Amount}
	got := check(in)
	advice := []SubmitProblem{{Code: ProblemUnassignedRemaining, Field: "lineItems", Message: "200.00 USD not yet assigned to a line item"}}
	if !got.Ready || !reflect.DeepEqual(got.Advice, advice) {
		t.Errorf("positive balance: check = %+v, want ready with the advice %+v", got, advice)
	}

	in.lineItems = LineItemFacts{Count: 1, PlannedCostPlan: usd("1000.01").Amount}
	got = check(in)
	if got.Ready || got.ProblemCount != 1 || got.LineItems[0].Code != ProblemNegativeUnassigned || len(got.Advice) != 0 {
		t.Errorf("negative balance: check = %+v, want one blocking line-item problem", got)
	}
	if c := completeness(got, in.lineItems); c != (Completeness{}) {
		t.Errorf("completeness = %+v: a negative balance is no field and no line item (D-109)", c)
	}

	if balance := unassigned(Activity{}, LineItemFacts{}); balance != nil {
		t.Errorf("unassigned without planned cost and line items = %v, want none", balance)
	}
}
