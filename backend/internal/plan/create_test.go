package plan

import (
	"fmt"
	"reflect"
	"testing"

	"mapp/internal/overspend"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

func TestDefaults_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	defaults, err := f.service.Defaults(t.Context(), priya)

	want := Defaults{
		CurrentYear: 2026, Years: []int{2026, 2027, 2028, 2029, 2030, 2031, 2032, 2033, 2034, 2035, 2036}, Year: 2026,
		TeamIDs: []string{teamLubricants}, OwnerID: ptr(priya.ID), // Closed team is inactive
		DefaultSpread: SpreadEvenly, GoalStage: StageINQ, // two active models: none chosen
		Currency: "USD", PlanType: TypeOperational, Overspend: overspend.Warn,
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

// A plan without teams, owner or business unit is saved as a Draft that lists its missing fields.
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
	assertError(t, err, problems("name", "Enter a plan name.", "year", "Choose a year from 2026 to 2036.", "modelId", "Choose a model."))
}

// A name another plan has in other letters is reported with every other problem, and nothing is saved.
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

// Last year and the year after current + 10 are refused, with every other problem (D-326).
func TestCreate_YearRange_034_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for _, year := range []int{2026, 2036} {
		in := input(fmt.Sprintf("Plan of %d", year))
		in.Year = ptr(year)
		if view := f.create(t, priya, in); view.Year != year {
			t.Errorf("year = %d, want %d", view.Year, year)
		}
	}
	for _, year := range []*int{ptr(2025), ptr(2037), nil} {
		in := input("Out of range")
		in.Year = year

		_, err := f.service.Create(t.Context(), priya, in)

		assertError(t, err, problems("year", "Choose a year from 2026 to 2036."))
	}
}

func TestCreate_OwnerNeedsTeams_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	in := input("No teams")
	in.TeamIDs = nil

	_, err := f.service.Create(t.Context(), priya, in)

	assertError(t, err, problems("ownerId", "Choose teams first — the owner must belong to one of the plan's teams."))
}

func TestCreate_AttributeProblemsCollected_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.settings.attributeProblems = []errs.FieldError{{Field: "attributes.SA-BU", Message: "'Marine' is inactive — choose another option."}}

	_, err := f.service.Create(t.Context(), priya, input(""))

	assertError(t, err, problems("name", "Enter a plan name.", "attributes.SA-BU", "'Marine' is inactive — choose another option."))
}

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
	f.assertLubricantsReferences(t, view)
	if len(view.Completeness.Missing) != 0 || view.Goals.Plan.MQL == nil || *view.Goals.Plan.MQL != 5000 {
		t.Errorf("completeness %+v, goals %+v; want all filled in and MQL 5,000", view.Completeness, view.Goals.Plan)
	}
	assertCreatedByPriya(t, f.entries(t), "P-1001", "FY2026 Lubricants Test")
}

func (f *fixture) assertLubricantsReferences(t *testing.T, view PlanView) {
	t.Helper()
	if !reflect.DeepEqual(view.Teams, []people.TeamRef{f.directory.teams[teamLubricants]}) ||
		!reflect.DeepEqual(view.Owner, ptr(f.directory.people[priya.ID])) || view.Model.Name != "Lubricants 1.0" ||
		!reflect.DeepEqual(view.BusinessUnit, []OptionRef{{ID: lubricants, Label: "Lubricants", Active: true}}) {
		t.Errorf("references: teams %+v, owner %+v, model %+v, business unit %+v", view.Teams, view.Owner, view.Model, view.BusinessUnit)
	}
}

func assertCreatedByPriya(t *testing.T, entries []history.Entry, id, name string) {
	t.Helper()
	if len(entries) != 1 || entries[0].Action != history.ActionCreated || entries[0].RecordID != id ||
		entries[0].RecordType != "plan" || entries[0].Actor != priya || entries[0].RecordName != name {
		t.Errorf("log = %+v, want one created entry by Priya", entries)
	}
}
