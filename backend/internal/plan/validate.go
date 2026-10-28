package plan

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

const (
	maxNameLength          = 120
	maxDescriptionLength   = 2000
	maxGoalWholeDigits     = 15 // a goal of INQ or MQL has at most 15 digits, like an amount
	firstYearBeforeCurrent = 0  // a new plan's year: the current year, never a past one …
	lastYearAfterCurrent   = 10 // … to the current year + 10 (D-326)
)

const (
	msgEnterName          = "Enter a plan name."
	msgNameTooLong        = "Use 120 characters or fewer."
	msgNameTaken          = "There is already a plan called '%s'. Choose another name."
	msgChooseYear         = "Choose a year from %d to %d."
	msgChooseModel        = "Choose a model."
	msgUnknownModel       = "Choose a model from the list."
	msgInactiveModel      = "'%s' is inactive — choose an active model."
	msgUnknownTeam        = "Choose teams from the list."
	msgInactiveTeam       = "'%s' is inactive — choose an active team."
	msgUnknownOwner       = "Choose an owner from the list."
	msgInactiveOwner      = "%s is inactive — choose someone else."
	msgOwnerNeedsTeams    = "Choose teams first — the owner must belong to one of the plan's teams."
	msgOwnerNotInTeams    = "%s isn't in any of the chosen teams — choose another owner or clear the field."
	msgDescriptionTooLong = "Use 2,000 characters or fewer."
	msgEnterAmount        = "Enter an amount, for example 1200000 or 1,200,000.00."
	msgTwoDecimals        = "Use at most 2 decimals."
	msgBudgetNegative     = "Plan budget can't be negative."
	msgUnknownBudget      = "Choose a finance budget from the list."
	msgBudgetOfYear       = "Choose a finance budget of %d."
	msgChooseSpread       = "Choose how line items are spread."
	msgChooseStage        = "Choose INQ, MQL or Revenue."
	msgEnterNumber        = "Enter a number, for example 250000."
	msgWholeNumber        = "Enter a whole number."
	msgGoalNegative       = "The goal can't be negative."
)

var numberText = regexp.MustCompile(`^(-?)([0-9]+)(?:\.([0-9]+))?$`)

// form is the plan form of a create or an update before its checks (D-078); Year is set only on a create.
type form struct {
	Name                   string
	Year                   *int
	ModelID                *string
	TeamIDs                []string
	OwnerID                *string
	AOP                    bool
	Description            string
	Budget                 *money.Input
	DefaultFinanceBudgetID *string
	DefaultSpread          string
	BCPIR                  bool
	GoalStage              string
	GoalAmount             *string
	Attributes             settings.AttributeValues
	Currency               *string
	PlanType               *string
	Overspend              *string
}

func formOf(in PlanInput) form {
	return form{
		Name: in.Name, Year: in.Year, ModelID: in.ModelID, TeamIDs: in.TeamIDs, OwnerID: in.OwnerID, AOP: in.AOP,
		Description: in.Description, Budget: in.Budget, DefaultFinanceBudgetID: in.DefaultFinanceBudgetID,
		DefaultSpread: in.DefaultSpread, BCPIR: in.BCPIR, GoalStage: in.GoalStage, GoalAmount: in.GoalAmount,
		Attributes: in.Attributes, Currency: in.Currency, PlanType: in.PlanType, Overspend: in.Overspend,
	}
}

func formOfUpdate(in PlanUpdate) form {
	return form{
		Name: in.Name, ModelID: in.ModelID, TeamIDs: in.TeamIDs, OwnerID: in.OwnerID, AOP: in.AOP,
		Description: in.Description, Budget: in.Budget, DefaultFinanceBudgetID: in.DefaultFinanceBudgetID,
		DefaultSpread: in.DefaultSpread, BCPIR: in.BCPIR, GoalStage: in.GoalStage, GoalAmount: in.GoalAmount,
		Attributes: in.Attributes, Currency: in.Currency, PlanType: in.PlanType, Overspend: in.Overspend,
	}
}

// check returns the plan to store: the form's values on stored, or a new plan when stored is nil. Every problem goes
// to problems so all are reported at once; a value the plan already has may stay although it became inactive.
func (s *Service) check(ctx context.Context, f form, stored *Plan, problems *errs.Problems) (Plan, error) {
	var p Plan
	if stored != nil {
		p = *stored // keeps the ID, the year, the status and the bookkeeping
	}
	p.Name = strings.TrimSpace(f.Name)
	if err := s.checkName(ctx, p.Name, p.ID, problems); err != nil {
		return Plan{}, err
	}
	if stored == nil {
		p.Year = s.checkYear(f.Year, problems)
		p.BudgetPhasing = rollup.NotPhased
	}
	p.PlanType = checkPlanType(f.PlanType, stored, problems)
	var err error
	if p.ModelID, err = s.checkModel(ctx, text(f.ModelID), stored, problems); err != nil {
		return Plan{}, err
	}
	if p.TeamIDs, err = s.checkTeams(ctx, f.TeamIDs, stored, problems); err != nil {
		return Plan{}, err
	}
	p.OwnerID = text(f.OwnerID)
	if err := s.checkOwner(ctx, p.OwnerID, p.TeamIDs, stored, problems); err != nil {
		return Plan{}, err
	}
	p.AOP, p.BCPIR = f.AOP, f.BCPIR
	p.Description = strings.TrimSpace(f.Description)
	if utf8.RuneCountInString(p.Description) > maxDescriptionLength {
		problems.Add("description", msgDescriptionTooLong)
	}
	if p.Currency, err = s.checkCurrency(ctx, f.Currency, stored, problems); err != nil {
		return Plan{}, err
	}
	p.Budget = checkBudget(f.Budget, p.Currency, problems)
	if p.Budget == nil { // a removed budget removes its phasing
		p.BudgetPhasing = rollup.NotPhased
	}
	p.DefaultFinanceBudgetID = text(f.DefaultFinanceBudgetID)
	if err := s.checkFinanceBudget(ctx, p.DefaultFinanceBudgetID, p.Year, problems); err != nil {
		return Plan{}, err
	}
	p.DefaultSpread = Spread(cmp.Or(f.DefaultSpread, string(SpreadEvenly)))
	if !slices.Contains([]Spread{SpreadEvenly, SpreadFirstMonth, SpreadLastMonth, SpreadCustom}, p.DefaultSpread) {
		problems.Add("defaultSpread", msgChooseSpread)
	}
	if p.Attributes, err = s.checkAttributes(ctx, f.Attributes, stored, problems); err != nil {
		return Plan{}, err
	}
	p.GoalStage = GoalStage(cmp.Or(f.GoalStage, string(StageINQ)))
	if !slices.Contains([]GoalStage{StageINQ, StageMQL, StageRevenue}, p.GoalStage) {
		problems.Add("goalStage", msgChooseStage)
	}
	amount, message := checkGoalAmount(text(f.GoalAmount), p.GoalStage)
	if message != "" {
		problems.Add("goalAmount", message)
	}
	p.GoalAmount = amount
	if p.Overspend, err = s.checkOverspend(ctx, f.Overspend, stored, problems); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *Service) checkTeams(ctx context.Context, sent []string, stored *Plan, problems *errs.Problems) ([]string, error) {
	teamIDs := []string{}
	for _, id := range sent {
		if id = strings.TrimSpace(id); id != "" && !slices.Contains(teamIDs, id) {
			teamIDs = append(teamIDs, id)
		}
	}
	refs, err := s.directory.TeamRefs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	unknown := false
	for _, id := range teamIDs {
		ref, found := refs[id]
		switch {
		case !found:
			unknown = true
		case !ref.Active && (stored == nil || !slices.Contains(stored.TeamIDs, id)):
			problems.Add("teamIds", fmt.Sprintf(msgInactiveTeam, ref.Name))
		}
	}
	if unknown {
		problems.Add("teamIds", msgUnknownTeam)
	}
	slices.SortFunc(teamIDs, func(x, y string) int {
		return cmp.Or(strings.Compare(strings.ToLower(refs[x].Name), strings.ToLower(refs[y].Name)), strings.Compare(x, y))
	})
	return teamIDs, nil
}

// checkOwner requires the owner to be in one of the plan's teams on every save, so a plan whose owner left its
// teams is refused until the owner is changed or cleared (D-085).
func (s *Service) checkOwner(ctx context.Context, ownerID string, teamIDs []string, stored *Plan, problems *errs.Problems) error {
	if ownerID == "" {
		return nil
	}
	refs, err := s.directory.PersonRefs(ctx, []string{ownerID})
	if err != nil {
		return err
	}
	owner, found := refs[ownerID]
	switch {
	case !found:
		problems.Add("ownerId", msgUnknownOwner)
	case !owner.Active && (stored == nil || stored.OwnerID != ownerID):
		problems.Add("ownerId", fmt.Sprintf(msgInactiveOwner, owner.Name))
	case len(teamIDs) == 0:
		problems.Add("ownerId", msgOwnerNeedsTeams)
	default:
		members, err := s.directory.MemberIDs(ctx, teamIDs)
		if err != nil {
			return err
		}
		if !ownerEligible(ownerID, teamIDs, members) {
			problems.Add("ownerId", fmt.Sprintf(msgOwnerNotInTeams, owner.Name))
		}
	}
	return nil
}

// checkGoalAmount returns the trimmed goal as stored, with the problem's message or "" when it is fine.
func checkGoalAmount(amount string, stage GoalStage) (stored, message string) {
	if amount == "" {
		return "", ""
	}
	parts := numberText.FindStringSubmatch(amount)
	if parts == nil {
		return "", msgEnterNumber
	}
	negative, whole, fraction := parts[1] == "-", strings.TrimLeft(parts[2], "0"), strings.TrimRight(parts[3], "0")
	if negative && (whole != "" || fraction != "") {
		return "", msgGoalNegative
	}
	if stage == StageRevenue {
		revenue, err := money.ParseAmount(strings.TrimPrefix(amount, "-"))
		switch {
		case errors.Is(err, money.ErrTooManyDecimals):
			return "", msgTwoDecimals
		case err != nil:
			return "", msgEnterNumber
		}
		return revenue.String(), ""
	}
	if fraction != "" {
		return "", msgWholeNumber
	}
	if len(whole) > maxGoalWholeDigits {
		return "", msgEnterNumber
	}
	return cmp.Or(whole, "0"), ""
}
