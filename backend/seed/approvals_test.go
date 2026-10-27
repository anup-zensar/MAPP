package seed_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/platform/date"
)

// The approval rounds of 008 data model §8 match the seeded statuses: approved activities have one approved
// round, Trade magazine ads was declined in round 1 and waits for Daniel Okoro in round 2, Fleet roadshow was
// declined, Fleet card launch waits at level 2 for Alex Morgan after Carlos Mendes approved level 1; Created
// activities have none (FR-037).
func TestLoad_ApprovalRounds_FR037(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	statusOf := map[string]activity.Status{}
	nameOf := map[string]string{}
	for _, a := range d.activities {
		statusOf[a.ID], nameOf[a.ID] = a.Status, a.Name
	}

	var got []string
	last := map[string]approval.Round{} // activity → its last round
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

	outcomeOf := map[activity.Status]approval.Outcome{
		activity.StatusApproved: approval.OutcomeApproved, activity.StatusDeclined: approval.OutcomeDeclined,
		activity.StatusSubmitted: approval.OutcomeOpen,
	}
	for id, status := range statusOf {
		round, has := last[id]
		if status == activity.StatusCreated {
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
	t.Parallel()
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
	t.Parallel()
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

// The demo data of feature 017 (FR-037, data-model §13): line managers; WF-1001 reminds after 1 day and
// escalates after 2 to Alex Morgan; WF-1002 after 3 and 5 with backups Alex Morgan and Carlos Mendes; WF-1003
// "Standard marketing approval" with its three rules; the seeded rounds carry their facts, timings and
// backups, Trade magazine ads waiting since its submission; DLG-1001 Daniel Okoro → Carlos Mendes, planned.
func TestSeed_ApprovalRules_FR037(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	managers := map[string]string{}
	for _, p := range d.people {
		managers[p.Name] = p.ManagerID
	}
	if managers["Priya Raman"] != "U-1003" || managers["Sara Novak"] != "U-1003" || managers["Mei Lin"] != "U-1005" ||
		managers["Daniel Okoro"] != "U-1001" || managers["Carlos Mendes"] != "U-1001" || managers["Alex Morgan"] != "" {
		t.Errorf("line managers = %v", managers)
	}

	var got []string
	for _, w := range d.workflows {
		line := fmt.Sprintf("%s %d/%d", w.ID, *w.ReminderDays, *w.EscalationDays)
		for _, l := range w.Levels {
			line += fmt.Sprintf(" | %s %s %s backup %s", l.ApproverType, l.Mode, strings.Join(l.ApproverIDs, ","), l.BackupApproverID)
			if l.AppliesWhen != nil {
				line += " only when"
			}
			if l.AutoApproveWhen != nil {
				line += " automatic when"
			}
		}
		got = append(got, line)
	}
	want := []string{
		"WF-1001 1/2 | people any U-1003 backup U-1001",
		"WF-1002 3/5 | people any U-1005 backup U-1001 | people any U-1001 backup U-1005",
		"WF-1003 3/5 | manager any  backup U-1001 automatic when | people any U-1001 backup U-1005 only when | people all U-1003,U-1001 backup U-1005 only when",
	}
	if !slices.Equal(got, want) {
		t.Errorf("workflows =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	standard := d.workflows[2]
	if standard.Name != "Standard marketing approval" || standard.Levels[0].AutoApproveWhen.PlannedCostBelow.String() != "10000.00" ||
		standard.Levels[1].AppliesWhen.PlannedCostAtLeast.String() != "100000.00" || d.label(standard.Levels[2].AppliesWhen.BusinessUnitIDs[0]) != "Mobility" {
		t.Errorf("WF-1003 = %+v", standard)
	}

	for _, r := range d.rounds {
		if r.Facts == nil || r.ReminderDays == nil || r.Levels[0].StartedAt == nil || r.Levels[0].Resolution != approval.ResolutionDecide ||
			r.Levels[0].Backup == nil {
			t.Errorf("round %s = %+v, want facts, timings, a start and a backup", r.ID, r)
		}
	}
	trade := d.rounds[3]
	if !trade.Levels[0].StartedAt.Equal(loadedAt.Add(-2*24*time.Hour)) || trade.Facts.PlannedCost == nil ||
		len(trade.Facts.BusinessUnits) != 1 || trade.Facts.BusinessUnits[0].Label != "Lubricants" {
		t.Errorf("Trade magazine ads round 2 = %+v", trade)
	}
	fleet := d.rounds[7]
	if !fleet.Levels[1].StartedAt.Equal(loadedAt.Add(-2*time.Hour)) || fleet.Levels[1].Backup.Name != "Carlos Mendes" {
		t.Errorf("Fleet card launch level 2 = %+v", fleet.Levels[1])
	}

	if len(d.delegations) != 1 {
		t.Fatalf("delegations = %+v, want DLG-1001", d.delegations)
	}
	dl := d.delegations[0]
	today := date.Today(loadedAt)
	if dl.ID != "DLG-1001" || dl.PersonID != "U-1003" || dl.DelegateID != "U-1005" || dl.StartDate != today.AddDays(7) ||
		dl.EndDate != today.AddDays(11) || dl.Note != "Annual leave" || dl.CancelledAt != nil || dl.CreatedBy.Name != "Daniel Okoro" {
		t.Errorf("DLG-1001 = %+v", dl)
	}
}
