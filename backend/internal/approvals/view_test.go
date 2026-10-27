package approvals

import (
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// The Approval section offers the actions of the activity's status: Submit for a Created one, Re-submit
// for a Declined one, Approve, Decline and Recall while Submitted; an Approved one cannot be recalled and
// says why (FR-009, FR-014, FR-021).
func TestApproval_Actions_FR009_FR014_FR021(t *testing.T) {
	t.Parallel()
	available := ActionAvailability{Available: true}
	tests := []struct {
		status string
		want   ActionSet
	}{
		{StatusCreated, ActionSet{Submit: available}},
		{StatusDeclined, ActionSet{Resubmit: available}},
		// 017 FR-017: approve and decline follow who may decide the open round's level (TestApproval_ViewerAuthority_FR017).
		{StatusSubmitted, ActionSet{Recall: available}},
		{StatusApproved, ActionSet{Recall: ActionAvailability{Reason: "Approved activities can't be recalled. " +
			"Changing its planned cost, dates or line-item amounts sends it back for approval."}}},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			facts := f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
			facts.Status = tt.status
			f.activities.facts["A-1001"] = facts

			approval, err := f.service.Approval(t.Context(), priya, "A-1001")

			if err != nil {
				t.Fatalf("Approval: %v", err)
			}
			if approval.Actions != tt.want {
				t.Errorf("actions = %+v, want %+v", approval.Actions, tt.want)
			}
		})
	}
}

// The section shows every round, newest first: its chain — each level with its designated approvers and
// its state (approved or declined with the decision, waiting, not reached, not decided) — and its history,
// newest first, with the round, level, approvers at the time, actor, "on behalf of" and comment (FR-024,
// FR-025).
func TestApproval_ChainAndHistory_FR024_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.addActivity(t, "A-1008", "Grand opening Shanghai", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.clock.Advance(time.Hour)
	f.approve(t, alex, "A-1009", "For Carlos.")
	f.clock.Advance(time.Hour)
	f.decline(t, alex, "A-1009", "Split the cost.")
	f.clock.Advance(time.Hour)
	f.submit(t, mei, "A-1009", "")
	f.submit(t, mei, "A-1008", "")
	f.endDirectly(t, "A-1008", ActionRecalled, OutcomeRecalled, StatusCreated, mei, "Wrong workflow.")

	approval, err := f.service.Approval(t.Context(), carlos, "A-1009")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	recalled, err := f.service.Approval(t.Context(), mei, "A-1008")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}

	if len(approval.Rounds) != 2 || approval.Rounds[0].Number != 2 || approval.Rounds[1].Number != 1 {
		t.Fatalf("rounds = %+v, want round 2, then round 1", approval.Rounds)
	}
	if states := chainStates(approval.Rounds[0]) + " / " + chainStates(approval.Rounds[1]); states !=
		"L1 waiting, L2 notReached / L1 approved, L2 declined" {
		t.Errorf("chains = %s", states)
	}
	if states := chainStates(recalled.Rounds[0]); states != "L1 notDecided, L2 notReached" {
		t.Errorf("recalled chain = %s, want L1 notDecided, L2 notReached", states)
	}
	carlosView := PersonView{ID: carlos.ID, Name: "Carlos Mendes", Initials: "CM", Active: true}
	alexView := PersonView{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}
	want := []HistoryEntryView{
		{ID: "AH-1003", RoundNumber: 1, Level: 2, LevelLabel: "Level 2", Approvers: []PersonView{alexView}, Actor: alex,
			OnBehalfOf: []actor.Actor{}, Action: "declined", At: start.Add(2 * time.Hour), Comment: "Split the cost.", Changes: []ChangeView{},
			Capacity: CapacityApprover},
		{ID: "AH-1002", RoundNumber: 1, Level: 1, LevelLabel: "Level 1", Approvers: []PersonView{carlosView}, Actor: alex,
			OnBehalfOf: []actor.Actor{carlos}, Action: "approved", At: start.Add(time.Hour), Comment: "For Carlos.", Changes: []ChangeView{},
			Capacity: CapacityOnBehalf},
		{ID: "AH-1001", RoundNumber: 1, Level: 1, LevelLabel: "Level 1", Approvers: []PersonView{carlosView}, Actor: mei,
			OnBehalfOf: []actor.Actor{}, Action: "submitted", At: start, Changes: []ChangeView{}},
	}
	if !reflect.DeepEqual(approval.Rounds[1].History, want) {
		t.Errorf("history of round 1 =\n%+v\nwant\n%+v", approval.Rounds[1].History, want)
	}
	if approval.LatestDecline != nil || !approval.Viewer.IsCurrentApprover {
		t.Errorf("latest decline %+v, viewer %+v; want none while Submitted, Carlos the current approver", approval.LatestDecline, approval.Viewer)
	}
}

// The sentences of the changed values of a restart (data model §3).
func TestChangeText_Sentences_FR022(t *testing.T) {
	t.Parallel()
	tests := []struct {
		change ChangeText
		want   string
	}{
		{ChangeText{"Planned cost", "$50,000.00", "$65,000.00"}, "Planned cost changed from $50,000.00 to $65,000.00"},
		{ChangeText{"End date", "Not filled in", "30 Jun 2026"}, "End date added: 30 Jun 2026"},
		{ChangeText{"End date", "30 Jun 2026", "Not filled in"}, "End date removed (was 30 Jun 2026)"},
		{ChangeText{Label: "Line item 'Agency fee' (L-1003) added: 1,200.00 USD"}, "Line item 'Agency fee' (L-1003) added: 1,200.00 USD"},
	}
	for _, tt := range tests {
		if got := changeText(tt.change); got != tt.want {
			t.Errorf("changeText(%+v) = %q, want %q", tt.change, got, tt.want)
		}
	}
}

// Before the first submission the section shows the activity's workflow and its levels as they are now
// ("Not submitted yet"); the viewer is told whether they own the activity (FR-024, edge case), and the
// owner is named, as the decline dialog needs (D-214); an activity without an owner has none.
func TestApproval_NotSubmittedYet_FR024(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)

	asOwner, err := f.service.Approval(t.Context(), mei, "A-1009")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	asOther, err := f.service.Approval(t.Context(), alex, "A-1009")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	_, gone := f.service.Approval(t.Context(), mei, "A-9999")

	want := ActivityApproval{
		ActivityID: "A-1009", ActivityName: "Fleet card launch", ActivityVersion: 1, Status: StatusCreated,
		DisplayStatus: StatusCreated, OwnerID: mei.ID, Owner: &PersonView{ID: mei.ID, Name: "Mei Lin", Initials: "ML", Active: true},
		Workflow: &WorkflowRef{ID: chinaFlow, Name: "China BC approval workflow", Active: true},
		UpcomingLevels: []LevelView{
			{Number: 1, Label: "Level 1", Approvers: []PersonView{{ID: carlos.ID, Name: "Carlos Mendes", Initials: "CM", Active: true}},
				ApproverType: SourcePeople, Mode: ModeAny, Summary: "Level 1 · Carlos Mendes"},
			{Number: 2, Label: "Level 2", Approvers: []PersonView{{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}},
				ApproverType: SourcePeople, Mode: ModeAny, Summary: "Level 2 · Alex Morgan"},
		},
		Rounds:  []RoundView{},
		Viewer:  Viewer{IsOwner: true, OnBehalfOf: []PersonView{}, AsDelegateOf: []PersonView{}},
		Actions: ActionSet{Submit: ActionAvailability{Available: true}},
	}
	if !reflect.DeepEqual(asOwner, want) {
		t.Errorf("approval =\n%+v\nwant\n%+v", asOwner, want)
	}
	if asOther.Viewer.IsOwner {
		t.Errorf("Alex sees isOwner true, want false")
	}
	assertError(t, gone, errs.NotFound("This activity no longer exists — someone may have deleted it."))
	f.addActivity(t, "A-1010", "Nobody's activity", "", chinaFlow)
	if ownerless, err := f.service.Approval(t.Context(), alex, "A-1010"); err != nil || ownerless.Owner != nil {
		t.Errorf("owner of an activity without one = %+v, %v; want nil", ownerless.Owner, err)
	}
}
