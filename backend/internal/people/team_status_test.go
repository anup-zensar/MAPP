package people

import (
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// Set team inactive keeps the team's members and the plans that use it; the log says "deactivated" with the
// status change. Asking again for the same status changes nothing. Both fields of the body are needed
// (FR-014, FR-028).
func TestSetTeamStatus_DeactivateKeepsMembersAndLogs_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	inactive := false

	got, err := f.service.SetTeamStatus(t.Context(), alex.Actor(), europe.ID, StatusUpdate{Active: &inactive, Version: version(1)})
	if err != nil || got.Active || got.Version != 2 || len(got.Members) != 1 {
		t.Fatalf("SetTeamStatus(inactive) = %+v, %v; want inactive at version 2 with Priya", got, err)
	}
	again, err := f.service.SetTeamStatus(t.Context(), alex.Actor(), europe.ID, StatusUpdate{Active: &inactive, Version: version(2)})
	if err != nil || again.Version != 2 {
		t.Errorf("SetTeamStatus(inactive) again = %+v, %v; want it unchanged at version 2", again, err)
	}
	if member, err := f.service.IsMemberOfAny(t.Context(), priya.ID, []string{europe.ID}); err != nil || !member {
		t.Errorf("Priya is a member of the inactive team = %v, %v; want true", member, err)
	}
	assertLog(t, f, []string{"Alex Morgan deactivated team 'Lubricants Europe' (T-1001): Status Active → Inactive"})

	_, missing := f.service.SetTeamStatus(t.Context(), alex.Actor(), europe.ID, StatusUpdate{})
	_, conflict := f.service.SetTeamStatus(t.Context(), alex.Actor(), europe.ID, StatusUpdate{Active: &inactive, Version: version(1)})
	_, gone := f.service.SetTeamStatus(t.Context(), alex.Actor(), "T-9999", StatusUpdate{Active: &inactive, Version: version(1)})
	assertError(t, missing, errs.Validation(
		errs.FieldError{Field: "active", Message: "Choose active or inactive."},
		errs.FieldError{Field: "version", Message: "Reload the page and try again."},
	))
	assertError(t, conflict, errs.VersionConflict("team", "Alex Morgan", start))
	assertError(t, gone, errs.NotFound("This team no longer exists — someone may have deleted it."))
}

// Reactivate makes an inactive team active again, logged as "reactivated" (FR-014).
func TestSetTeamStatus_Reactivate_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	china := f.addTeam(t, "Mobility China", "violet", false)
	active := true

	got, err := f.service.SetTeamStatus(t.Context(), alex.Actor(), china.ID, StatusUpdate{Active: &active, Version: version(1)})

	if err != nil || !got.Active || got.Version != 2 {
		t.Errorf("SetTeamStatus(active) = %+v, %v; want active at version 2", got, err)
	}
	assertLog(t, f, []string{"Alex Morgan reactivated team 'Mobility China' (T-1001): Status Inactive → Active"})
}

// A team that any plan uses, including plans in the recycle bin, is not deleted: the refusal counts the
// plans and suggests setting the team inactive (FR-014).
func TestDeleteTeam_UsedByPlanRefused_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true)
	china := f.addTeam(t, "Mobility China", "violet", true)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Lubricants Europe", TeamIDs: []string{europe.ID}},
		PlanUse{ID: "P-1002", Name: "FY2025 Lubricants Europe", TeamIDs: []string{europe.ID}, InRecycleBin: true},
		PlanUse{ID: "P-1003", Name: "FY2025 Mobility China", TeamIDs: []string{china.ID}, InRecycleBin: true},
	))

	_, usedTwice := f.service.DeleteTeam(t.Context(), alex.Actor(), europe.ID)
	_, usedInBin := f.service.DeleteTeam(t.Context(), alex.Actor(), china.ID)

	assertError(t, usedTwice, errs.InUse("Lubricants Europe is used by 2 plans, so it can't be deleted. Set it inactive instead."))
	assertError(t, usedInBin, errs.InUse("Mobility China is used by 1 plan, so it can't be deleted. Set it inactive instead."))
	for _, id := range []string{europe.ID, china.ID} {
		if _, stored := f.storedTeam(t, id); !stored {
			t.Errorf("team %s was deleted although a plan uses it", id)
		}
	}
	assertLog(t, f, nil)
}

// A team no plan uses goes to the recycle bin together with its memberships, in one entry with one log
// entry (FR-015); a team that is gone answers not_found.
func TestDeleteTeam_UnusedMovesWithMemberships_FR015(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	asia := f.addTeam(t, "Lubricants Asia", "teal", true, priya, daniel)
	f.addTeam(t, "Mobility China", "violet", true, priya)

	entry, err := f.service.DeleteTeam(t.Context(), alex.Actor(), asia.ID)
	if err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}

	wantContents := []history.ContentCount{{RecordType: "teamMembership", Label: "team memberships", Count: 2}}
	if entry.RecordName != "Lubricants Asia" || entry.RecordType != "team" || !slices.Equal(entry.Contents, wantContents) {
		t.Errorf("bin entry = %+v, want Lubricants Asia with 2 team memberships", entry.View())
	}
	if _, stored := f.storedTeam(t, asia.ID); stored {
		t.Error("the team is still stored")
	}
	if got := f.memberships(t); !slices.Equal(got, []string{"T-1002/" + priya.ID}) {
		t.Errorf("memberships = %q, want only Mobility China's", got)
	}
	assertLog(t, f, []string{"Alex Morgan deleted team 'Lubricants Asia' (T-1001) — with 2 team memberships"})
	_, err = f.service.DeleteTeam(t.Context(), alex.Actor(), asia.ID)
	assertError(t, err, errs.NotFound("This team no longer exists — someone may have deleted it."))
}
