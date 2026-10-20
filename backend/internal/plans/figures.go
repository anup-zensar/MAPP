package plans

import (
	"slices"

	"northstar/internal/people"
	"northstar/internal/platform/money"
	"northstar/internal/rollup"
	"northstar/internal/settings"
)

// converter turns amounts of one currency into the currency it answers in (D-079, 016 D-261): Convert(amount,
// from, to), half up to the cent, once. The display converter of a person reads USD amounts (from = 1);
// reading converts a plan currency's amounts; identity keeps a plan currency's amounts as they are.
type converter struct {
	currency string     // the currency of the answers
	from, to money.Rate // the rates per 1 USD of the amounts read and of currency
}

// usdConverter shows USD amounts in US dollars.
var usdConverter = converter{currency: money.BaseCurrency, from: money.One, to: money.One}

// converterOf returns the converter of a person's display currency, for USD amounts; an unusable rate falls
// back to USD.
func converterOf(d people.DisplayRate) converter {
	rate, err := money.ParseRate(d.Rate)
	if err != nil || d.Code == "" {
		return usdConverter
	}
	return converter{currency: d.Code, from: money.One, to: rate}
}

// identity answers amounts of code as they are (the plan currency's figures, 016 FR-013).
func identity(code string) converter {
	return converter{currency: code, from: money.One, to: money.One}
}

// reading returns the display converter c for amounts in the currency code with the rates (016 FR-013): the
// same amount when code is the display currency; an amount whose currency has no rate stays in its currency.
func (c converter) reading(code string, rates money.Rates) converter {
	if code == c.currency {
		return identity(code)
	}
	from, err := rates.Of(code)
	if err != nil {
		return identity(code)
	}
	return converter{currency: c.currency, from: from, to: c.to}
}

// of converts an amount.
func (c converter) of(amount money.Amount) money.Money {
	return money.Money{Amount: money.Convert(amount, c.from, c.to), Currency: c.currency}
}

// optional converts an amount, or gives nil when there is none.
func (c converter) optional(amount *money.Amount) *money.Money {
	if amount == nil {
		return nil
	}
	converted := c.of(*amount)
	return &converted
}

// figuresOf works out the plan's money and progress from its budget and what the levels below report
// (FR-012, domain model §4.1–4.2, D-015, D-026, D-088). Every sum is worked out in the plan currency and
// converted at the end with c: the display converter reading the plan currency, or identity (016 FR-013).
//
//	budget 1,200,000.00, campaign budgets 1,300,000.00 → remaining −100,000.00, over-allocated by 100,000.00
//	budget 1,200,000.00, planned cost 960,000.00       → budget used 80 %
func figuresOf(p Plan, f Facts, c converter) Figures {
	figures := Figures{
		Currency:           c.currency,
		Budget:             c.optional(p.Budget),
		PlannedCost:        c.optional(f.PlannedCost),
		CampaignBudgets:    c.of(f.CampaignBudgets),
		Activities:         f.Activities,
		ActivitiesApproved: f.ApprovedActivities,
		ApprovalProgress:   approvalProgress(f.Activities, f.ApprovedActivities),
	}
	if p.Budget == nil {
		return figures
	}
	remaining := p.Budget.Sub(f.CampaignBudgets)
	figures.RemainingBudget = c.optional(&remaining)
	if remaining.Sign() < 0 {
		over := remaining.Neg()
		figures.OverAllocatedBy = c.optional(&over)
	}
	var planned money.Amount // no planned cost counts as 0 here
	if f.PlannedCost != nil {
		planned = *f.PlannedCost
	}
	figures.OverBudget = planned.Cmp(*p.Budget) > 0
	if !p.Budget.IsZero() {
		if used := ratio(planned.Cents(), 100, p.Budget.Cents()); used != nil {
			percent := int(*used)
			figures.BudgetUsedPercent = &percent
		}
	}
	return figures
}

// approvalProgress says how far a plan's activities are approved (D-026).
func approvalProgress(activities, approved int) ApprovalProgress {
	switch {
	case activities == 0:
		return ProgressNoActivities
	case approved >= activities:
		return ProgressAllApproved
	default:
		return ProgressInProgress
	}
}

// contentsOf counts what is inside the plan, from what the levels below report.
func contentsOf(f Facts) Contents {
	return Contents{
		Campaigns: f.Campaigns, Initiatives: f.Initiatives, Activities: f.Activities,
		ApprovedActivities: f.ApprovedActivities, LineItems: f.LineItems,
	}
}

// ownerEligible reports whether the owner belongs to one of the plan's teams (R1). members holds each
// team's member ids.
func ownerEligible(ownerID string, teamIDs []string, members map[string][]string) bool {
	for _, teamID := range teamIDs {
		if slices.Contains(members[teamID], ownerID) {
			return true
		}
	}
	return false
}

// ownMissing lists the plan's own needed-to-publish fields that are empty, in form order (FR-017, D-085):
// Teams, Owner (empty or in none of the plan's teams), the shown needed Shell attributes (Business unit
// among them), Goal stage.
func ownMissing(p Plan, ownerOK bool, attributes []settings.MissingField) []settings.MissingField {
	missing := []settings.MissingField{}
	if len(p.TeamIDs) == 0 {
		missing = append(missing, settings.MissingField{Field: "teamIds", Label: labelTeams})
	}
	if p.OwnerID == "" || !ownerOK {
		missing = append(missing, settings.MissingField{Field: "ownerId", Label: labelOwner})
	}
	missing = append(missing, attributes...)
	if p.GoalStage == "" {
		missing = append(missing, settings.MissingField{Field: "goalStage", Label: labelGoalStage})
	}
	if problem := rollup.PhasingProblem(p.BudgetPhasing, p.Budget, p.Year, p.Currency); problem != nil { // 016 FR-022
		missing = append(missing, *problem)
	}
	return missing
}

// itemsBelow returns the records below the plan that are not complete, as the API shows them; never nil.
func itemsBelow(f Facts) []AttentionItem {
	items := make([]AttentionItem, 0, len(f.NeedsAttention))
	for _, item := range f.NeedsAttention {
		if item.Missing == nil {
			item.Missing = []settings.MissingField{}
		}
		items = append(items, item)
	}
	return items
}
