package plan

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

func (s *Service) converter(ctx context.Context, a actor.Actor) (converter, error) {
	rate, err := s.directory.DisplayRateOf(ctx, a.ID)
	if err != nil {
		return converter{}, err
	}
	return converterOf(rate), nil
}

type lookups struct {
	people  map[string]people.PersonRef
	teams   map[string]people.TeamRef
	members map[string][]string
}

func (s *Service) lookupsFor(ctx context.Context, plans []Plan) (lookups, error) {
	var ownerIDs, teamIDs []string
	for _, p := range plans {
		if p.OwnerID != "" && !slices.Contains(ownerIDs, p.OwnerID) {
			ownerIDs = append(ownerIDs, p.OwnerID)
		}
		for _, id := range p.TeamIDs {
			if !slices.Contains(teamIDs, id) {
				teamIDs = append(teamIDs, id)
			}
		}
	}
	var l lookups
	var err error
	if l.people, err = s.directory.PersonRefs(ctx, ownerIDs); err != nil {
		return lookups{}, err
	}
	if l.teams, err = s.directory.TeamRefs(ctx, teamIDs); err != nil {
		return lookups{}, err
	}
	if l.members, err = s.directory.MemberIDs(ctx, teamIDs); err != nil {
		return lookups{}, err
	}
	return l, nil
}

type shown struct {
	owner        *people.PersonRef
	ownerProblem string
	teams        []people.TeamRef
	businessUnit []OptionRef
	figures      Figures
	completeness Completeness
}

func (s *Service) shownOf(ctx context.Context, p Plan, f Facts, l lookups, c converter) (shown, error) { //nolint:revive // argument-limit 5: the plan, what the levels below report, and the lookups and converter the list reads once
	sh := shown{teams: []people.TeamRef{}, figures: figuresOf(p, f, c)}
	for _, id := range p.TeamIDs {
		team, found := l.teams[id]
		if !found {
			team = people.TeamRef{ID: id, Name: id, Colour: fallbackColour}
		}
		sh.teams = append(sh.teams, team)
	}
	sh.teams = sortedTeams(sh.teams)
	ownerOK := false
	if p.OwnerID != "" {
		owner, found := l.people[p.OwnerID]
		if !found {
			owner = people.PersonRef{ID: p.OwnerID, Name: p.OwnerID}
		}
		sh.owner = &owner
		if ownerOK = ownerEligible(p.OwnerID, p.TeamIDs, l.members); !ownerOK {
			sh.ownerProblem = fmt.Sprintf(msgOwnerNotInTeams, owner.Name)
		}
	}
	unitIDs, err := s.settings.BusinessUnit(ctx, p.Attributes)
	if err != nil {
		return shown{}, err
	}
	if sh.businessUnit, err = s.optionRefs(ctx, unitIDs); err != nil {
		return shown{}, err
	}
	attributes, err := s.settings.MissingAttributes(ctx, settings.LevelPlan, p.Attributes, unitIDs)
	if err != nil {
		return shown{}, err
	}
	sh.completeness = Completeness{Missing: ownMissing(p, ownerOK, attributes), ItemsBelow: itemsBelow(f)}
	return sh, nil
}

func (s *Service) optionRefs(ctx context.Context, optionIDs []string) ([]OptionRef, error) {
	refs := make([]OptionRef, 0, len(optionIDs))
	for _, id := range optionIDs {
		o, found, err := s.settings.Option(ctx, id)
		if err != nil {
			return nil, err
		}
		if !found {
			refs = append(refs, OptionRef{ID: id, Label: id})
			continue
		}
		refs = append(refs, OptionRef{ID: o.ID, Label: o.Label, Active: o.Active})
	}
	return refs, nil
}

func sortedTeams(teams []people.TeamRef) []people.TeamRef {
	slices.SortFunc(teams, func(x, y people.TeamRef) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
	})
	return teams
}

func (s *Service) view(ctx context.Context, a actor.Actor, p Plan) (PlanView, error) { //nolint:gocyclo // 13: one view whose every read returns its error.
	facts, err := s.facts(ctx, []string{p.ID})
	if err != nil {
		return PlanView{}, err
	}
	f := *facts[p.ID]
	c, err := s.converter(ctx, a)
	if err != nil {
		return PlanView{}, err
	}
	rates, err := s.settings.Rates(ctx)
	if err != nil {
		return PlanView{}, err
	}
	l, err := s.lookupsFor(ctx, []Plan{p})
	if err != nil {
		return PlanView{}, err
	}
	sh, err := s.shownOf(ctx, p, f, l, c.reading(p.Currency, rates))
	if err != nil {
		return PlanView{}, err
	}
	m, found, err := s.settings.Model(ctx, p.ModelID)
	if err != nil {
		return PlanView{}, err
	}
	if !found {
		m = settings.Model{ID: p.ModelID, Name: p.ModelID, AverageDealSize: money.USD(money.Amount{})}
	}
	view := PlanView{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: p.Status,
		Model: ModelSummary{
			ID: m.ID, Name: m.Name, Active: m.Active, InqToMqlRate: m.InqToMqlRate, MqlToDealRate: m.MqlToDealRate,
			AverageDealSize: money.USD(m.AverageDealSize.Amount),
		},
		Teams: sh.teams, Owner: sh.owner, OwnerProblem: sh.ownerProblem, AOP: p.AOP, Description: p.Description,
		DefaultSpread: p.DefaultSpread, BCPIR: p.BCPIR, GoalStage: p.GoalStage, Attributes: p.Attributes,
		BusinessUnit: sh.businessUnit, Figures: sh.figures, Goals: goalsOf(p, m, f.ActivityGoals, c),
		Completeness: sh.completeness, Contents: contentsOf(f), Meta: p.Meta,
		Access:   authz.PlanAccessOf(ctx, AccessFacts(p)),
		Currency: p.Currency, PlanType: p.PlanType, BudgetPhasing: rollup.ViewOf(p.BudgetPhasing, p.Budget, p.Year, p.Currency),
		PlanFigures: figuresOf(p, f, identity(p.Currency)), CurrencyLock: currencyLock(f),
	}
	if view.Attributes == nil {
		view.Attributes = settings.AttributeValues{}
	}
	if p.Budget != nil {
		budget := money.Money{Amount: *p.Budget, Currency: p.Currency}
		view.Budget = &budget
	}
	if p.GoalAmount != "" {
		goal := p.GoalAmount
		view.GoalAmount = &goal
	}
	if view.DefaultFinanceBudget, err = s.financeBudgetRef(ctx, p); err != nil {
		return PlanView{}, err
	}
	if view.Allocation, err = s.allocationView(ctx, p); err != nil {
		return PlanView{}, err
	}
	view.Overspend = overspendControl(ctx, p, view.Allocation)
	return view, nil
}

func (s *Service) financeBudgetRef(ctx context.Context, p Plan) (*FinanceBudgetRef, error) {
	if p.DefaultFinanceBudgetID == "" {
		return nil, nil
	}
	b, found, err := s.settings.FinanceBudget(ctx, p.DefaultFinanceBudgetID)
	if err != nil {
		return nil, err
	}
	if !found {
		b = settings.FinanceBudget{ID: p.DefaultFinanceBudgetID, Year: p.Year, Name: p.DefaultFinanceBudgetID}
	}
	return &FinanceBudgetRef{ID: b.ID, Year: b.Year, Name: b.Name}, nil
}
