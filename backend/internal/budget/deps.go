package budget

import (
	"context"

	"mapp/internal/overspend"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

type PlanInfo struct {
	ID          string
	Name        string
	Year        int
	Status      string
	Version     int
	TeamIDs     []string
	OwnerID     string
	CreatedByID string
	Currency    string
	Budget      *money.Amount
	Overspend   overspend.Mode
}

// PlanBudget is what an allocation sets on its plan; Overspend "" keeps the plan's setting (037 FR-004).
type PlanBudget struct {
	PlanID      string
	Amount      money.Amount
	Overspend   overspend.Mode
	PlanVersion int
	Comment     string
}

type Plans interface {
	Plan(ctx context.Context, id string) (PlanInfo, bool, error)
	PlansOfYear(ctx context.Context, year int) ([]PlanInfo, error)
	PlannedCosts(ctx context.Context, planIDs []string) (map[string]*money.Amount, error)
	SetBudgetFromAllocation(ctx context.Context, a actor.Actor, in PlanBudget) error
	AllocationRemoved(ctx context.Context, a actor.Actor, planID, comment string) error
}

type Directory interface {
	PersonRefs(ctx context.Context, ids []string) (map[string]people.PersonRef, error)
	TeamRefs(ctx context.Context, ids []string) (map[string]people.TeamRef, error)
}

type Rates interface {
	Rates(ctx context.Context) (money.Rates, error)
}

type OverspendDefaults interface {
	DefaultMode(ctx context.Context) (overspend.Mode, error)
}

// Access asks feature 014's access rules what the person may do; a refusal is a 403 the API answers as it is.
type Access interface {
	ManageAll(ctx context.Context) error
	Manage(ctx context.Context) error
	ManageIn(ctx context.Context, plan PlanInfo) error
	SeeGlobalBudgets(ctx context.Context) bool
	CanViewPlan(ctx context.Context, plan PlanInfo) bool
	CanManageAll(ctx context.Context, personID string) (bool, error)
}

// unrestricted allows everything, as authz treats a context without grants (internal calls, the seed, tests).
type unrestricted struct{}

func (unrestricted) ManageAll(context.Context) error                    { return nil }
func (unrestricted) Manage(context.Context) error                       { return nil }
func (unrestricted) ManageIn(context.Context, PlanInfo) error           { return nil }
func (unrestricted) SeeGlobalBudgets(context.Context) bool              { return true }
func (unrestricted) CanViewPlan(context.Context, PlanInfo) bool         { return true }
func (unrestricted) CanManageAll(context.Context, string) (bool, error) { return true, nil }

type Deps struct {
	Plans     Plans
	Directory Directory
	Rates     Rates
	Access    Access
	Overspend OverspendDefaults
}
