package plan

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// planGoneMessage answers a plan that does not exist (any more).
const planGoneMessage = "This plan no longer exists — someone may have deleted it."

// fallbackColour is the colour of a team that no longer resolves: a neutral one of the palette.
const fallbackColour = "slate"

// Service holds the rules of plans (feature 004): every check, calculation and change of a plan.
type Service struct {
	tx        store.Transactor
	repo      Store
	ids       *ids.Generator
	history   *history.Service
	settings  Settings
	directory Directory
	now       clock.Clock
	levels    []Level // the levels below the plan, top-down (RegisterLevel)

	allocations Allocations // 014's global-budget allocations (UseAllocations); nil: no plan is allocated
}

// NewService returns the plans service. It keeps plans in repo and reads Settings and people and teams
// through settings and directory. It knows no level below the plan until RegisterLevel adds one.
func NewService(tools history.Tools, repo Store, settings Settings, directory Directory) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, settings: settings, directory: directory, now: tools.Now,
	}
}

// Defaults returns what a new plan starts with for a (FR-006): the current year and the years a plan may
// have, a's active teams by name, a as owner when they have one (R1), the only active model, No for AOP
// and BCPIR, Evenly and INQ.
func (s *Service) Defaults(ctx context.Context, a actor.Actor) (Defaults, error) {
	first, last := s.years()
	defaults := Defaults{
		CurrentYear: s.now().Year(), Year: s.now().Year(), Years: []int{}, TeamIDs: []string{},
		DefaultSpread: SpreadEvenly, GoalStage: StageINQ, Currency: money.BaseCurrency, PlanType: TypeOperational,
	}
	for year := first; year <= last; year++ {
		defaults.Years = append(defaults.Years, year)
	}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		if defaults.TeamIDs, err = s.activeTeamIDsOf(ctx, a); err != nil {
			return err
		}
		if len(defaults.TeamIDs) > 0 {
			defaults.OwnerID = &a.ID
		}
		defaults.ModelID, err = s.onlyActiveModel(ctx)
		return err
	})
	if err != nil {
		return Defaults{}, err
	}
	return defaults, nil
}

func (s *Service) activeTeamIDsOf(ctx context.Context, a actor.Actor) ([]string, error) {
	teamIDs, err := s.directory.TeamIDsOf(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	teams, err := s.directory.TeamRefs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	active := []people.TeamRef{}
	for _, id := range teamIDs {
		if team, found := teams[id]; found && team.Active {
			active = append(active, team)
		}
	}
	activeIDs := []string{}
	for _, team := range sortedTeams(active) {
		activeIDs = append(activeIDs, team.ID)
	}
	return activeIDs, nil
}

// onlyActiveModel returns nil when there are no active models, or several to choose from.
func (s *Service) onlyActiveModel(ctx context.Context) (*string, error) {
	models, err := s.settings.Models(ctx)
	if err != nil {
		return nil, err
	}
	var activeModels []string
	for _, m := range models {
		if m.Active {
			activeModels = append(activeModels, m.ID)
		}
	}
	if len(activeModels) != 1 {
		return nil, nil
	}
	return &activeModels[0], nil
}
