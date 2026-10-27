package approval

import (
	"slices"
	"testing"

	"mapp/internal/platform/actor"
)

// The notifications of approval rules (017 FR-032, data-model §8): approval-needed reaches the waiting
// approvers and their current delegates only — not a delegation that starts later or has ended — and never
// the person who acted, even when they are a current delegate; "Approved automatically" reaches the owner
// only when someone else submitted, and the submitter is told nothing. (A partial approval of an "all must
// approve" level telling nobody is TestDecide_AllMustApprove_FR015; reminders and escalations are
// TestRunEscalations_FR026_FR027_FR028_SC005.)
func TestNotifications_RuleEvents_FR032(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addStandardFlow(t)
	f.setManager(t, priya, daniel)
	today := f.today()
	f.delegate(t, "DLG-1001", daniel, carlos, today, today.AddDays(5))            // current
	f.delegate(t, "DLG-1002", daniel, sara, today.AddDays(1), today.AddDays(5))   // planned
	f.delegate(t, "DLG-1003", daniel, alex, today.AddDays(-5), today.AddDays(-1)) // ended
	f.delegate(t, "DLG-1004", daniel, mei, today, today)                          // current, and Mei submits

	// Level 1 (the owner's line manager, Daniel Okoro) needs a decision; levels 2 and 3 are not needed.
	f.addActivity(t, "A-1001", "Launch film", priya.ID, standardFlow)
	f.setFacts(t, "A-1001", "50000.00", lubricantsUnit)
	counts := f.counts(t)
	f.submit(t, mei, "A-1001", "")
	if got, want := f.notifiedSince(t, counts), []string{"Daniel Okoro: approvalNeeded", "Carlos Mendes: approvalNeeded"}; !slices.Equal(got, want) {
		t.Errorf("after the submission: notified %q, want %q", got, want)
	}
	assertNotifiedFirstBy(t, f, carlos, mei.Name,
		"approvalNeeded: Approval needed — Mei Lin submitted 'Launch film' (A-1001) — you approve as delegate of Daniel Okoro")

	// Approved at submission by the owner: nobody is told.
	f.addActivity(t, "A-1002", "Social always-on", priya.ID, standardFlow)
	f.setFacts(t, "A-1002", "5000.00", lubricantsUnit)
	counts = f.counts(t)
	if approval := f.submit(t, priya, "A-1002", ""); approval.Status != StatusApproved {
		t.Fatalf("A-1002 is %s, want approved at submission", approval.Status)
	}
	if got := f.notifiedSince(t, counts); len(got) != 0 {
		t.Errorf("the owner's own automatic approval notified %q, want nobody", got)
	}

	// Approved at submission, submitted by someone else: only the owner is told, by MAPP.
	f.addActivity(t, "A-1003", "Billboard winter wave", priya.ID, standardFlow)
	f.setFacts(t, "A-1003", "5000.00", lubricantsUnit)
	counts = f.counts(t)
	f.submit(t, mei, "A-1003", "")
	if got, want := f.notifiedSince(t, counts), []string{"Priya Raman: autoApproved"}; !slices.Equal(got, want) {
		t.Errorf("after the automatic approval: notified %q, want %q", got, want)
	}
	assertNotifiedFirst(t, f, priya, "autoApproved: Activity approved automatically — 'Billboard winter wave' (A-1003) was approved "+
		"automatically: planned cost is below $10,000.00")
}

// assertNotifiedFirstBy checks the newest notification of p, written as "<kind>: <title> — <text>", and that
// the person named actorName caused it.
func assertNotifiedFirstBy(t *testing.T, f *fixture, p actor.Actor, actorName, want string) {
	t.Helper()
	list := f.notifications(t, p.ID)
	if len(list) == 0 {
		t.Fatalf("%s has no notification", p.Name)
	}
	if got := string(list[0].Kind) + ": " + list[0].Title + " — " + list[0].Text; got != want || list[0].Actor.Name != actorName {
		t.Errorf("%s's newest notification = %q by %s, want %q by %s", p.Name, got, list[0].Actor.Name, want, actorName)
	}
}
