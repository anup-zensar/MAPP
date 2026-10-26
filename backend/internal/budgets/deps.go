package budgets

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// Budgets reads plans, people and exchange rates, and asks the access rules, through the small interfaces
// below: budgets declares what it needs, internal/app wires what implements it (budgets_adapter.go). Every
// method joins the caller's transaction, which ctx carries; ctx also carries the signed-in person.

// PlanInfo is what budgets needs to know about one plan (Go only).
type PlanInfo struct {
	ID          string
	Name        string
	Year        int
	Status      string // "draft" or "published"
	Version     int
	TeamIDs     []string
	OwnerID     string        // "" = none
	CreatedByID string        // who created the plan: a team-less plan is in their scope (014 FR-005)
	Currency    string        // the plan currency (016; D-303)
	Budget      *money.Amount // in the plan's currency; nil = none
}

// Plans is what budgets reads and changes of plans (feature 004).
type Plans interface {
	// Plan returns the plan with id, and false when there is none (deleted plans are not stored).
	Plan(ctx context.Context, id string) (PlanInfo, bool, error)
	// PlansOfYear returns the plans of year, any status.
	PlansOfYear(ctx context.Context, year int) ([]PlanInfo, error)
	// PlannedCosts returns the planned cost in USD of each plan; nil for a plan without planned cost.
	PlannedCosts(ctx context.Context, planIDs []string) (map[string]*money.Amount, error)
	// SetBudgetFromAllocation sets the plan's budget to amount (its currency) from the plan version the
	// person saw, logging the change with comment on the plan (409 version_conflict when out of date).
	SetBudgetFromAllocation(ctx context.Context, a actor.Actor, planID string, amount money.Amount, version int, comment string) error
	// AllocationRemoved logs on the plan that its allocation was removed, with comment.
	AllocationRemoved(ctx context.Context, a actor.Actor, planID, comment string) error
}

// Directory is what budgets reads of people and teams (feature 003). The people service implements it.
type Directory interface {
	// PersonRefs returns the people with the ids, as records show them; unknown ids are left out.
	PersonRefs(ctx context.Context, ids []string) (map[string]people.PersonRef, error)
	// TeamRefs returns the teams with the ids, as records show them; unknown ids are left out.
	TeamRefs(ctx context.Context, ids []string) (map[string]people.TeamRef, error)
}

// Rates gives the current exchange rate of every currency (feature 002): allocations in a plan's own
// currency count in USD at the current rate, on read (D-303). The settings service implements it.
type Rates interface {
	Rates(ctx context.Context) (money.Rates, error)
}

// Access is how budgets asks feature 014's access rules (platform/authz) what the signed-in person, whose
// grants ctx carries, may do (plan Table C; FR-029, FR-030, FR-033). internal/app implements it over authz.
// A refusal is an error the API answers as it is (403 forbidden, with the message of the access rules);
// Availability answers carry its message as the reason.
type Access interface {
	// ManageAll refuses a person who does not hold "Create global budgets and allocate them to plans"
	// (globalBudgets.manage) for All markets: creating, changing and deleting a global budget, which spans
	// markets (FR-029).
	ManageAll(ctx context.Context) error
	// Manage refuses a person who holds globalBudgets.manage in no scope at all (Allocate).
	Manage(ctx context.Context) error
	// ManageIn refuses a person who does not hold globalBudgets.manage in the plan's scope: allocating to
	// the plan, changing or removing its allocation (FR-030).
	ManageIn(ctx context.Context, plan PlanInfo) error
	// SeeGlobalBudgets reports whether the person holds "See global budgets" (globalBudgets.view) in any
	// scope: the plan page links to the global budget (PlanAllocation.canOpen).
	SeeGlobalBudgets(ctx context.Context) bool
	// CanViewPlan reports whether the person can see the plan (FR-006): a global budget's rows show only
	// those plans, the others are added up as Other markets (FR-033).
	CanViewPlan(ctx context.Context, plan PlanInfo) bool
	// CanManageAll reports whether the person personID holds globalBudgets.manage for All markets: who may
	// own a global budget (FR-029).
	CanManageAll(ctx context.Context, personID string) (bool, error)
}

// unrestricted is the Access of a service that was given none (Deps.Access nil): every action is allowed and
// every person may own a global budget, as authz treats a context without grants (internal calls, the seed,
// service tests; D-228).
type unrestricted struct{}

func (unrestricted) ManageAll(context.Context) error                    { return nil }
func (unrestricted) Manage(context.Context) error                       { return nil }
func (unrestricted) ManageIn(context.Context, PlanInfo) error           { return nil }
func (unrestricted) SeeGlobalBudgets(context.Context) bool              { return true }
func (unrestricted) CanViewPlan(context.Context, PlanInfo) bool         { return true }
func (unrestricted) CanManageAll(context.Context, string) (bool, error) { return true, nil }

// Deps are the services budgets reads, wired by internal/app.
type Deps struct {
	Plans     Plans
	Directory Directory
	Rates     Rates
	Access    Access // nil: unrestricted
}
