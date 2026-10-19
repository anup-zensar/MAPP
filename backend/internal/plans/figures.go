package plans

import (
	"slices"

	"northstar/internal/people"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// converter turns USD amounts into the reader's display currency (D-079): usd × rate, half up to the cent.
type converter struct {
	currency string
	rate     money.Rate
}

// usdConverter shows amounts in US dollars.
var usdConverter = converter{currency: money.BaseCurrency, rate: money.One}

// converterOf returns the converter of a person's display currency; an unusable rate falls back to USD.
func converterOf(d people.DisplayRate) converter {
	rate, err := money.ParseRate(d.Rate)
	if err != nil || d.Code == "" {
		return usdConverter
	}
	return converter{currency: d.Code, rate: rate}
}

// of converts usd.
func (c converter) of(usd money.Amount) money.Money {
	return money.Money{Amount: money.FromUSD(usd, c.rate), Currency: c.currency}
}

// optional converts usd, or gives nil when there is no amount.
func (c converter) optional(usd *money.Amount) *money.Money {
	if usd == nil {
		return nil
	}
	converted := c.of(*usd)
	return &converted
}

// figuresOf works out the plan's money and progress from its budget and what the levels below report
// (FR-012, domain model §4.1–4.2, D-015, D-026, D-088). Every sum is worked out in USD and converted at
// the end.
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
