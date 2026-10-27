package activity

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

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

// checkFields checks the fields of a create (before nil) or an update of before, adds every problem to
// problems in the order of the form, and returns the activity as it would be saved (FR-006 – FR-018).
// Rules that hold only for new choices — an active type the model allows, an active owner, an active
// workflow, dates inside the initiative's — are checked when the value is new or changed (FR-013, edge
// case "initiative dates moved").
func (s *Service) checkFields(ctx context.Context, problems *errs.Problems, pc planContext, in Input, before *Activity) (Activity, error) {
	next := Activity{Status: StatusCreated}
	if before != nil {
		next = *before
	}
	next.InitiativeID = pc.initiative.ID
	exceptID := next.ID

	next.Name = strings.TrimSpace(in.Name)
	switch {
	case next.Name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(next.Name) > maxNameLength:
		problems.Add("name", nameTooLongMessage)
	default:
		siblings, err := s.repo.ListByInitiative(ctx, pc.initiative.ID)
		if err != nil {
			return Activity{}, err
		}
		if other, taken := namedLike(siblings, next.Name, exceptID); taken {
			problems.Add("name", fmt.Sprintf(nameTakenFormat, pc.initiative.Name, other.Name))
		}
	}
	next.ExternalName = strings.TrimSpace(in.ExternalName)
	if utf8.RuneCountInString(next.ExternalName) > maxNameLength {
		problems.Add("externalName", nameTooLongMessage)
	}

	next.ActivityTypeID = strings.TrimSpace(in.ActivityTypeID)
	if next.ActivityTypeID == "" {
		problems.Add("activityTypeId", chooseTypeMessage)
	} else if before == nil || before.ActivityTypeID != next.ActivityTypeID {
		t, found, err := s.settings.ActivityType(ctx, next.ActivityTypeID)
		if err != nil {
			return Activity{}, err
		}
		if !found || !t.Active || !pc.model.Allows(t.ID) {
			problems.Add("activityTypeId", fmt.Sprintf(typeNotAllowedMessage, pc.model.Name))
		}
	}

	next.OwnerID = strings.TrimSpace(deref(in.OwnerID))
	if next.OwnerID != "" && (before == nil || before.OwnerID != next.OwnerID) {
		person, found, err := s.people.Get(ctx, next.OwnerID)
		if err != nil {
			return Activity{}, err
		}
		if !found || !person.Active {
			problems.Add("ownerId", choosePersonMessage)
		}
	}

	next.Ongoing = in.Ongoing
	next.StartDate = parseDate(problems, "startDate", in.StartDate)
	next.EndDate = nil
	if !in.Ongoing {
		next.EndDate = parseDate(problems, "endDate", in.EndDate)
	}
	if before == nil || datesChanged(*before, next) {
		checkDates(problems, next.StartDate, next.EndDate, next.Ongoing, pc.initiative, latestEndDate(pc.plan.Year))
	}
	if before != nil && before.Ongoing && !in.Ongoing {
		problems.Add("ongoing", ongoingLockedMessage)
	}

	next.PlannedCost = nil
	if in.PlannedCost != nil {
		next.PlannedCost = parseAmountField(problems, "plannedCost", *in.PlannedCost, pc.plan.Currency, costNegativeMessage,
			fmt.Sprintf(costCurrencyMessage, pc.plan.Currency))
	}

	next.ApprovalWorkflowID = strings.TrimSpace(deref(in.ApprovalWorkflowID))
	if before == nil || before.ApprovalWorkflowID != next.ApprovalWorkflowID {
		switch {
		case before != nil && before.Status == StatusSubmitted: // an open approval round (008 FR-013)
			problems.Add("approvalWorkflowId", workflowRecallMessage)
		case before != nil && inApproval(*before):
			problems.Add("approvalWorkflowId", workflowLockedMessage)
		case next.ApprovalWorkflowID != "":
			workflow, found, err := s.workflows.GetWorkflow(ctx, next.ApprovalWorkflowID)
			if err != nil {
				return Activity{}, err
			}
			if !found || !workflow.Active {
				problems.Add("approvalWorkflowId", chooseWorkflowMessage)
			}
		}
	}

	next.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(next.Description) > maxDescriptionLength {
		problems.Add("description", descriptionTooLong)
	}

	values, err := s.checkAttributes(ctx, problems, pc, in.Attributes, before)
	if err != nil {
		return Activity{}, err
	}
	next.Attributes = values

	goals := parseGoals(problems, in.Goals)
	next.INQ, next.SetLaterStagesMyself, next.MQL, next.Revenue = goals.INQ, goals.SetLaterStagesMyself, goals.MQL, goals.Revenue
	return next, nil
}

// checkAttributes checks the Shell attribute values of a save with feature 002's rules and returns the
// values to store. Values of attributes hidden by the plan's business unit are kept even when the save
// leaves them out (FR-015, D-108).
func (s *Service) checkAttributes(ctx context.Context, problems *errs.Problems, pc planContext, values settings.AttributeValues, before *Activity) (settings.AttributeValues, error) {
	previous := settings.AttributeValues{}
	if before != nil && before.Attributes != nil {
		previous = before.Attributes
	}
	if values == nil {
		values = settings.AttributeValues{}
	}
	result, err := s.settings.CheckAttributes(ctx, settings.LevelActivity,
		settings.AttributeInput{Values: values, Previous: previous, BusinessUnit: pc.businessUnit})
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
		if kept, had := previous[attr.ID]; had && !sent && !settings.Visible(attr, pc.businessUnit) {
			stored[attr.ID] = kept
		}
	}
	return stored, nil
}

// parseAmountField reads an amount of a request in currency: at most two decimals (D-190), in currency, not
// negative. A problem is added under field with the first message that applies, and the amount is then nil.
func parseAmountField(problems *errs.Problems, field string, in money.Input, currency, negativeMessage, currencyMessage string) *money.Money {
	amount, err := money.ParseAmount(strings.TrimSpace(in.Amount))
	switch {
	case err != nil:
		problems.Add(field, enterAmountMessage)
	case in.Currency != currency:
		problems.Add(field, currencyMessage)
	case amount.Sign() < 0:
		problems.Add(field, negativeMessage)
	default:
		return &money.Money{Amount: amount, Currency: currency}
	}
	return nil
}

// parseUSD reads a USD amount that may not be negative, with the generic currency message: goals stay in USD
// (016 FR-019).
func parseUSD(problems *errs.Problems, field string, in money.Input, negativeMessage string) *money.Money {
	return parseAmountField(problems, field, in, money.BaseCurrency, negativeMessage, revenueCurrencyMessage)
}

// deref returns the text p points at, or "" for nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
