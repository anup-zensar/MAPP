package workbook

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/errs"
)

func TestApply_CreateUpdateUnchanged_012FR029_FR030(t *testing.T) {
	t.Parallel()
	f := &fakePlanning{plan: testPlan()}
	up := uploadOf(t, map[string][][]string{
		"Campaigns":   {{"ID", "Campaign name", "Owner"}, {"C-1", "Helix launch", "priya.raman@shell.com"}, {"", "Rimula", ""}},
		"Initiatives": {{"Campaign", "Initiative name"}, {"rimula ", "Events"}, {"C-1", "Print"}},
		"Activities": {{"ID", "Campaign", "Initiative", "Activity name", "Activity type", "Description"},
			{"A-2", "Helix launch", "Digital", "Podcast", "Event", "Weekly"}},
	})

	check := applied(t, f, up)

	wantCreated := []string{"C-901 Rimula under P-1", "I-902 Events under C-901", "I-903 Print under C-1"}
	if diff := cmp.Diff(wantCreated, f.created); diff != "" {
		t.Errorf("created (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]map[string]bool{{fieldDescription: true}}, f.updated); diff != "" {
		t.Errorf("updated fields (-want +got):\n%s", diff)
	}
	want := []LevelCounts{{Level: LevelCampaign, Create: 1, Unchanged: 1}, {Level: LevelInitiative, Create: 2},
		{Level: LevelActivity, Update: 1}, {Level: LevelLineItem}}
	if diff := cmp.Diff(want, check.Levels); diff != "" || check.Ready != 5 || check.NothingToChange {
		t.Errorf("levels (-want +got):\n%s ready %d", diff, check.Ready)
	}
}

func TestApply_ParentsIDsAndLeftOutBelow_012FR029_FR033(t *testing.T) {
	t.Parallel()
	f := &fakePlanning{plan: testPlan()}
	up := uploadOf(t, map[string][][]string{
		"Campaigns":   {{"Campaign name", "Start date"}, {"Spring", "2026-02-30"}},
		"Initiatives": {{"ID", "Campaign", "Initiative name"}, {"", "Spring", "Forecourt"}, {"I-1", "Spring", "Digital"}},
		"Activities": {{"ID", "Campaign", "Initiative", "Activity name", "Activity type"},
			{"", "Spring", "Forecourt", "Toppers", "Event"}, {"", "Helix launch", "Nowhere", "Toppers", "Event"},
			{"C-1", "Helix launch", "Digital", "Toppers", "Event"}},
	})

	check := applied(t, f, up)

	want := []string{
		"Campaigns 2 Start date: Not a real date — write it as 2026-03-12",
		"Initiatives 2 : Its campaign on Campaigns row 2 has problems",
		"Initiatives 3 : Its campaign on Campaigns row 2 has problems",
		"Activities 2 : Its campaign on Campaigns row 2 has problems",
		"Activities 3 Initiative: No initiative called 'Nowhere' here — add it on the Initiatives sheet or use a name from the template",
		"Activities 4 ID: C-1 is not an activity of this plan",
	}
	if diff := cmp.Diff(want, problemsOf(check)); diff != "" {
		t.Errorf("problems (-want +got):\n%s", diff)
	}
	if len(f.created) != 0 || check.WithProblems != 6 {
		t.Errorf("created %v, with problems %d", f.created, check.WithProblems)
	}
}

func TestApply_RefusalsUnderTheirColumns_ApprovedOnScreen_012FR031_FR032(t *testing.T) {
	t.Parallel()
	f := &fakePlanning{plan: testPlan(), refusals: map[string]error{
		"Webinar": errs.Validation(errs.FieldError{Field: "plannedCost.amount", Message: "Too much."},
			errs.FieldError{Field: "version", Message: "Reload."}),
	}}
	up := uploadOf(t, map[string][][]string{
		"Activities": {{"ID", "Campaign", "Initiative", "Activity name", "Activity type", "Planned cost", "Description"},
			{"A-1", "Helix launch", "Digital", "Launch film", "Event", "2000", ""},
			{"A-2", "Helix launch", "Digital", "Podcast", "Event", "", "Kept"},
			{"", "Helix launch", "Digital", "Webinar", "Event", "5", ""}},
	})

	check := applied(t, f, up)

	want := []string{
		"Activities 2 : Approved — changing its planned cost, dates or Ongoing needs approval again; change it on screen",
		"Activities 4 Planned cost: Too much.",
		"Activities 4 : Reload.",
	}
	if diff := cmp.Diff(want, problemsOf(check)); diff != "" {
		t.Errorf("problems (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]map[string]bool{{fieldDescription: true}}, f.updated); diff != "" {
		t.Errorf("updated (-want +got):\n%s", diff)
	}
}

func TestApply_UnexpectedErrorStopsTheRun_012FR022(t *testing.T) {
	t.Parallel()
	broken := errors.New("the database went away")
	f := &fakePlanning{plan: testPlan(), refusals: map[string]error{"Webinar": broken}}
	sheets, _, err := readUpload(uploadOf(t, map[string][][]string{
		"Activities": {{"Campaign", "Initiative", "Activity name", "Activity type"}, {"Helix launch", "Digital", "Webinar", "Event"}},
	}), f.plan.Choices)
	if err != nil {
		t.Fatal(err)
	}

	if err := newApplier(priya, f, f.plan).run(t.Context(), sheets); !errors.Is(err, broken) {
		t.Errorf("run error = %v, want the storage failure", err)
	}
}
