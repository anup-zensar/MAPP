package plan

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"mapp/internal/people"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// The plan page shows a plan by its ID; an ID that does not exist answers not found (FR-011).
func TestGet_NotFound_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	created := f.create(t, priya, input("Visible plan"))

	got, err := f.service.Get(t.Context(), daniel, created.ID)
	_, errGone := f.service.Get(t.Context(), daniel, "P-9999")

	if err != nil || !reflect.DeepEqual(got, created) {
		t.Errorf("Get = %+v, %v\nwant  %+v", got, err, created)
	}
	assertError(t, errGone, errs.NotFound("This plan no longer exists — someone may have deleted it."))
}

// User story 3 — Understand a plan at a glance.

// The plan page's figures, goals, completeness and contents come from what the levels below report, in
// the reader's display currency: remaining −100,000.00 USD (over-allocated), 80 % used, 4 of 7 activities
// approved, 60 % of the INQ goal planned, and the campaign that needs attention (FR-012, FR-015 – FR-017,
// US3 scenarios 2, 3 and 5).
func TestGet_FactsFromLevels_FR012_FR016_FR017(t *testing.T) { //nolint:gocyclo // 14: the figures, goals, completeness and contents of one plan, then its figures in euros
	t.Parallel()
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
	t.Parallel()
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

// Lookup reads a plan inside a caller's Write (for 005–007); References lists what every plan uses, the
// plans in the recycle bin included and marked, for the "Used by" counts of 002 and 003; a recycle-bin
// snapshot decodes to the same references as the stored plan.
func TestLookupAndReferences(t *testing.T) { //nolint:gocyclo // 11: two lookups, then the references of both plans and of a snapshot
	t.Parallel()
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
		var lookupErr error
		if lookedUp, found, lookupErr = f.service.Lookup(ctx, priya, kept.ID); lookupErr != nil {
			return lookupErr
		}
		_, missing, lookupErr = f.service.Lookup(ctx, priya, deleted.ID)
		return lookupErr
	})
	refs, refsErr := f.service.References(t.Context(), priya)
	fromSnapshot, snapshotErr := ReferencesFromSnapshot(entry.Snapshots[0].Data)

	if err != nil || !found || lookedUp.Name != "FY2026 Lubricants Test" || missing {
		t.Errorf("Lookup = %q, %v; deleted found %v; %v", lookedUp.Name, found, missing, err)
	}
	want := []References{
		{PlanID: kept.ID, Name: "FY2026 Lubricants Test", ModelID: lubricantsModel, OwnerID: priya.ID,
			TeamIDs: []string{teamLubricants}, Attributes: attributesOf(lubricants), Currency: "USD"},
		{PlanID: deleted.ID, Name: "FY2026 Mobility Test", ModelID: mobilityModel, TeamIDs: []string{teamMobility},
			Attributes: attributesOf(lubricants), InRecycleBin: true, Currency: "USD"},
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
