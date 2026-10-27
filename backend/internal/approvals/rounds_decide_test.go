package approvals

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Approving a level other than the last moves the round to the next level and keeps the activity
// Submitted; the next level's approvers are asked and the owner is told (FR-015, FR-035).
func TestApprove_MovesToNextLevel_FR015(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.clock.Advance(time.Hour)
	approvedAt := f.clock.Now()

	approval := f.approve(t, carlos, "A-1009", " Looks good. ")

	if facts := f.activities.facts["A-1009"]; facts.Status != StatusSubmitted || facts.Version != 2 {
		t.Errorf("activity = %s, version %d; want still submitted, version 2", facts.Status, facts.Version)
	}
	round := f.openRound(t, "A-1009")
	if round.CurrentLevel != 2 || round.Version != 2 || round.UpdatedBy != carlos || round.EndedAt != nil {
		t.Errorf("round = %+v, want open at level 2, changed by Carlos Mendes", round)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 approved, L2 waiting" {
		t.Errorf("chain = %s, want L1 approved, L2 waiting", states)
	}
	wantDecision := &Decision{Action: "approved", Actor: carlos, OnBehalfOf: []actor.Actor{}, At: approvedAt, Comment: "Looks good.",
		Capacity: CapacityApprover}
	if got := approval.Rounds[0].Levels[0].Decision; !reflect.DeepEqual(got, wantDecision) {
		t.Errorf("decision = %+v, want %+v", got, wantDecision)
	}
	assertLog(t, f, "A-1009", []string{
		`Carlos Mendes approved level 1 of activity 'Fleet card launch' "Looks good." ` +
			`(Round 1 · level 1 of 2 · now waiting for level 2 (Alex Morgan))`,
		"Mei Lin submitted activity 'Fleet card launch' [created → submitted] (Round 1 · China BC approval workflow · waiting for level 1 (Carlos Mendes))",
	})
	assertNotified(t, f, alex, "approvalNeeded: Approval needed — Carlos Mendes approved level 1 of 'Fleet card launch' (A-1009) — level 2 waits for you")
	assertNotified(t, f, mei, "levelApproved: Level 1 approved — Carlos Mendes approved level 1 of 2 of 'Fleet card launch' (A-1009)")
	assertNotified(t, f, carlos, "approvalNeeded: Approval needed — Mei Lin submitted 'Fleet card launch' (A-1009)")
}

// Approving the last level ends the round as approved and makes the activity Approved; the owner is told
// (FR-015, FR-035).
func TestApprove_LastLevelApproves_FR015(t *testing.T) { //nolint:gocyclo // 14: the activity, round, chain, actions, log and notices after one approval
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.approve(t, carlos, "A-1009", "")
	round := f.openRound(t, "A-1009")
	f.clock.Advance(time.Hour)
	approvedAt := f.clock.Now()

	approval := f.approve(t, alex, "A-1009", "")

	if facts := f.activities.facts["A-1009"]; facts.Status != StatusApproved || facts.Version != 3 || facts.UpdatedBy != alex {
		t.Errorf("activity = %s, version %d, by %v; want approved, version 3, by Alex Morgan", facts.Status, facts.Version, facts.UpdatedBy)
	}
	ended := f.rounds(t, "A-1009")[0]
	if ended.ID != round.ID || ended.Outcome != OutcomeApproved || ended.CurrentLevel != 2 || ended.EndedAt == nil || !ended.EndedAt.Equal(approvedAt) {
		t.Errorf("round = %+v, want approved at level 2, ended now", ended)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 approved, L2 approved" {
		t.Errorf("chain = %s, want both levels approved", states)
	}
	if approval.Status != StatusApproved || approval.Actions.Approve.Available || approval.Actions.Recall.Available {
		t.Errorf("approval: %s, actions %+v; want approved without approve or recall", approval.Status, approval.Actions)
	}
	if entries := f.log(t, "A-1009"); logLine(entries[0]) != "Alex Morgan approved activity 'Fleet card launch' [submitted → approved] (Round 1 · level 2 of 2)" {
		t.Errorf("latest log entry = %s", logLine(entries[0]))
	}
	assertNotified(t, f, mei,
		"approved: Activity approved — Alex Morgan approved 'Fleet card launch' (A-1009)",
		"levelApproved: Level 1 approved — Carlos Mendes approved level 1 of 2 of 'Fleet card launch' (A-1009)")
}

// Every level applies to every submission, in order, and any one designated approver of a level approves
// it (FR-004, D-205).
func TestApprove_EveryLevelAppliesAnyApprover_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addWorkflow(t, "WF-1003", "Two team leaders, then the champion", []string{daniel.ID, sara.ID}, []string{carlos.ID})
	f.addActivity(t, "A-1001", "Launch film", priya.ID, "WF-1003")
	f.submit(t, priya, "A-1001", "")

	approval := f.approve(t, sara, "A-1001", "")

	if approval.Status != StatusSubmitted || chainStates(approval.Rounds[0]) != "L1 approved, L2 waiting" {
		t.Errorf("approval: %s, chain %s; want still submitted, waiting at level 2", approval.Status, chainStates(approval.Rounds[0]))
	}
	if decision := approval.Rounds[0].Levels[0].Decision; len(decision.OnBehalfOf) != 0 {
		t.Errorf("Sara Novak approved on behalf of %v; she is an approver of the level", decision.OnBehalfOf)
	}
	assertNotified(t, f, carlos, "approvalNeeded: Approval needed — Sara Novak approved level 1 of 'Launch film' (A-1001) — level 2 waits for you")
	assertNotified(t, f, daniel, "approvalNeeded: Approval needed — Priya Raman submitted 'Launch film' (A-1001)")
}

// Declining needs a comment that is not blank; comments hold at most 2,000 characters. A refused
// decision changes nothing (FR-016).
func TestDecline_CommentRequired_FR016(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	round := f.openRound(t, "A-1004")
	level := version(1)

	_, blank := f.service.Decline(t.Context(), daniel, "A-1004", DecisionInput{RoundID: round.ID, Level: level, Comment: "  \n "})
	_, long := f.service.Decline(t.Context(), daniel, "A-1004", DecisionInput{RoundID: round.ID, Level: level, Comment: strings.Repeat("x", 2001)})
	_, longApproval := f.service.Approve(t.Context(), daniel, "A-1004", DecisionInput{RoundID: round.ID, Level: level, Comment: strings.Repeat("x", 2001)})

	assertError(t, blank, errs.Validation(errs.FieldError{Field: "comment", Message: "Add a comment explaining why the activity is declined."}))
	assertError(t, long, errs.Validation(errs.FieldError{Field: "comment", Message: "Use 2,000 characters or fewer."}))
	assertError(t, longApproval, errs.Validation(errs.FieldError{Field: "comment", Message: "Use 2,000 characters or fewer."}))
	if facts, actions := f.activities.facts["A-1004"], f.actions(t, "A-1004"); facts.Status != StatusSubmitted || len(actions) != 1 {
		t.Errorf("activity %s with %d history entries, want still submitted with only the submission", facts.Status, len(actions))
	}
}

// A decline ends the round at any level and makes the activity Declined; the comment heads its Approval
// section and the owner is told with the comment's first 80 characters (FR-016, FR-035).
func TestDecline_EndsRoundAtAnyLevel_FR016(t *testing.T) { //nolint:gocyclo // 14: the chains, round, note, actions, log and notice after two declines
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.addActivity(t, "A-1008", "Grand opening Shanghai", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.submit(t, mei, "A-1008", "")
	f.approve(t, carlos, "A-1009", "")
	f.clock.Advance(time.Hour)
	comment := "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3."

	atLevel2 := f.decline(t, alex, "A-1009", comment)
	atLevel1 := f.decline(t, carlos, "A-1008", "Not this year.")

	if chainStates(atLevel2.Rounds[0]) != "L1 approved, L2 declined" || chainStates(atLevel1.Rounds[0]) != "L1 declined, L2 notReached" {
		t.Errorf("chains = %s / %s", chainStates(atLevel2.Rounds[0]), chainStates(atLevel1.Rounds[0]))
	}
	ended := f.rounds(t, "A-1009")[0]
	if ended.Outcome != OutcomeDeclined || ended.CurrentLevel != 2 || ended.EndedAt == nil || f.activities.facts["A-1009"].Status != StatusDeclined {
		t.Errorf("round %+v, activity %s; want declined at level 2", ended, f.activities.facts["A-1009"].Status)
	}
	wantNote := &DeclineNote{RoundNumber: 1, Level: 2, LevelLabel: "Level 2", Actor: alex, OnBehalfOf: []actor.Actor{}, At: f.clock.Now(),
		Comment: comment, Capacity: CapacityApprover}
	if !reflect.DeepEqual(atLevel2.LatestDecline, wantNote) {
		t.Errorf("latest decline = %+v, want %+v", atLevel2.LatestDecline, wantNote)
	}
	if !atLevel2.Actions.Resubmit.Available || atLevel2.Actions.Decline.Available {
		t.Errorf("actions = %+v, want re-submit only", atLevel2.Actions)
	}
	if entries := f.log(t, "A-1009"); logLine(entries[0]) != "Alex Morgan declined activity 'Fleet card launch' [submitted → declined] \""+comment+"\" (Round 1 · level 2 of 2)" {
		t.Errorf("latest log entry = %s", logLine(entries[0]))
	}
	notifications := f.notifications(t, mei.ID)
	want := `Alex Morgan declined 'Fleet card launch' (A-1009): "The planned cost is above this quarter's budget — please reduce it or move part …"`
	if notifications[1].Kind != NotifyDeclined || notifications[1].Title != "Activity declined" || notifications[1].Text != want {
		t.Errorf("Mei's notification = %+v, want the decline quoting the comment's first 80 characters", notifications[1])
	}
}

// Anyone may decide (D-020); a person who is not an approver of the level decides on behalf of its
// approvers, and the confirmation, the history and the log say so (FR-017).
func TestDecide_OnBehalfOf_FR017(t *testing.T) { //nolint:gocyclo // 10: the viewer, the history entry and the log of one decision on behalf
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	before, err := f.service.Approval(t.Context(), alex, "A-1004")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}

	approval := f.approve(t, alex, "A-1004", "Daniel is on leave.")

	danielView := PersonView{ID: daniel.ID, Name: "Daniel Okoro", Initials: "DO", Active: true}
	if before.Viewer.IsCurrentApprover || !reflect.DeepEqual(before.Viewer.OnBehalfOf, []PersonView{danielView}) {
		t.Errorf("viewer = %+v, want Alex deciding on behalf of Daniel Okoro", before.Viewer)
	}
	entry := approval.Rounds[0].History[0]
	if entry.Actor != alex || !reflect.DeepEqual(entry.OnBehalfOf, []actor.Actor{daniel}) || entry.Action != "approved" ||
		entry.LevelLabel != "Level 1" || !reflect.DeepEqual(entry.Approvers, []PersonView{danielView}) {
		t.Errorf("history entry = %+v, want Alex Morgan on behalf of Daniel Okoro at level 1", entry)
	}
	if entries := f.log(t, "A-1004"); logLine(entries[0]) != "Alex Morgan approved activity 'Trade magazine ads' [submitted → approved] "+
		`"Daniel is on leave." (Round 1 · level 1 of 1; on behalf of Daniel Okoro)` {
		t.Errorf("latest log entry = %s", logLine(entries[0]))
	}
}

// In progress and Complete are displayed statuses of an approved activity: time passing changes them
// without any action, so nobody is ever notified of them (FR-019).
func TestDisplayStatusesNeverNotify_FR019(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	f.approve(t, daniel, "A-1004", "")
	told := len(f.notifications(t, priya.ID)) + len(f.notifications(t, daniel.ID))
	facts := f.activities.facts["A-1004"]
	facts.DisplayStatus = "complete" // the dates have passed
	f.activities.facts["A-1004"] = facts
	f.clock.Advance(200 * 24 * time.Hour)

	approval, err := f.service.Approval(t.Context(), priya, "A-1004")

	if err != nil || approval.Status != StatusApproved || approval.DisplayStatus != "complete" {
		t.Fatalf("approval = %s / %s (%v), want approved, shown complete", approval.Status, approval.DisplayStatus, err)
	}
	if now := len(f.notifications(t, priya.ID)) + len(f.notifications(t, daniel.ID)); now != told {
		t.Errorf("%d notifications, want still %d", now, told)
	}
}

// Every approval action is in the round's history and the activity log with its round, level, designated
// approvers, actor and "on behalf of" (FR-026, SC-004).
func TestApprovalActions_Logged_FR026_SC004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "First try.")
	f.approve(t, alex, "A-1009", "")                   // level 1, on behalf of Carlos Mendes
	f.decline(t, alex, "A-1009", "Split the cost.")    // level 2, Alex's own level
	f.submit(t, mei, "A-1009", "Split.")               // round 2
	f.approve(t, carlos, "A-1009", "")                 // level 1
	f.approve(t, sara, "A-1009", "Approved for Alex.") // level 2, on behalf of Alex Morgan

	var got []string
	for _, e := range f.actions(t, "A-1009") {
		line := fmt.Sprintf("round %d level %d (%s) %s by %s", e.RoundNumber, e.Level, names(e.Approvers), e.Action, e.Actor.Name)
		if len(e.OnBehalfOf) > 0 {
			line += " on behalf of " + names(e.OnBehalfOf)
		}
		got = append(got, line)
	}
	want := []string{
		"round 1 level 1 (Carlos Mendes) submitted by Mei Lin",
		"round 1 level 1 (Carlos Mendes) approved by Alex Morgan on behalf of Carlos Mendes",
		"round 1 level 2 (Alex Morgan) declined by Alex Morgan",
		"round 2 level 1 (Carlos Mendes) resubmitted by Mei Lin",
		"round 2 level 1 (Carlos Mendes) approved by Carlos Mendes",
		"round 2 level 2 (Alex Morgan) approved by Sara Novak on behalf of Alex Morgan",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("history =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	assertLog(t, f, "A-1009", []string{
		`Sara Novak approved activity 'Fleet card launch' [submitted → approved] "Approved for Alex." (Round 2 · level 2 of 2; on behalf of Alex Morgan)`,
		"Carlos Mendes approved level 1 of activity 'Fleet card launch' (Round 2 · level 1 of 2 · now waiting for level 2 (Alex Morgan))",
		`Mei Lin re-submitted activity 'Fleet card launch' [declined → submitted] "Split." (Round 2 · China BC approval workflow · waiting for level 1 (Carlos Mendes))`,
		`Alex Morgan declined activity 'Fleet card launch' [submitted → declined] "Split the cost." (Round 1 · level 2 of 2)`,
		"Alex Morgan approved level 1 of activity 'Fleet card launch' (Round 1 · level 1 of 2 · now waiting for level 2 (Alex Morgan); on behalf of Carlos Mendes)",
		`Mei Lin submitted activity 'Fleet card launch' [created → submitted] "First try." (Round 1 · China BC approval workflow · waiting for level 1 (Carlos Mendes))`,
	})
}
