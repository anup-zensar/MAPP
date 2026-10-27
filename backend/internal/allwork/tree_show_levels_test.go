package allwork

import (
	"reflect"
	"testing"
)

// Complete branches only keeps the branches that reach at least one activity (FR-007): the initiative
// without activities goes, and so does a matching initiative whose activities a filter hides.
func TestShowLevels_CompleteBranchesOnly_FR007(t *testing.T) {
	t.Parallel()
	all := view(seedYear(), Query{ShowLevels: ShowCompleteBranches})
	if hasRow(all.Rows, "I-1007") || !hasRow(all.Rows, "I-1006") || all.Count.Shown != 10 {
		t.Errorf("complete branches: Loyalty shown %v, Fleet cards shown %v, count %+v", hasRow(all.Rows, "I-1007"),
			hasRow(all.Rows, "I-1006"), all.Count)
	}
	daniels := view(seedYear(), Query{ShowLevels: ShowCompleteBranches, Owners: []string{daniel.ID}})
	want := []string{"P-1001 context", "C-1002 match", "I-1004 match", "A-1007 match", "A-1007:unassigned match"}
	if got := rolesOf(daniels); !reflect.DeepEqual(got, want) {
		t.Errorf("Daniel's complete branches = %q\nwant %q", got, want)
	}
	if daniels.Summary.Initiatives != 1 {
		t.Errorf("a hidden initiative is not counted: %d initiatives", daniels.Summary.Initiatives)
	}
}

// Campaigns without initiatives lists only campaigns that have no initiative at all, under their plans as
// context, and the count counts campaigns (FR-007, FR-008).
func TestShowLevels_CampaignsWithoutInitiatives_FR007(t *testing.T) {
	t.Parallel()
	b := seedYear()
	b.campaign("P-1002", "C-1004", "Brand awareness", owner(mei))

	r := view(b, Query{ShowLevels: ShowCampaignsWithoutInitiatives})

	if got := rolesOf(r); !reflect.DeepEqual(got, []string{"P-1002 context", "C-1004 match"}) {
		t.Errorf("rows = %q", got)
	}
	if r.Count != (Count{Shown: 1, Total: 4, Noun: "campaigns"}) || r.Summary.Campaigns != 1 || r.Summary.Activities != 0 {
		t.Errorf("count %+v, summary %+v", r.Count, r.Summary)
	}
	// It meets the filters on its own level too: Daniel owns no campaign without initiatives.
	if r := view(b, Query{ShowLevels: ShowCampaignsWithoutInitiatives, Owners: []string{daniel.ID}}); len(r.Rows) != 0 {
		t.Errorf("Daniel's campaigns without initiatives = %q", rowKeys(r.Rows))
	}
}

// Initiatives without activities lists only initiatives with no activity at all, under their plans and
// campaigns as context, and the count counts initiatives (FR-007, FR-008, US2 scenario 4).
func TestShowLevels_InitiativesWithoutActivities_FR007(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{ShowLevels: ShowInitiativesWithoutActivities})

	if got := rolesOf(r); !reflect.DeepEqual(got, []string{"P-1002 context", "C-1003 context", "I-1007 match"}) {
		t.Errorf("rows = %q", got)
	}
	if r.Count != (Count{Shown: 1, Total: 7, Noun: "initiatives"}) || r.Summary.Initiatives != 1 || r.Summary.Campaigns != 0 {
		t.Errorf("count %+v, summary %+v", r.Count, r.Summary)
	}
}
