package activity

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// fieldCheck is a save whose fields checkFields checks; before is nil for a create.
type fieldCheck struct {
	pc     planContext
	in     Input
	before *Activity
}

// checkFields checks the fields of a create (c.before nil) or an update of c.before, adds every problem to
// problems in the order of the form, and returns the activity as it would be saved (FR-006 – FR-018).
// Rules that hold only for new choices — an active type the model allows, an active owner, an active
// workflow, dates inside the initiative's — are checked when the value is new or changed (FR-013, edge
// case "initiative dates moved").
func (s *Service) checkFields(ctx context.Context, problems *errs.Problems, c fieldCheck) (Activity, error) {
	next := Activity{Status: StatusCreated}
	if c.before != nil {
		next = *c.before
	}
	next.InitiativeID = c.pc.initiative.ID
	var err error
	if next.Name, err = s.checkName(ctx, problems, c, next.ID); err != nil {
		return Activity{}, err
	}
	next.ExternalName = strings.TrimSpace(c.in.ExternalName)
	if utf8.RuneCountInString(next.ExternalName) > maxNameLength {
		problems.Add("externalName", nameTooLongMessage)
	}
	if next.ActivityTypeID, err = s.checkType(ctx, problems, c); err != nil {
		return Activity{}, err
	}
	if next.OwnerID, err = s.checkOwner(ctx, problems, c); err != nil {
		return Activity{}, err
	}
	next.Ongoing = c.in.Ongoing
	next.StartDate, next.EndDate = checkSchedule(problems, c)
	next.PlannedCost = checkPlannedCost(problems, c)
	if next.ApprovalWorkflowID, err = s.checkWorkflow(ctx, problems, c); err != nil {
		return Activity{}, err
	}
	next.Description = strings.TrimSpace(c.in.Description)
	if utf8.RuneCountInString(next.Description) > maxDescriptionLength {
		problems.Add("description", descriptionTooLong)
	}
	if next.Attributes, err = s.checkAttributes(ctx, problems, c); err != nil {
		return Activity{}, err
	}
	goals := parseGoals(problems, c.in.Goals)
	next.INQ, next.SetLaterStagesMyself, next.MQL, next.Revenue = goals.INQ, goals.SetLaterStagesMyself, goals.MQL, goals.Revenue
	return next, nil
}

func (s *Service) checkName(ctx context.Context, problems *errs.Problems, c fieldCheck, exceptID string) (string, error) {
	name := strings.TrimSpace(c.in.Name)
	switch {
	case name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(name) > maxNameLength:
		problems.Add("name", nameTooLongMessage)
	default:
		siblings, err := s.repo.ListByInitiative(ctx, c.pc.initiative.ID)
		if err != nil {
			return "", err
		}
		if other, taken := namedLike(siblings, name, exceptID); taken {
			problems.Add("name", fmt.Sprintf(nameTakenFormat, c.pc.initiative.Name, other.Name))
		}
	}
	return name, nil
}

func (s *Service) checkType(ctx context.Context, problems *errs.Problems, c fieldCheck) (string, error) {
	typeID := strings.TrimSpace(c.in.ActivityTypeID)
	switch {
	case typeID == "":
		problems.Add("activityTypeId", chooseTypeMessage)
	case c.before == nil || c.before.ActivityTypeID != typeID:
		t, found, err := s.settings.ActivityType(ctx, typeID)
		if err != nil {
			return "", err
		}
		if !found || !t.Active || !c.pc.model.Allows(t.ID) {
			problems.Add("activityTypeId", fmt.Sprintf(typeNotAllowedMessage, c.pc.model.Name))
		}
	}
	return typeID, nil
}

func (s *Service) checkOwner(ctx context.Context, problems *errs.Problems, c fieldCheck) (string, error) {
	ownerID := strings.TrimSpace(deref(c.in.OwnerID))
	if ownerID == "" || (c.before != nil && c.before.OwnerID == ownerID) {
		return ownerID, nil
	}
	person, found, err := s.people.Get(ctx, ownerID)
	if err != nil {
		return "", err
	}
	if !found || !person.Active {
		problems.Add("ownerId", choosePersonMessage)
	}
	return ownerID, nil
}

// checkSchedule returns the dates of c; only new or changed dates are checked against the initiative's.
func checkSchedule(problems *errs.Problems, c fieldCheck) (start, end *date.Date) {
	dated := Activity{Ongoing: c.in.Ongoing, StartDate: parseDate(problems, "startDate", c.in.StartDate)}
	if !c.in.Ongoing {
		dated.EndDate = parseDate(problems, "endDate", c.in.EndDate)
	}
	if c.before == nil || datesChanged(*c.before, dated) {
		checkDates(problems, dated, c.pc.initiative, latestEndDate(c.pc.plan.Year))
	}
	if c.before != nil && c.before.Ongoing && !c.in.Ongoing {
		problems.Add("ongoing", ongoingLockedMessage)
	}
	return dated.StartDate, dated.EndDate
}

func checkPlannedCost(problems *errs.Problems, c fieldCheck) *money.Money {
	if c.in.PlannedCost == nil {
		return nil
	}
	return parseAmountField(problems, fieldPlannedCost, *c.in.PlannedCost, amountRule{
		currency: c.pc.plan.Currency, negativeMessage: costNegativeMessage,
		currencyMessage: fmt.Sprintf(costCurrencyMessage, c.pc.plan.Currency),
	})
}

func (s *Service) checkWorkflow(ctx context.Context, problems *errs.Problems, c fieldCheck) (string, error) {
	workflowID := strings.TrimSpace(deref(c.in.ApprovalWorkflowID))
	if c.before != nil && c.before.ApprovalWorkflowID == workflowID {
		return workflowID, nil
	}
	if lock := workflowLock(c.before); lock != "" {
		problems.Add("approvalWorkflowId", lock)
		return workflowID, nil
	}
	if workflowID == "" {
		return workflowID, nil
	}
	workflow, found, err := s.workflows.GetWorkflow(ctx, workflowID)
	if err != nil {
		return "", err
	}
	if !found || !workflow.Active {
		problems.Add("approvalWorkflowId", chooseWorkflowMessage)
	}
	return workflowID, nil
}

// workflowLock is why the workflow of before can't change, or "" when it can.
func workflowLock(before *Activity) string {
	switch {
	case before == nil:
		return ""
	case before.Status == StatusSubmitted: // an open approval round (008 FR-013)
		return workflowRecallMessage
	case inApproval(*before):
		return workflowLockedMessage
	}
	return ""
}

func (c fieldCheck) previousAttributes() settings.AttributeValues {
	if c.before == nil || c.before.Attributes == nil {
		return settings.AttributeValues{}
	}
	return c.before.Attributes
}
