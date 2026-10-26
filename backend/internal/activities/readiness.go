package activities

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/campaigns"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Messages of the plan's "Ready to submit panel" row.
const (
	neededToSubmitFormat     = "%s — needed to submit"
	splitIncompleteFormat    = "%s shares add up to %s %% — they must add up to 100.00 %%"
	datesOutsideFormat       = "Dates are outside the initiative's dates (%s)"
	typeNotAllowedFormat     = "Activity type '%s' isn't allowed by the plan's model '%s' — choose another type"
	inactiveWorkflowFormat   = "Approval workflow '%s' is no longer active — choose an active one"
	negativeUnassignedFormat = "Line items exceed the activity's planned cost by %s"
	unassignedAdviceFormat   = "%s not yet assigned to a line item"
	lineItemProblemFormat    = "Line item '%s' (%s): %s"
)

// checkInput is everything the ready-to-submit check looks at, read by the service.
type checkInput struct {
	activity    Activity
	initiative  campaigns.Initiative
	missing     []settings.MissingField // the activity's shown needed attributes still empty, or splits not at 100.00 %
	typeName    string
	typeAllowed bool // by the plan's model (R7)
	modelName   string
	workflow    *WorkflowFacts // the chosen workflow; nil when none is chosen or it no longer exists
	lineItems   LineItemFacts
}

// check lists everything that stops the activity from being submitted, all at once, in the order of data
// model §4.1 (FR-023, FR-024): this activity's empty needed fields, unfinished splits, dates outside the
// initiative's, a type the model no longer allows and an inactive workflow; then a negative unassigned
// planned cost and the line items' problems; a positive unassigned planned cost is advice only (D-204).
func check(in checkInput) SubmitCheck {
	a := in.activity
	result := SubmitCheck{Activity: []SubmitProblem{}, LineItems: []SubmitProblem{}, Advice: []SubmitProblem{}}
	addActivity := func(code, field, message string) {
		result.Activity = append(result.Activity, SubmitProblem{Code: code, Field: field, Message: message})
	}

	fixed := []struct {
		field, label string
		empty        bool
	}{
		{"ownerId", "Owner", a.OwnerID == ""},
		{"startDate", "Start date", a.StartDate == nil},
		{"endDate", "End date", a.EndDate == nil && !a.Ongoing},
		{"plannedCost", "Planned cost", a.PlannedCost == nil}, // 0.00 is a value (FR-011)
		{"approvalWorkflowId", "Approval workflow", a.ApprovalWorkflowID == "" || in.workflow == nil},
	}
	for _, f := range fixed {
		if f.empty {
			addActivity(ProblemMissingField, f.field, fmt.Sprintf(neededToSubmitFormat, f.label))
		}
	}
	for _, m := range in.missing {
		if m.Reason == "" {
			addActivity(ProblemMissingField, m.Field, fmt.Sprintf(neededToSubmitFormat, m.Label))
			continue
		}
		total := splitTotal(a.Attributes[strings.TrimPrefix(m.Field, "attributes.")])
		addActivity(ProblemSplitIncomplete, m.Field, fmt.Sprintf(splitIncompleteFormat, m.Label, total.String()))
	}
	if !datesFit(a, in.initiative) {
		addActivity(ProblemDatesOutsideInitiative, "startDate", fmt.Sprintf(datesOutsideFormat,
			date.FormatRangeFull(*in.initiative.StartDate, *in.initiative.EndDate)))
	}
	if !in.typeAllowed {
		addActivity(ProblemTypeNotAllowed, "activityTypeId", fmt.Sprintf(typeNotAllowedFormat, in.typeName, in.modelName))
	}
	if in.workflow != nil && !in.workflow.Active {
		addActivity(ProblemInactiveWorkflow, "approvalWorkflowId", fmt.Sprintf(inactiveWorkflowFormat, in.workflow.Name))
	}

	balance := unassigned(a, in.lineItems)
	if balance != nil && balance.Amount.Sign() < 0 {
		result.LineItems = append(result.LineItems, SubmitProblem{Code: ProblemNegativeUnassigned, Field: "lineItems",
			Message: fmt.Sprintf(negativeUnassignedFormat, money.Money{Amount: balance.Amount.Neg(), Currency: balance.Currency}.Format())})
	}
	for _, p := range in.lineItems.Problems {
		id, name := p.LineItemID, p.LineItemName
		result.LineItems = append(result.LineItems, SubmitProblem{Code: p.Code, Field: p.Field, LineItemID: &id,
			LineItemName: &name, Message: fmt.Sprintf(lineItemProblemFormat, p.LineItemName, p.LineItemID, p.Message)})
	}
	if balance != nil && balance.Amount.Sign() > 0 {
		result.Advice = append(result.Advice, SubmitProblem{Code: ProblemUnassignedRemaining, Field: "lineItems",
			Message: fmt.Sprintf(unassignedAdviceFormat, balance.Format())})
	}

	result.ProblemCount = len(result.Activity) + len(result.LineItems)
	result.Ready = result.ProblemCount == 0
	return result
}

// completeness counts the check for the completeness chip (FR-022, D-109): the items of "This activity",
// and the distinct line items with a problem.
func completeness(c SubmitCheck, li LineItemFacts) Completeness {
	lineItems := map[string]bool{}
	for _, p := range li.Problems {
		lineItems[p.LineItemID] = true
	}
	return Completeness{MissingFields: len(c.Activity), LineItemsNeedingAttention: len(lineItems)}
}

// missingFields returns the "This activity" items of c as fields still needed, for the completeness of the
// plan above (004 itemsBelow): an empty field with its label; an unfinished split as Settings reports it
// (attributes are the check's attribute items); dates, type or workflow with the problem as the reason.
func missingFields(c SubmitCheck, attributes []settings.MissingField) []errs.MissingField {
	fields := make([]errs.MissingField, 0, len(c.Activity))
	for _, p := range c.Activity {
		if i := slices.IndexFunc(attributes, func(m settings.MissingField) bool { return m.Field == p.Field }); i >= 0 {
			fields = append(fields, attributes[i])
			continue
		}
		field := errs.MissingField{Field: p.Field, Label: labels[p.Field]}
		if p.Code != ProblemMissingField {
			field.Reason = p.Message
		}
		fields = append(fields, field)
	}
	return fields
}

// unassigned is the activity's unassigned planned cost (domain model §4.3, 016 FR-015): its planned cost
// (0.00 when empty) minus its line items' planned costs in the plan currency; nil when there is neither a
// planned cost nor a line item.
func unassigned(a Activity, li LineItemFacts) *money.Money {
	if a.PlannedCost == nil && li.Count == 0 {
		return nil
	}
	var planned money.Amount
	currency := cmp.Or(li.Currency, money.BaseCurrency)
	if a.PlannedCost != nil {
		planned, currency = a.PlannedCost.Amount, a.PlannedCost.Currency
	}
	balance := money.Money{Amount: planned.Sub(li.PlannedCostPlan), Currency: currency}
	return &balance
}
