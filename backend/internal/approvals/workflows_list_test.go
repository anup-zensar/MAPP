package approvals

import (
	"reflect"
	"strings"
	"testing"

	"mapp/internal/platform/errs"
)

// The list is sorted by name and shows each workflow's levels text, the stored activities that use it —
// in the Approval workflow field or through a round — those waiting for approval with it, and a warning
// for each level without an active approver (FR-005, edge case).
func TestListWorkflows_UsageAndWarnings_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, chinaFlow) // Created: used by its field
	f.addActivity(t, "A-1002", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1002", "") // waiting
	f.addActivity(t, "A-1003", "Fleet roadshow", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1003", "")
	f.decline(t, daniel, "A-1003", "No.")
	moved := f.activities.facts["A-1003"]
	moved.WorkflowID = chinaFlow // Lubricants keeps it through its round
	f.activities.facts["A-1003"] = moved
	f.addActivity(t, "A-1004", "Deleted one", priya.ID, chinaFlow)
	delete(f.activities.facts, "A-1004") // in the recycle bin: not counted
	f.deactivatePerson(t, daniel)

	views := f.workflowViews(t, alex)

	type usage struct {
		id, levelsText  string
		usedBy, waiting int
		warnings        []string
	}
	var got []usage
	for _, v := range views {
		got = append(got, usage{v.ID, v.LevelsText, v.UsedByCount, v.WaitingCount, v.Warnings})
	}
	want := []usage{
		{chinaFlow, "Level 1 · Carlos Mendes → Level 2 · Alex Morgan", 3, 1, []string{}},
		{lubricantsFlow, "Level 1 · Daniel Okoro (inactive)", 1, 0, []string{"Level 1 has no active approver"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("workflows =\n%+v\nwant\n%+v", got, want)
	}
}

// The details list the activities that use the workflow, by plan name and then activity name, with their
// path, displayed status and whether they wait for approval; a workflow that is gone answers not_found
// (FR-005).
func TestGetWorkflow_ActivitiesUsingIt_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Zeta event", priya.ID, chinaFlow)
	mobility := f.addActivity(t, "A-1002", "Alpha launch", mei.ID, chinaFlow)
	mobility.Path = []errs.RecordRef{
		{Type: "plan", ID: "P-1003", Name: "FY2026 Mobility China"},
		{Type: "campaign", ID: "C-1005", Name: "Retail growth"},
		{Type: "initiative", ID: "I-1009", Name: "Fleet cards"},
	}
	f.activities.facts["A-1002"] = mobility
	f.addActivity(t, "A-1003", "beta fair", priya.ID, chinaFlow)
	f.submit(t, mei, "A-1002", "")

	details, err := f.service.GetWorkflow(t.Context(), alex, chinaFlow)
	_, gone := f.service.GetWorkflow(t.Context(), alex, "WF-9999")

	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	var got []string
	for _, a := range details.Activities {
		line := a.Activity.ID + " " + a.Activity.Name + " in " + planName(a.Path) + " " + a.DisplayStatus
		if a.Waiting {
			line += " waiting"
		}
		got = append(got, line)
	}
	want := []string{
		"A-1003 beta fair in FY2026 Lubricants Europe created",
		"A-1001 Zeta event in FY2026 Lubricants Europe created",
		"A-1002 Alpha launch in FY2026 Mobility China submitted waiting",
	}
	if !reflect.DeepEqual(got, want) || details.Workflow.UsedByCount != 3 || details.Workflow.WaitingCount != 1 {
		t.Errorf("activities =\n%s\nwant\n%s\n(used by %d, waiting %d)", strings.Join(got, "\n"), strings.Join(want, "\n"),
			details.Workflow.UsedByCount, details.Workflow.WaitingCount)
	}
	assertError(t, gone, errs.NotFound("This approval workflow no longer exists — someone may have deleted it."))
}
