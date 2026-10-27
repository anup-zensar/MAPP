package seed_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"mapp/internal/history"
)

// Every approval action 008's seed dates within the 14 days has one "statusChanged" entry on its activity
// with the same actor, time and comment, 008's verb and details and the status change; older actions have
// none (011 plan "Demo data" step 3).
func TestSeed_History_ApprovalActions_011(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	entries, _ := demoLog(t, loadedStore(t))

	var got []string
	for _, e := range entries {
		if e.Action == history.ActionStatusChanged && e.RecordType == "activity" {
			status := ""
			if len(e.Changes) == 1 {
				status = " " + e.Changes[0].OldText + " → " + e.Changes[0].NewText
			}
			got = append(got, fmt.Sprintf("%s %s %s %s (%s)%s | %s | %s", loadedAt.Sub(e.At), e.Actor.Name, e.Verb, e.RecordName,
				e.RecordID, status, e.Details, e.Comment))
		}
	}
	split := "Please split the agency fee by country before re-submitting."
	budget := "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3."
	want := []string{
		"216h0m0s Priya Raman submitted Trade magazine ads (A-1004) Created → Submitted | Round 1 · Lubricants EU approval · waiting for level 1 (Daniel Okoro) | ",
		"192h0m0s Daniel Okoro declined Trade magazine ads (A-1004) Submitted → Declined | Round 1 · level 1 of 1 | " + split,
		"144h0m0s Priya Raman submitted Fleet roadshow (A-1005) Created → Submitted | Round 1 · Lubricants EU approval · waiting for level 1 (Daniel Okoro) | ",
		"120h0m0s Daniel Okoro declined Fleet roadshow (A-1005) Submitted → Declined | Round 1 · level 1 of 1 | " + budget,
		"96h0m0s Mei Lin submitted Fleet card launch (A-1009) Created → Submitted | Round 1 · China BC approval workflow · waiting for level 1 (Carlos Mendes) | ",
		"48h0m0s Priya Raman re-submitted Trade magazine ads (A-1004) Declined → Submitted | Round 2 · Lubricants EU approval · waiting for level 1 (Daniel Okoro) | ",
		"2h0m0s Carlos Mendes approved level 1 of Fleet card launch (A-1009) | Round 1 · level 1 of 2 · now waiting for level 2 (Alex Morgan) | ",
	}
	if !slices.Equal(got, want) {
		t.Errorf("activity status entries:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	inWindow := 0
	for _, a := range d.actions {
		if !a.At.Before(loadedAt.Add(-14 * day)) {
			inWindow++
		}
	}
	if inWindow != len(want) {
		t.Errorf("008 seeds %d approval actions within the 14 days, the log has %d", inWindow, len(want))
	}
}
