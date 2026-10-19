package plans

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"northstar/internal/history"
	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/settings"
)

// User story 1 — Create a plan.

// A new plan starts with the current year (choices 2025–2031), the person's active teams, the person as
// owner when they have a team, the only active model, No, Evenly and INQ (FR-006, US1 scenario 1).
func TestDefaults_FR006(t *testing.T) {
	f := newFixture(t)

	defaults, err := f.service.Defaults(t.Context(), priya)

	want := Defaults{
		CurrentYear: 2026, Years: []int{2025, 2026, 2027, 2028, 2029, 2030, 2031}, Year: 2026,
		TeamIDs: []string{teamLubricants}, OwnerID: ptr(priya.ID), // Closed team is inactive
		DefaultSpread: SpreadEvenly, GoalStage: StageINQ, // two active models: none chosen
	}
	if err != nil || !reflect.DeepEqual(defaults, want) {
		t.Errorf("Defaults = %+v, %v\nwant       %+v", defaults, err, want)
	}

	mobilityModel := f.settings.models[mobilityModel]
	mobilityModel.Active = false
	f.settings.models[mobilityModel.ID] = mobilityModel
	alone, err := f.service.Defaults(t.Context(), nina) // Nina is in no team
	if err != nil || len(alone.TeamIDs) != 0 || alone.OwnerID != nil || !reflect.DeepEqual(alone.ModelID, ptr(lubricantsModel)) {
		t.Errorf("Defaults(Nina) = %+v, %v; want no teams, no owner and the only active model", alone, err)
	}
}

// Only name, year and model are needed to save; a plan without teams, owner or business unit is saved as
// Draft, its missing fields listed (FR-010, spec edge case "The user belongs to no team").
func TestCreate_OnlyNameYearModelRequired_FR010(t *testing.T) {
	f := newFixture(t)

	view, err := f.service.Create(t.Context(), nina, PlanInput{Name: "Bare plan", Year: ptr(2026), ModelID: ptr(lubricantsModel)})

	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	stored := f.stored(t, view.ID)
	if stored.Status != StatusDraft || stored.DefaultSpread != SpreadEvenly || stored.GoalStage != StageINQ ||
		stored.TeamIDs == nil || stored.Attributes == nil {
		t.Errorf("stored = %+v; want Draft with the defaults Evenly and INQ, empty lists", stored)
	}
	wantMissing := []settings.MissingField{
		{Field: "teamIds", Label: "Teams"}, {Field: "ownerId", Label: "Owner"},
		{Field: "attributes.SA-BU", Label: "Business unit"},
	}
	if !reflect.DeepEqual(view.Completeness.Missing, wantMissing) {
		t.Errorf("missing = %+v, want %+v", view.Completeness.Missing, wantMissing)
	}

	_, err = f.service.Create(t.Context(), nina, PlanInput{})
	assertError(t, err, problems("name", "Enter a plan name.", "year", "Choose a year from 2025 to 2031.", "modelId", "Choose a model."))
}

// Every problem of a form is reported in one attempt, also a name another plan has in other letters; and
// nothing is saved (FR-010, US1 scenario 3).
func TestCreate_AllProblemsAtOnce_FR010(t *testing.T) {
	f := newFixture(t)
	f.create(t, priya, input("FY2026 Lubricants Test"))
	empty := input(" ")
	empty.ModelID = nil
	empty.Budget = &money.Input{Amount: "-5", Currency: "USD"}
	taken := input("fy2026 LUBRICANTS test")
	taken.ModelID = ptr("")
	taken.GoalAmount = ptr("-250000")

	_, errEmpty := f.service.Create(t.Context(), priya, empty)
	_, errTaken := f.service.Create(t.Context(), priya, taken)

	assertError(t, errEmpty, problems("name", "Enter a plan name.", "modelId", "Choose a model.", "budget", "Plan budget can't be negative."))
	assertError(t, errTaken, problems(
		"name", "There is already a plan called 'FY2026 Lubricants Test'. Choose another name.",
		"modelId", "Choose a model.", "goalAmount", "The goal can't be negative."))
	if entries := f.entries(t); len(entries) != 1 {
		t.Errorf("%d log entries, want only the first plan's", len(entries))
	}
}

// A plan name is saved without its surrounding spaces, has at most 120 characters (not bytes) and is
// unique ignoring letter case and surrounding spaces (FR-010).
func TestCreate_NameRules_FR010(t *testing.T) {
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
			_, err := f.service.Create(t.Context(), priya, input(tt.input))

			assertError(t, err, problems("name", tt.message))
		})
	}
}

// Description at most 2,000 characters; budget in USD with at most two decimals and not negative; a goal
// not negative, whole for INQ and MQL and with at most two decimals for revenue. Valid values are stored in
// their exact form (FR-010).
func TestCreate_DescriptionBudgetGoalChecks_FR010(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name    string
		change  func(*PlanInput)
		problem *errs.Error
	}{
		{"description of 2,001 characters", func(in *PlanInput) { in.Description = strings.Repeat("d", 2001) },
			problems("description", "Use 2,000 characters or fewer.")},
		{"budget in EUR", func(in *PlanInput) { in.Budget = &money.Input{Amount: "100", Currency: "EUR"} },
			problems("budget", "Enter the plan budget in USD.")},
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

// A new plan's year is from the current year − 1 to + 5 (FR-006).
func TestCreate_YearRange_FR006(t *testing.T) {
	f := newFixture(t)
	for _, year := range []int{2025, 2031} {
		in := input(fmt.Sprintf("Plan of %d", year))
		in.Year = ptr(year)
		if view := f.create(t, priya, in); view.Year != year {
			t.Errorf("year = %d, want %d", view.Year, year)
		}
	}
	for _, year := range []*int{ptr(2024), ptr(2032), nil} {
		in := input("Out of range")
		in.Year = year

		_, err := f.service.Create(t.Context(), priya, in)

		assertError(t, err, problems("year", "Choose a year from 2025 to 2031."))
	}
}

// Only active models may be chosen, and only a finance budget of the plan's year (FR-007).
func TestCreate_ActiveModelAndYearFinanceBudget_FR007(t *testing.T) {
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

// Without a team the owner can't be chosen: the owner must belong to one of the plan's teams (FR-007, R1,
// US1 scenario 4).
func TestCreate_OwnerNeedsTeams_FR007(t *testing.T) {
	f := newFixture(t)
	in := input("No teams")
	in.TeamIDs = nil

	_, err := f.service.Create(t.Context(), priya, in)

	assertError(t, err, problems("ownerId", "Choose teams first — the owner must belong to one of the plan's teams."))
}

// The owner must be a member of one of the chosen teams (R1); unknown and inactive teams and owners are
// refused; teams are stored once each, sorted by name (FR-021, FR-007).
func TestCreate_OwnerMustBeInChosenTeam_FR021(t *testing.T) {
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

// The problems Settings finds in the Shell attribute values are reported with the others (FR-008).
func TestCreate_AttributeProblemsCollected_FR008(t *testing.T) {
	f := newFixture(t)
	f.settings.attributeProblems = []errs.FieldError{{Field: "attributes.SA-BU", Message: "'Marine' is inactive — choose another option."}}

	_, err := f.service.Create(t.Context(), priya, input(""))

	assertError(t, err, problems("name", "Enter a plan name.", "attributes.SA-BU", "'Marine' is inactive — choose another option."))
}

// A new plan is Draft with the next permanent ID, P-1001, its creator and time, and one "created" log entry
// by its creator (FR-010, FR-031, US1 scenario 2).
func TestCreate_DraftWithIDAndLog_FR010_FR031(t *testing.T) {
	f := newFixture(t)
	in := input("FY2026 Lubricants Test")
	in.Budget = &money.Input{Amount: "1200000.00", Currency: "USD"}
	in.GoalAmount = ptr("250000")

	view := f.create(t, priya, in)

	if view.ID != "P-1001" || view.Status != StatusDraft || view.Meta != record.NewMeta(priya, start) {
		t.Errorf("plan %s, %s, %+v; want P-1001, Draft, created by Priya now", view.ID, view.Status, view.Meta)
	}
	if !reflect.DeepEqual(view.Teams, []people.TeamRef{f.directory.teams[teamLubricants]}) ||
		!reflect.DeepEqual(view.Owner, ptr(f.directory.people[priya.ID])) || view.Model.Name != "Lubricants 1.0" ||
		!reflect.DeepEqual(view.BusinessUnit, []OptionRef{{ID: lubricants, Label: "Lubricants", Active: true}}) {
		t.Errorf("references: teams %+v, owner %+v, model %+v, business unit %+v", view.Teams, view.Owner, view.Model, view.BusinessUnit)
	}
	if len(view.Completeness.Missing) != 0 || view.Goals.Plan.MQL == nil || *view.Goals.Plan.MQL != 5000 {
		t.Errorf("completeness %+v, goals %+v; want all filled in and MQL 5,000", view.Completeness, view.Goals.Plan)
	}
	entries := f.entries(t)
	if len(entries) != 1 || entries[0].Action != history.ActionCreated || entries[0].RecordID != "P-1001" ||
		entries[0].RecordType != "plan" || entries[0].Actor != priya || entries[0].RecordName != "FY2026 Lubricants Test" {
		t.Errorf("log = %+v, want one created entry by Priya", entries)
	}
}

// Two people saving plans with the same name: the second gets the duplicate-name message (spec edge case).
// Writes run one after the other, and the name is checked inside the Write that saves.
func TestCreate_SameNameTwice_EdgeCase(t *testing.T) {
	f := newFixture(t)
	f.create(t, priya, input("Shared name"))

	_, err := f.service.Create(t.Context(), daniel, input("Shared name"))

	assertError(t, err, problems("name", "There is already a plan called 'Shared name'. Choose another name."))
}

// The plan page shows a plan by its ID; an ID that does not exist answers not found (FR-011).
func TestGet_NotFound_FR011(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, priya, input("Visible plan"))

	got, err := f.service.Get(t.Context(), daniel, created.ID)
	_, errGone := f.service.Get(t.Context(), daniel, "P-9999")

	if err != nil || !reflect.DeepEqual(got, created) {
		t.Errorf("Get = %+v, %v\nwant  %+v", got, err, created)
	}
	assertError(t, errGone, errs.NotFound("This plan no longer exists — someone may have deleted it."))
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

// putDemoPlans stores the three plans of the demo data, with planned costs below two of them.
func (f *fixture) putDemoPlans(t *testing.T) {
	t.Helper()
	f.put(t,
		seedPlan(t, "P-1001", "FY2026 Lubricants Europe", 2026, teamLubricants, priya.ID, "1200000.00", StatusPublished),
		seedPlan(t, "P-1002", "FY2026 Mobility China", 2026, teamMobility, "U-1004", "800000.00", StatusDraft),
		seedPlan(t, "P-1003", "FY2027 Global Brand", 2027, teamGlobal, "U-1001", "500000.00", StatusDraft))
	planned := func(text string) *money.Amount { amount := usd(t, text); return &amount }
	f.addLevel(t, "activity", map[string]Facts{
		"P-1001": {PlannedCost: planned("960000.00")}, "P-1003": {PlannedCost: planned("100000.00")},
	})
}

// itemIDs returns the IDs of a list's plans, in order.
func itemIDs(list PlanList) []string {
	planIDs := []string{}
	for _, item := range list.Items {
		planIDs = append(planIDs, item.ID)
	}
	return planIDs
}

// The Plans list is sorted by year, newest first, then by name; status and year narrow it; total counts
// the plans in the team scope, and the summary covers the scope and the chosen year whatever the status
// (FR-001, FR-003, D-080, US2 scenarios 1 and 2).
func TestList_FiltersSortAndSummary_FR001_FR003(t *testing.T) {
	f := newFixture(t)
	f.putDemoPlans(t)
	usdOf := func(text string) *money.Money { m := money.USD(usd(t, text)); return &m }

	all, err := f.service.List(t.Context(), priya, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	draft2026, err := f.service.List(t.Context(), priya, ListFilter{Status: "draft", Year: ptr(2026)})
	if err != nil {
		t.Fatalf("List(draft, 2026): %v", err)
	}

	if ids := itemIDs(all); !slices.Equal(ids, []string{"P-1003", "P-1001", "P-1002"}) || all.Total != 3 ||
		!slices.Equal(all.Years, []int{2027, 2026}) {
		t.Errorf("all: %q, total %d, years %v", ids, all.Total, all.Years)
	}
	wantAll := ListSummary{Plans: 3, Draft: 2, Published: 1, PlannedCost: usdOf("1060000.00"), Budget: usdOf("2500000.00")}
	if !reflect.DeepEqual(all.Summary, wantAll) {
		t.Errorf("summary = %+v, want %+v", all.Summary, wantAll)
	}
	if ids := itemIDs(draft2026); !slices.Equal(ids, []string{"P-1002"}) || draft2026.Total != 3 {
		t.Errorf("draft 2026: %q, total %d; want P-1002 of 3", ids, draft2026.Total)
	}
	want2026 := ListSummary{Plans: 2, Draft: 1, Published: 1, PlannedCost: usdOf("960000.00"), Budget: usdOf("2000000.00")}
	if !reflect.DeepEqual(draft2026.Summary, want2026) {
		t.Errorf("summary of 2026 = %+v, want %+v", draft2026.Summary, want2026)
	}
	item := all.Items[1] // FY2026 Lubricants Europe
	if item.Owner == nil || item.Owner.Name != "Priya Raman" || item.Figures.BudgetUsedPercent == nil ||
		*item.Figures.BudgetUsedPercent != 80 || len(item.Completeness.Missing) != 0 || item.Teams[0].Name != "Lubricants Europe" {
		t.Errorf("FY2026 Lubricants Europe = %+v", item)
	}

	_, err = f.service.List(t.Context(), priya, ListFilter{Scope: "mine", Status: "open"})
	assertError(t, err, problems("scope", "Choose all or myTeams.", "status", "Choose draft or published."))
}

// My teams keeps the plans with one of the person's teams; All plans is every plan (FR-004, Q-006, US2
// scenario 4).
func TestList_MyTeams_FR004(t *testing.T) {
	f := newFixture(t)
	f.putDemoPlans(t)

	mei, err := f.service.List(t.Context(), actorOf("U-1004", "Mei Lin"), ListFilter{Scope: ScopeMyTeams})
	carlos, _ := f.service.List(t.Context(), actorOf("U-1005", "Carlos Mendes"), ListFilter{Scope: ScopeMyTeams})
	alone, _ := f.service.List(t.Context(), nina, ListFilter{Scope: ScopeMyTeams})

	if ids := itemIDs(mei); err != nil || !slices.Equal(ids, []string{"P-1002"}) || mei.Total != 1 || !slices.Equal(mei.Years, []int{2026}) {
		t.Errorf("Mei: %q, total %d, years %v, %v", ids, mei.Total, mei.Years, err)
	}
	if ids := itemIDs(carlos); !slices.Equal(ids, []string{"P-1003", "P-1002"}) {
		t.Errorf("Carlos: %q, want both of his teams' plans", ids)
	}
	if len(alone.Items) != 0 || alone.Total != 0 || alone.Summary.Budget != nil || alone.Summary.PlannedCost != nil {
		t.Errorf("Nina: %+v, want nothing and no totals", alone)
	}
}

// The search finds any part of the name or of the ID, ignoring letter case and surrounding spaces (FR-003,
// US2 scenario 3).
func TestList_SearchNameOrID_FR003(t *testing.T) {
	f := newFixture(t)
	f.putDemoPlans(t)
	for _, q := range []string{"MOBILITY", "p-1002", " 1002 "} {
		list, err := f.service.List(t.Context(), priya, ListFilter{Q: q})

		if ids := itemIDs(list); err != nil || !slices.Equal(ids, []string{"P-1002"}) || list.Total != 3 {
			t.Errorf("q %q: %q, total %d, %v; want P-1002 of 3", q, ids, list.Total, err)
		}
	}
}

// With 200 plans the list asks each level below once for all of them, never once per plan (SC-009).
func TestList_200PlansOneFactsCallPerLevel_SC009(t *testing.T) {
	f := newFixture(t)
	many := make([]Plan, 0, 200)
	for i := range 200 {
		many = append(many, seedPlan(t, fmt.Sprintf("P-%d", 1001+i), fmt.Sprintf("Plan %03d", i), 2026, teamLubricants, priya.ID, "1000.00", StatusDraft))
	}
	f.put(t, many...)
	campaigns := f.addLevel(t, "campaign", nil)
	activities := f.addLevel(t, "activity", nil)

	list, err := f.service.List(t.Context(), priya, ListFilter{})

	if err != nil || len(list.Items) != 200 || campaigns.factCalls != 1 || activities.factCalls != 1 {
		t.Errorf("%d plans, %v; facts asked %d and %d times, want once per level", len(list.Items), err, campaigns.factCalls, activities.factCalls)
	}
}

// actorOf returns a person as an actor.
func actorOf(id, name string) actor.Actor {
	return actor.Actor{ID: id, Name: name}
}

// User story 3 — Understand a plan at a glance.

// The plan page's figures, goals, completeness and contents come from what the levels below report, in
// the reader's display currency: remaining −100,000.00 USD (over-allocated), 80 % used, 4 of 7 activities
// approved, 60 % of the INQ goal planned, and the campaign that needs attention (FR-012, FR-015 – FR-017,
// US3 scenarios 2, 3 and 5).
func TestGet_FactsFromLevels_FR012_FR016_FR017(t *testing.T) {
	f := newFixture(t)
	in := input("FY2026 Lubricants Europe")
	in.Budget = &money.Input{Amount: "1200000.00", Currency: "USD"}
	in.GoalAmount = ptr("250000")
	created := f.create(t, priya, in)
	planned := usd(t, "960000.00")
	retail := AttentionItem{Record: errs.RecordRef{Type: "campaign", ID: "C-1001", Name: "Retail growth"},
		Missing: []settings.MissingField{{Field: "attributes.SA-1003", Label: "Cluster"}}, BlocksPublish: true}
	f.addLevel(t, "campaign", map[string]Facts{created.ID: {
		Campaigns: 2, Initiatives: 4, Activities: 7, ApprovedActivities: 4, LineItems: 15,
		CampaignBudgets: usd(t, "1300000.00"), PlannedCost: &planned,
		ActivityGoals:  GoalTotals{INQ: 150_000, MQL: 3_000, Revenue: money.USD(usd(t, "1500000.00"))},
		NeedsAttention: []AttentionItem{retail},
	}})

	view, err := f.service.Get(t.Context(), priya, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	usdOf := func(text string) *money.Money { m := money.USD(usd(t, text)); return &m }
	wantFigures := Figures{
		Currency: "USD", Budget: usdOf("1200000.00"), PlannedCost: usdOf("960000.00"), CampaignBudgets: *usdOf("1300000.00"),
		RemainingBudget: usdOf("-100000.00"), OverAllocatedBy: usdOf("100000.00"), BudgetUsedPercent: ptr(80),
		Activities: 7, ActivitiesApproved: 4, ApprovalProgress: ProgressInProgress,
	}
	if !reflect.DeepEqual(view.Figures, wantFigures) {
		t.Errorf("figures = %+v\nwant      %+v", view.Figures, wantFigures)
	}
	if view.Goals.PlannedPercent == nil || *view.Goals.PlannedPercent != 60 || view.Goals.Activities.INQ != 150_000 {
		t.Errorf("goals = %+v; want 60 %% of the INQ goal planned", view.Goals)
	}
	if !reflect.DeepEqual(view.Completeness.ItemsBelow, []AttentionItem{retail}) || len(view.Completeness.Missing) != 0 {
		t.Errorf("completeness = %+v", view.Completeness)
	}
	if view.Contents != (Contents{Campaigns: 2, Initiatives: 4, Activities: 7, ApprovedActivities: 4, LineItems: 15}) {
		t.Errorf("contents = %+v", view.Contents)
	}

	f.directory.rates[daniel.ID] = people.DisplayRate{Code: "EUR", Rate: "0.92"}
	inEuros, err := f.service.Get(t.Context(), daniel, created.ID)
	if err != nil || inEuros.Figures.Currency != "EUR" || inEuros.Goals.Plan.Revenue.Format() != "2,300,000.00 EUR" ||
		inEuros.Goals.Activities.Revenue.Format() != "1,380,000.00 EUR" || inEuros.Budget.Format() != "1,200,000.00 USD" {
		t.Errorf("in euros: figures %s, goal revenue %v, activities %v, typed budget %v, %v", inEuros.Figures.Currency,
			inEuros.Goals.Plan.Revenue, inEuros.Goals.Activities.Revenue, inEuros.Budget, err)
	}
}

// A plan without business unit shows it as the one field needed (FR-017, US3 scenario 4); an owner who is
// in none of the plan's teams is flagged with the R1 message (FR-021, D-085).
func TestGet_MissingFieldsAndOwnerProblem_FR017_FR021(t *testing.T) {
	f := newFixture(t)
	p := seedPlan(t, "P-1001", "No business unit", 2026, teamMobility, priya.ID, "", StatusDraft)
	p.Attributes = settings.AttributeValues{}
	f.put(t, p)

	view, err := f.service.Get(t.Context(), priya, "P-1001")

	wantMissing := []settings.MissingField{{Field: "ownerId", Label: "Owner"}, {Field: "attributes.SA-BU", Label: "Business unit"}}
	if err != nil || !reflect.DeepEqual(view.Completeness.Missing, wantMissing) ||
		view.OwnerProblem != "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field." {
		t.Errorf("missing %+v, owner problem %q, %v", view.Completeness.Missing, view.OwnerProblem, err)
	}
}

// Cross-cutting rules.

// Every change of a plan writes exactly one activity-log entry naming who made it: create, edit, publish,
// unpublish, copy (on the new plan) and delete (FR-031).
func TestEveryChangeLogged_FR031(t *testing.T) {
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	edit := updateOf(f.stored(t, plan.ID))
	edit.Description = "Lubricants marketing."
	steps := []func() error{
		func() error { _, err := f.service.Update(t.Context(), daniel, plan.ID, edit); return err },
		func() error { _, err := f.service.Publish(t.Context(), priya, plan.ID); return err },
		func() error {
			_, err := f.service.Unpublish(t.Context(), daniel, plan.ID, UnpublishRequest{Reason: "Review"})
			return err
		},
		func() error {
			_, err := f.service.Copy(t.Context(), daniel, plan.ID, CopyRequest{Name: "FY2027 Lubricants Test", Year: ptr(2027)})
			return err
		},
		func() error { _, err := f.service.Delete(t.Context(), priya, plan.ID); return err },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("step %d: %v", i+1, err)
		}
	}

	type logged struct {
		action history.Action
		verb   string
		actor  actor.Actor
	}
	var got []logged
	for _, e := range slices.Backward(f.entries(t)) {
		got = append(got, logged{e.Action, e.Verb, e.Actor})
	}
	want := []logged{
		{history.ActionCreated, "created", priya}, {history.ActionChanged, "changed", daniel},
		{history.ActionStatusChanged, "published", priya}, {history.ActionStatusChanged, "unpublished", daniel},
		{history.ActionCopied, "copied", daniel}, {history.ActionDeleted, "deleted", priya},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("log, oldest first = %+v\nwant                %+v", got, want)
	}
}

// There are no role checks: anyone signed in may edit, publish, unpublish, copy and delete a plan someone else
// created, even someone in none of its teams (FR-032, D-004).
func TestNoRoleChecks_FR032(t *testing.T) {
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	edit := updateOf(f.stored(t, plan.ID))
	edit.Description = "Edited by Nina."

	_, updateErr := f.service.Update(t.Context(), nina, plan.ID, edit)
	_, publishErr := f.service.Publish(t.Context(), nina, plan.ID)
	_, unpublishErr := f.service.Unpublish(t.Context(), nina, plan.ID, UnpublishRequest{Reason: "Review"})
	_, copyErr := f.service.Copy(t.Context(), nina, plan.ID, CopyRequest{Name: "Copied by Nina", Year: ptr(2026)})
	_, deleteErr := f.service.Delete(t.Context(), nina, plan.ID)

	for name, err := range map[string]error{"Update": updateErr, "Publish": publishErr, "Unpublish": unpublishErr,
		"Copy": copyErr, "Delete": deleteErr} {
		if err != nil {
			t.Errorf("%s by Nina: %v, want no refusal", name, err)
		}
	}
}

// Lookup reads a plan inside a caller's Write (for 005–007); References lists what every plan uses, the
// plans in the recycle bin included and marked, for the "Used by" counts of 002 and 003; a recycle-bin
// snapshot decodes to the same references as the stored plan.
func TestLookupAndReferences(t *testing.T) {
	f := newFixture(t)
	kept := f.create(t, priya, input("FY2026 Lubricants Test"))
	in := input("FY2026 Mobility Test")
	in.ModelID, in.TeamIDs, in.OwnerID = ptr(mobilityModel), []string{teamMobility}, nil
	deleted := f.create(t, priya, in)
	deletedPlan := f.stored(t, deleted.ID)
	entry, err := f.service.Delete(t.Context(), priya, deleted.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var found, missing bool
	var lookedUp Plan
	err = f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		if lookedUp, found, err = f.service.Lookup(ctx, priya, kept.ID); err != nil {
			return err
		}
		_, missing, err = f.service.Lookup(ctx, priya, deleted.ID)
		return err
	})
	refs, refsErr := f.service.References(t.Context(), priya)
	fromSnapshot, snapshotErr := ReferencesFromSnapshot(entry.Snapshots[0].Data)

	if err != nil || !found || lookedUp.Name != "FY2026 Lubricants Test" || missing {
		t.Errorf("Lookup = %q, %v; deleted found %v; %v", lookedUp.Name, found, missing, err)
	}
	want := []References{
		{PlanID: kept.ID, Name: "FY2026 Lubricants Test", ModelID: lubricantsModel, OwnerID: priya.ID,
			TeamIDs: []string{teamLubricants}, Attributes: attributesOf(lubricants)},
		{PlanID: deleted.ID, Name: "FY2026 Mobility Test", ModelID: mobilityModel, TeamIDs: []string{teamMobility},
			Attributes: attributesOf(lubricants), InRecycleBin: true},
	}
	if got, expected := asJSON(t, refs), asJSON(t, want); refsErr != nil || got != expected {
		t.Errorf("References = %s, %v\nwant         %s", got, refsErr, expected)
	}
	if got, expected := asJSON(t, fromSnapshot), asJSON(t, referencesOf(deletedPlan, true)); snapshotErr != nil || got != expected {
		t.Errorf("ReferencesFromSnapshot = %s, %v\nwant                     %s", got, snapshotErr, expected)
	}
}

// asJSON returns v as compact JSON, so that stored attribute values compare whatever their spacing.
func asJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(data)
}
