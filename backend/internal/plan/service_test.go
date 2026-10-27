package plan

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// A plan name is saved without its surrounding spaces, has at most 120 characters (not bytes) and is
// unique ignoring letter case and surrounding spaces, also when a second person saves it (FR-010, edge case).
func TestCreate_NameRules_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if saved := f.create(t, priya, input("  Ideas plan  ")); saved.Name != "Ideas plan" {
		t.Errorf("name = %q, want it trimmed", saved.Name)
	}
	longest := strings.Repeat("é", 120)
	f.create(t, priya, input(longest))
	tests := []struct{ name, input, message string }{
		{"empty", "", "Enter a plan name."},
		{"spaces only", "   ", "Enter a plan name."},
		{"121 characters", strings.Repeat("a", 121), "Use 120 characters or fewer."},
		{"taken in other letters, with spaces", " IDEAS PLAN ", "There is already a plan called 'Ideas plan'. Choose another name."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.Create(t.Context(), daniel, input(tt.input))

			assertError(t, err, problems("name", tt.message))
		})
	}
}

// Description at most 2,000 characters; budget in USD with at most two decimals and not negative; a goal
// not negative, whole for INQ and MQL and with at most two decimals for revenue. Valid values are stored in
// their exact form (FR-010).
func TestCreate_DescriptionBudgetGoalChecks_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tests := []struct {
		name    string
		change  func(*PlanInput)
		problem *errs.Error
	}{
		{"description of 2,001 characters", func(in *PlanInput) { in.Description = strings.Repeat("d", 2001) },
			problems("description", "Use 2,000 characters or fewer.")},
		{"budget in EUR", func(in *PlanInput) { in.Budget = &money.Input{Amount: "100", Currency: "EUR"} },
			problems("budget", "Enter the amount in USD, the plan's currency.")},
		{"budget not a number", func(in *PlanInput) { in.Budget = &money.Input{Amount: "lots", Currency: "USD"} },
			problems("budget", "Enter an amount, for example 1200000 or 1,200,000.00.")},
		{"budget with 3 decimals", func(in *PlanInput) { in.Budget = &money.Input{Amount: "1.234", Currency: "USD"} },
			problems("budget", "Use at most 2 decimals.")},
		{"budget negative", func(in *PlanInput) { in.Budget = &money.Input{Amount: "-1", Currency: "USD"} },
			problems("budget", "Plan budget can't be negative.")},
		{"goal not a number", func(in *PlanInput) { in.GoalAmount = ptr("many") },
			problems("goalAmount", "Enter a number, for example 250000.")},
		{"goal negative", func(in *PlanInput) { in.GoalAmount = ptr("-5") },
			problems("goalAmount", "The goal can't be negative.")},
		{"INQ goal with decimals", func(in *PlanInput) { in.GoalAmount = ptr("2.5") },
			problems("goalAmount", "Enter a whole number.")},
		{"revenue goal with 3 decimals", func(in *PlanInput) { in.GoalStage, in.GoalAmount = "revenue", ptr("1.234") },
			problems("goalAmount", "Use at most 2 decimals.")},
		{"unknown spread and stage", func(in *PlanInput) { in.DefaultSpread, in.GoalStage = "weekly", "deals" },
			problems("defaultSpread", "Choose how line items are spread.", "goalStage", "Choose INQ, MQL or Revenue.")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := input("Checked plan")
			tt.change(&in)

			_, err := f.service.Create(t.Context(), priya, in)

			assertError(t, err, tt.problem)
		})
	}

	valid := input("Valid plan")
	valid.Description = "  " + strings.Repeat("d", 2000) + " "
	valid.Budget = &money.Input{Amount: "1200000", Currency: "USD"}
	valid.GoalStage, valid.GoalAmount = "revenue", ptr("2000000")
	stored := f.stored(t, f.create(t, priya, valid).ID)
	if len(stored.Description) != 2000 || stored.Budget == nil || stored.Budget.String() != "1200000.00" || stored.GoalAmount != "2000000.00" {
		t.Errorf("stored description of %d, budget %v, goal %q", len(stored.Description), stored.Budget, stored.GoalAmount)
	}
	whole := input("Whole goal")
	whole.GoalAmount = ptr(" 000250000 ")
	if stored := f.stored(t, f.create(t, priya, whole).ID); stored.GoalAmount != "250000" {
		t.Errorf("stored INQ goal %q, want 250000", stored.GoalAmount)
	}
}

// Only active models may be chosen, and only a finance budget of the plan's year (FR-007).
func TestCreate_ActiveModelAndYearFinanceBudget_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tests := []struct {
		name    string
		change  func(*PlanInput)
		problem *errs.Error
	}{
		{"inactive model", func(in *PlanInput) { in.ModelID = ptr(oldModel) },
			problems("modelId", "'Old model' is inactive — choose an active model.")},
		{"unknown model", func(in *PlanInput) { in.ModelID = ptr("MOD-9") }, problems("modelId", "Choose a model from the list.")},
		{"finance budget of another year", func(in *PlanInput) { in.DefaultFinanceBudgetID = ptr(budget2027) },
			problems("defaultFinanceBudgetId", "Choose a finance budget of 2026.")},
		{"unknown finance budget", func(in *PlanInput) { in.DefaultFinanceBudgetID = ptr("FB-9") },
			problems("defaultFinanceBudgetId", "Choose a finance budget from the list.")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := input("Checked plan")
			tt.change(&in)

			_, err := f.service.Create(t.Context(), priya, in)

			assertError(t, err, tt.problem)
		})
	}

	in := input("With budget")
	in.DefaultFinanceBudgetID = ptr(budget2026)
	view := f.create(t, priya, in)
	if !reflect.DeepEqual(view.DefaultFinanceBudget, &FinanceBudgetRef{ID: budget2026, Year: 2026, Name: "2026 Lubricants Budget"}) {
		t.Errorf("default finance budget = %+v", view.DefaultFinanceBudget)
	}
}

// The owner must be a member of one of the chosen teams (R1); unknown and inactive teams and owners are
// refused; teams are stored once each, sorted by name (FR-021, FR-007).
func TestCreate_OwnerMustBeInChosenTeam_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tests := []struct {
		name    string
		change  func(*PlanInput)
		problem *errs.Error
	}{
		{"owner in none of the teams", func(in *PlanInput) { in.OwnerID = ptr("U-1004") },
			problems("ownerId", "Mei Lin isn't in any of the chosen teams — choose another owner or clear the field.")},
		{"unknown owner", func(in *PlanInput) { in.OwnerID = ptr("U-9999") }, problems("ownerId", "Choose an owner from the list.")},
		{"inactive owner", func(in *PlanInput) { in.OwnerID = ptr("U-1009") },
			problems("ownerId", "Old Person is inactive — choose someone else.")},
		{"inactive team", func(in *PlanInput) { in.TeamIDs = []string{teamClosed, teamLubricants} },
			problems("teamIds", "'Closed team' is inactive — choose an active team.")},
		{"unknown team", func(in *PlanInput) { in.TeamIDs = []string{"T-9999", teamLubricants} },
			problems("teamIds", "Choose teams from the list.")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := input("Owned plan")
			tt.change(&in)

			_, err := f.service.Create(t.Context(), priya, in)

			assertError(t, err, tt.problem)
		})
	}

	in := input("Two teams")
	in.TeamIDs = []string{teamMobility, teamLubricants, teamMobility}
	in.OwnerID = ptr("U-1006") // Sara is in both
	view := f.create(t, priya, in)
	if stored := f.stored(t, view.ID); !slices.Equal(stored.TeamIDs, []string{teamLubricants, teamMobility}) || view.OwnerProblem != "" {
		t.Errorf("teams = %q, owner problem %q; want each team once, by name, and no problem", stored.TeamIDs, view.OwnerProblem)
	}
}

// attributesOf returns plan values with the business unit options.
func attributesOf(units ...string) settings.AttributeValues {
	encoded, _ := json.Marshal(units)
	return settings.AttributeValues{businessUnit: encoded}
}

// User story 2 — Find and compare plans.

// seedPlan is a stored plan for the list tests, with a budget in USD ("" for none).
func seedPlan(t *testing.T, id, name string, year int, team, owner, budget string, status Status) Plan {
	t.Helper()
	p := Plan{
		ID: id, Name: name, Year: year, ModelID: lubricantsModel, TeamIDs: []string{team}, OwnerID: owner,
		DefaultSpread: SpreadEvenly, GoalStage: StageINQ, Attributes: attributesOf(lubricants), Status: status,
		Meta: record.NewMeta(priya, start),
	}
	if budget != "" {
		amount := usd(t, budget)
		p.Budget = &amount
	}
	return p
}

// put stores plans as they are, in one Write.
func (f *fixture) put(t *testing.T, plans ...Plan) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		for _, p := range plans {
			if err := f.service.repo.Put(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
}

// actorOf returns a person as an actor.
func actorOf(id, name string) actor.Actor {
	return actor.Actor{ID: id, Name: name}
}
