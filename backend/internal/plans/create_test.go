package plans

import (
	"fmt"
	"reflect"
	"testing"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// User story 1 — Create a plan.

// A new plan starts with the current year (choices 2025–2031), the person's active teams, the person as
// owner when they have a team, the only active model, No, Evenly and INQ (FR-006, US1 scenario 1).
func TestDefaults_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	defaults, err := f.service.Defaults(t.Context(), priya)

	want := Defaults{
		CurrentYear: 2026, Years: []int{2025, 2026, 2027, 2028, 2029, 2030, 2031}, Year: 2026,
		TeamIDs: []string{teamLubricants}, OwnerID: ptr(priya.ID), // Closed team is inactive
		DefaultSpread: SpreadEvenly, GoalStage: StageINQ, // two active models: none chosen
		Currency: "USD", PlanType: TypeOperational, // 016 FR-011, FR-032
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
	t.Parallel()
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
	t.Parallel()
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

// A new plan's year is from the current year − 1 to + 5 (FR-006).
func TestCreate_YearRange_FR006(t *testing.T) {
	t.Parallel()
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

// Without a team the owner can't be chosen: the owner must belong to one of the plan's teams (FR-007, R1,
// US1 scenario 4).
func TestCreate_OwnerNeedsTeams_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	in := input("No teams")
	in.TeamIDs = nil

	_, err := f.service.Create(t.Context(), priya, in)

	assertError(t, err, problems("ownerId", "Choose teams first — the owner must belong to one of the plan's teams."))
}

// The problems Settings finds in the Shell attribute values are reported with the others (FR-008).
func TestCreate_AttributeProblemsCollected_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.settings.attributeProblems = []errs.FieldError{{Field: "attributes.SA-BU", Message: "'Marine' is inactive — choose another option."}}

	_, err := f.service.Create(t.Context(), priya, input(""))

	assertError(t, err, problems("name", "Enter a plan name.", "attributes.SA-BU", "'Marine' is inactive — choose another option."))
}

// A new plan is Draft with the next permanent ID, P-1001, its creator and time, and one "created" log entry
// by its creator (FR-010, FR-031, US1 scenario 2).
func TestCreate_DraftWithIDAndLog_FR010_FR031(t *testing.T) {
	t.Parallel()
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
