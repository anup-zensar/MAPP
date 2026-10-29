package people

import (
	"slices"
	"testing"

	"mapp/internal/platform/errs"
)

// Deactivate keeps the person's memberships, logs the status change, and stops them signing in.
func TestSetPersonStatus_DeactivateKeepsMembershipsAndLogs_FR005_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	inactive := false

	got, err := f.service.SetPersonStatus(t.Context(), alex.Actor(), priya.ID, StatusUpdate{Active: &inactive, Version: version(1)})

	if err != nil || got.Active || got.Version != 2 || !slices.Equal(got.Teams, []TeamRef{europe.Ref()}) {
		t.Fatalf("SetPersonStatus = %+v, %v; want inactive at version 2, still in Lubricants Europe", got, err)
	}
	assertLog(t, f, []string{"Alex Morgan deactivated person 'Priya Raman' (U-1002): Status Active → Inactive"})
	if active, _ := f.service.ActivePeople(t.Context()); len(active) != 1 || active[0].ID != alex.ID {
		t.Errorf("ActivePeople = %+v, want only Alex", active)
	}
	if stored, _ := f.storedTeam(t, europe.ID); stored.Version != 1 {
		t.Errorf("team version = %d, want 1: deactivating keeps memberships", stored.Version)
	}
}

func TestSetPersonStatus_Yourself_NotAllowed_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	inactive := false

	_, err := f.service.SetPersonStatus(t.Context(), alex.Actor(), alex.ID, StatusUpdate{Active: &inactive, Version: version(1)})

	assertError(t, err, errs.NotAllowed("You can't deactivate yourself — ask someone else to do it."))
	if stored := f.storedPerson(t, alex.ID); !stored.Active || stored.Version != 1 {
		t.Errorf("Alex = %+v, want active at version 1", stored)
	}
}

// Deactivating an inactive person changes nothing: no new version, no log entry.
func TestSetPersonStatus_AlreadyInactiveUnchanged_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	mei := f.addPerson(t, "Mei Lin", "Marketer", false)
	inactive := false

	got, err := f.service.SetPersonStatus(t.Context(), alex.Actor(), mei.ID, StatusUpdate{Active: &inactive, Version: version(1)})

	if err != nil || got.Active || got.Version != 1 {
		t.Errorf("SetPersonStatus = %+v, %v; want Mei unchanged", got, err)
	}
	assertLog(t, f, nil)
}

// Reactivate lets the person sign in and be chosen again, still in their teams.
func TestSetPersonStatus_ReactivateMembershipsIntact_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	mei := f.addPerson(t, "Mei Lin", "Marketer", false)
	china := f.addTeam(t, "Mobility China", "violet", true, mei)
	active := true

	got, err := f.service.SetPersonStatus(t.Context(), alex.Actor(), mei.ID, StatusUpdate{Active: &active, Version: version(1)})

	if err != nil || !got.Active || got.Version != 2 || !slices.Equal(got.Teams, []TeamRef{china.Ref()}) {
		t.Errorf("SetPersonStatus = %+v, %v; want active at version 2 in Mobility China", got, err)
	}
	assertLog(t, f, []string{"Alex Morgan reactivated person 'Mei Lin' (U-1002): Status Inactive → Active"})
	if signIn, _ := f.service.ActivePeople(t.Context()); len(signIn) != 2 {
		t.Errorf("ActivePeople = %+v, want Alex and Mei", signIn)
	}
}

// The status action needs both fields of its body, checks the version and answers not_found for an unknown person.
func TestSetPersonStatus_MissingActiveAndVersion(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	inactive := false

	_, missing := f.service.SetPersonStatus(t.Context(), alex.Actor(), priya.ID, StatusUpdate{})
	_, conflict := f.service.SetPersonStatus(t.Context(), alex.Actor(), priya.ID, StatusUpdate{Active: &inactive, Version: version(3)})
	_, gone := f.service.SetPersonStatus(t.Context(), alex.Actor(), "U-9999", StatusUpdate{Active: &inactive, Version: version(1)})

	assertError(t, missing, errs.Validation(
		errs.FieldError{Field: "active", Message: "Choose active or inactive."},
		errs.FieldError{Field: "version", Message: "Reload the page and try again."},
	))
	assertError(t, conflict, errs.VersionConflict("person", "SP&A One", start))
	assertError(t, gone, errs.NotFound("This person doesn't exist."))
}
