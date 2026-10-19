package seed_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/activities"
	"northstar/internal/approvals"
)

// The approval rounds of 008 data model §8 match the seeded statuses: approved activities have one approved
// round, Trade magazine ads was declined in round 1 and waits for Daniel Okoro in round 2, Fleet roadshow was
// declined, Fleet card launch waits at level 2 for Alex Morgan after Carlos Mendes approved level 1; Created
// activities have none (FR-037).
func TestLoad_ApprovalRounds_FR037(t *testing.T) {
	d := loadDemo(t)
	statusOf := map[string]activities.Status{}
	nameOf := map[string]string{}
	for _, a := range d.activities {
		statusOf[a.ID], nameOf[a.ID] = a.Status, a.Name
	}

	var got []string
	last := map[string]approvals.Round{} // activity → its last round
	for _, r := range d.rounds {
		ended := "open"
		if r.EndedAt != nil {
			ended = "ended " + days(r.EndedAt.Sub(loadedAt))
		}
		waiting := r.Levels[r.CurrentLevel-1].Approvers[0].Name
		got = append(got, fmt.Sprintf("%s %s #%d %s %s level %d of %d (%s) submitted %s by %s, %s", r.ID, nameOf[r.ActivityID],
			r.Number, r.WorkflowID, r.Outcome, r.CurrentLevel, len(r.Levels), waiting, days(r.SubmittedAt.Sub(loadedAt)),
			r.SubmittedBy.Name, ended))
		last[r.ActivityID] = r
	}
	want := []string{
		"AR-1001 Social always-on #1 WF-1001 approved level 1 of 1 (Daniel Okoro) submitted -40d by Priya Raman, ended -38d",
		"AR-1002 Billboard winter wave #1 WF-1001 approved level 1 of 1 (Daniel Okoro) submitted -40d by Priya Raman, ended -38d",
		"AR-1003 Trade magazine ads #1 WF-1001 declined level 1 of 1 (Daniel Okoro) submitted -9d by Priya Raman, ended -8d",
		"AR-1004 Trade magazine ads #2 WF-1001 open level 1 of 1 (Daniel Okoro) submitted -2d by Priya Raman, open",
		"AR-1005 Fleet roadshow #1 WF-1001 declined level 1 of 1 (Daniel Okoro) submitted -6d by Priya Raman, ended -5d",
		"AR-1006 Christmas trade gifting #1 WF-1001 approved level 1 of 1 (Daniel Okoro) submitted -40d by Priya Raman, ended -38d",
		"AR-1007 Partner co-marketing fund #1 WF-1001 approved level 1 of 1 (Daniel Okoro) submitted -40d by Priya Raman, ended -38d",
		"AR-1008 Fleet card launch #1 WF-1002 open level 2 of 2 (Alex Morgan) submitted -4d by Mei Lin, open",
	}
	if !slices.Equal(got, want) {
		t.Errorf("rounds =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	outcomeOf := map[activities.Status]approvals.Outcome{
		activities.StatusApproved: approvals.OutcomeApproved, activities.StatusDeclined: approvals.OutcomeDeclined,
		activities.StatusSubmitted: approvals.OutcomeOpen,
	}
	for id, status := range statusOf {
		round, has := last[id]
		if status == activities.StatusCreated {
			if has {
				t.Errorf("%s is Created but has round %s", nameOf[id], round.ID)
			}
			continue
		}
		if !has || round.Outcome != outcomeOf[status] {
			t.Errorf("%s is %s but its last round is %+v", nameOf[id], status, round)
		}
	}
}

// Each round has its history entries, oldest first: the submission, then each decision with its comment
// (008 data model §3).
func TestLoad_ApprovalHistory_FR025(t *testing.T) {
	d := loadDemo(t)
	var got []string
	for _, a := range d.actions {
		got = append(got, fmt.Sprintf("%s %s #%d %s level %d by %s %s %q", a.ID, a.ActivityID, a.RoundNumber, a.Action, a.Level,
			a.Actor.Name, days(a.At.Sub(loadedAt)), a.Comment))
		if len(a.Approvers) == 0 || a.OnBehalfOf == nil || a.Changes == nil {
			t.Errorf("%s: approvers %v, on behalf of %v, changes %v", a.ID, a.Approvers, a.OnBehalfOf, a.Changes)
		}
	}
	want := []string{
		`AH-1001 A-1002 #1 submitted level 1 by Priya Raman -40d ""`,
		`AH-1002 A-1002 #1 approved level 1 by Daniel Okoro -38d ""`,
		`AH-1003 A-1003 #1 submitted level 1 by Priya Raman -40d ""`,
		`AH-1004 A-1003 #1 approved level 1 by Daniel Okoro -38d ""`,
		`AH-1005 A-1004 #1 submitted level 1 by Priya Raman -9d ""`,
		`AH-1006 A-1004 #1 declined level 1 by Daniel Okoro -8d "Please split the agency fee by country before re-submitting."`,
		`AH-1007 A-1004 #2 resubmitted level 1 by Priya Raman -2d ""`,
		`AH-1008 A-1005 #1 submitted level 1 by Priya Raman -6d ""`,
		`AH-1009 A-1005 #1 declined level 1 by Daniel Okoro -5d "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3."`,
		`AH-1010 A-1006 #1 submitted level 1 by Priya Raman -40d ""`,
		`AH-1011 A-1006 #1 approved level 1 by Daniel Okoro -38d ""`,
		`AH-1012 A-1007 #1 submitted level 1 by Priya Raman -40d ""`,
		`AH-1013 A-1007 #1 approved level 1 by Daniel Okoro -38d ""`,
		`AH-1014 A-1009 #1 submitted level 1 by Mei Lin -4d ""`,
		`AH-1015 A-1009 #1 approved level 1 by Carlos Mendes -2h ""`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("history =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The notifications of 008 data model §8: Daniel, Alex and Mei Lin each have one unread; Priya Raman has
// the unread decline of Fleet roadshow and five read ones (FR-033).
func TestLoad_Notifications_FR033(t *testing.T) {
	d := loadDemo(t)
	var got []string
	for _, p := range d.people {
		for _, n := range d.notifications[p.ID] {
			read := "unread"
			if n.Read {
				read = "read"
			}
			got = append(got, fmt.Sprintf("%s: %s %s · %s (%s, %s)", p.Name, n.Kind, n.Title, n.Text, days(n.At.Sub(loadedAt)), read))
			if n.Read != (n.ReadAt != nil) || n.RecipientID == n.Actor.ID {
				t.Errorf("%s: read %v at %v, actor %s", n.ID, n.Read, n.ReadAt, n.Actor.ID)
			}
		}
	}
	want := []string{
		"Alex Morgan: approvalNeeded Approval needed · Carlos Mendes approved level 1 of 'Fleet card launch' (A-1009) — level 2 waits for you (-2h, unread)",
		"Priya Raman: declined Activity declined · Daniel Okoro declined 'Fleet roadshow' (A-1005): " +
			`"The planned cost is above this quarter's budget — please reduce it or move part …" (-5d, unread)`,
		"Priya Raman: declined Activity declined · Daniel Okoro declined 'Trade magazine ads' (A-1004): " +
			`"Please split the agency fee by country before re-submitting." (-8d, read)`,
		"Priya Raman: approved Activity approved · Daniel Okoro approved 'Partner co-marketing fund' (A-1007) (-38d, read)",
		"Priya Raman: approved Activity approved · Daniel Okoro approved 'Christmas trade gifting' (A-1006) (-38d, read)",
		"Priya Raman: approved Activity approved · Daniel Okoro approved 'Billboard winter wave' (A-1003) (-38d, read)",
		"Priya Raman: approved Activity approved · Daniel Okoro approved 'Social always-on' (A-1002) (-38d, read)",
		"Daniel Okoro: approvalNeeded Approval needed · Priya Raman re-submitted 'Trade magazine ads' (A-1004) (-2d, unread)",
		"Mei Lin: levelApproved Level 1 approved · Carlos Mendes approved level 1 of 2 of 'Fleet card launch' (A-1009) (-2h, unread)",
	}
	if !slices.Equal(got, want) {
		t.Errorf("notifications =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// days writes a duration before the seed time in whole days ("-40d"), or hours below one day ("-2h").
func days(d time.Duration) string {
	if d > -24*time.Hour && d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
