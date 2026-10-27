package activity

import (
	"context"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

const noun = "activity"

const (
	maxNameLength        = 120
	maxDescriptionLength = 2000
)

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
	workflowRecallMessage   = "Recall the activity to choose another workflow."
	ongoingLockedMessage    = "Ongoing can't be undone — to plan with an end date, create a new activity."
	costNegativeMessage     = "Planned cost can't be negative."
	costCurrencyMessage     = "Enter the amount in %s, the plan's currency."
	revenueCurrencyMessage  = "Enter the revenue in USD."
	enterAmountMessage      = "Enter an amount, such as 12,500.00."
	restartApprovalMessage  = "This activity will need approval again — confirm the change to save it."
	reloadMessage           = "Reload the page and try again."
	approvalRestartedFormat = "Approval restarted because %s changed."
	restartVerb             = "restarted approval of"
)

type Service struct {
	tx        Transactor
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
	// displayRates reads the reader's display currency; nil shows USD.
	displayRates DisplayRates
}

// NewService gives activities no line items and no approval hooks until UseLineItems and UseApprovalHooks.
func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now,
		plans: deps.Plans, hierarchy: deps.Hierarchy, settings: deps.Settings, people: deps.People, workflows: deps.Workflows,
		lineItems: noLineItems{}, hooks: noApprovalHooks{},
	}
}

func (s *Service) UseLineItems(li LineItems) {
	s.lineItems = li
}

func (s *Service) UseApprovalHooks(h ApprovalHooks) {
	s.hooks = h
}

// checkAttributes keeps the values of attributes the plan's business unit hides when the save leaves them out.
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

type amountRule struct {
	currency        string
	negativeMessage string
	currencyMessage string
}

// parseAmountField adds a problem under field with the first message that applies, and then returns nil.
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

// parseUSD reads a goal amount: goals stay in USD whatever the plan's currency.
func parseUSD(problems *errs.Problems, field string, in money.Input, negativeMessage string) *money.Money {
	return parseAmountField(problems, field, in,
		amountRule{currency: money.BaseCurrency, negativeMessage: negativeMessage, currencyMessage: revenueCurrencyMessage})
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
