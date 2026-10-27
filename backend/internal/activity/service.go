package activity

import (
	"context"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
)

// noun names an activity in the version-conflict message: "Not saved — Priya Raman changed this activity".
const noun = "activity"

// The longest texts, in characters (data model §1).
const (
	maxNameLength        = 120
	maxDescriptionLength = 2000
)

// Messages of the plan's "User-visible messages" table.
const (
	activityGoneMessage     = "This activity no longer exists — someone may have deleted it."
	initiativeGoneMessage   = "This initiative no longer exists — someone may have deleted it."
	planGoneMessage         = "This plan no longer exists — someone may have deleted it."
	enterNameMessage        = "Enter a name."
	nameTooLongMessage      = "Use 120 characters or fewer."
	nameTakenFormat         = "'%s' already has an activity called '%s'."
	descriptionTooLong      = "Use 2,000 characters or fewer."
	chooseTypeMessage       = "Choose an activity type."
	typeNotAllowedMessage   = "Choose an activity type that the model '%s' allows."
	choosePersonMessage     = "Choose an active person."
	chooseWorkflowMessage   = "Choose an active approval workflow."
	workflowLockedMessage   = "The approval workflow can't change while the activity is submitted or approved."
	workflowRecallMessage   = "Recall the activity to choose another workflow." // while Submitted (008 FR-013)
	ongoingLockedMessage    = "Ongoing can't be undone — to plan with an end date, create a new activity."
	costNegativeMessage     = "Planned cost can't be negative."
	costCurrencyMessage     = "Enter the amount in %s, the plan's currency." // 016 FR-011
	revenueCurrencyMessage  = "Enter the revenue in USD."
	enterAmountMessage      = "Enter an amount, such as 12,500.00."
	restartApprovalMessage  = "This activity will need approval again — confirm the change to save it."
	reloadMessage           = "Reload the page and try again."
	approvalRestartedFormat = "Approval restarted because %s changed."
	restartVerb             = "restarted approval of"
)

// Service holds the rules of activities (feature 006).
type Service struct {
	tx        store.Transactor
	repo      Store
	ids       *ids.Generator
	history   *history.Service
	now       clock.Clock
	plans     PlanReader
	hierarchy HierarchyReader
	settings  SettingsReader
	people    PeopleReader
	workflows Workflows
	lineItems LineItems
	hooks     ApprovalHooks
	// displayRates reads the reader's display currency (UseDisplayRates, D-220); nil shows USD.
	displayRates DisplayRates
}

// NewService returns the activities service. It keeps activities in repo, changes them with tools and
// reads the plans, campaigns, initiatives, Settings, people and approval workflows of deps.
// Until UseLineItems and UseApprovalHooks are called, activities have no line items and nothing reacts to
// approval changes.
func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now,
		plans: deps.Plans, hierarchy: deps.Hierarchy, settings: deps.Settings, people: deps.People, workflows: deps.Workflows,
		lineItems: noLineItems{}, hooks: noApprovalHooks{},
	}
}

// UseLineItems tells the service where line items are (feature 007, through internal/app).
func (s *Service) UseLineItems(li LineItems) {
	s.lineItems = li
}

// UseApprovalHooks tells the service who reacts to approval changes (feature 008, through internal/app).
func (s *Service) UseApprovalHooks(h ApprovalHooks) {
	s.hooks = h
}

// checkAttributes checks the Shell attribute values of a save with feature 002's rules and returns the
// values to store. Values of attributes hidden by the plan's business unit are kept even when the save
// leaves them out (FR-015, D-108).
func (s *Service) checkAttributes(ctx context.Context, problems *errs.Problems, c fieldCheck) (settings.AttributeValues, error) {
	previous := c.previousAttributes()
	values := c.in.Attributes
	if values == nil {
		values = settings.AttributeValues{}
	}
	result, err := s.settings.CheckAttributes(ctx, settings.LevelActivity,
		settings.AttributeInput{Values: values, Previous: previous, BusinessUnit: c.pc.businessUnit})
	if err != nil {
		return nil, err
	}
	*problems = append(*problems, result.Problems...)
	stored := result.Values
	if stored == nil {
		stored = settings.AttributeValues{}
	}
	attrs, err := s.settings.Attributes(ctx, settings.LevelActivity)
	if err != nil {
		return nil, err
	}
	for _, attr := range attrs {
		_, sent := values[attr.ID]
		if kept, had := previous[attr.ID]; had && !sent && !settings.Visible(attr, c.pc.businessUnit) {
			stored[attr.ID] = kept
		}
	}
	return stored, nil
}

// amountRule is what an amount field takes: an amount in currency, not negative, and the message of each.
type amountRule struct {
	currency        string
	negativeMessage string
	currencyMessage string
}

// parseAmountField reads an amount of a request: at most two decimals (D-190), in the rule's currency, not
// negative. A problem is added under field with the first message that applies, and the amount is then nil.
func parseAmountField(problems *errs.Problems, field string, in money.Input, rule amountRule) *money.Money {
	amount, err := money.ParseAmount(strings.TrimSpace(in.Amount))
	switch {
	case err != nil:
		problems.Add(field, enterAmountMessage)
	case in.Currency != rule.currency:
		problems.Add(field, rule.currencyMessage)
	case amount.Sign() < 0:
		problems.Add(field, rule.negativeMessage)
	default:
		return &money.Money{Amount: amount, Currency: rule.currency}
	}
	return nil
}

// parseUSD reads a USD amount that may not be negative, with the generic currency message: goals stay in USD
// (016 FR-019).
func parseUSD(problems *errs.Problems, field string, in money.Input, negativeMessage string) *money.Money {
	return parseAmountField(problems, field, in,
		amountRule{currency: money.BaseCurrency, negativeMessage: negativeMessage, currencyMessage: revenueCurrencyMessage})
}

// deref returns the text p points at, or "" for nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
