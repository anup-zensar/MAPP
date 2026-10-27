package people

import (
	"context"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

// The teams joined get their own entry, and the next sign-in uses the new e-mail.
func TestUpdatePerson_ChangesLoggedOldAndNew_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	china := f.addTeam(t, "Mobility China", "violet", true)

	got, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: text("Priya Raman"), Email: text("priya.r@shell.com"), JobTitle: text("Senior marketer"),
		TeamIDs: idList(europe.ID, china.ID), Version: version(1),
	})

	if err != nil || got.Version != 2 || got.Email != "priya.r@shell.com" || len(got.Teams) != 2 {
		t.Fatalf("UpdatePerson = %+v, %v; want the new e-mail and two teams at version 2", got, err)
	}
	assertLog(t, f, []string{
		"Alex Morgan changed person 'Priya Raman' (U-1002): E-mail priya.raman@shell.com → priya.r@shell.com: " +
			"Job title Marketer → Senior marketer: Teams Lubricants Europe → Lubricants Europe, Mobility China",
		"Alex Morgan changed team 'Mobility China' (T-1002): Members No members → Priya Raman",
	})
	f.read(t, func(ctx context.Context, repo fakeStore) error {
		found, ok, err := repo.FindByEmail(ctx, "priya.r@shell.com")
		if !ok || found.ID != priya.ID {
			t.Errorf("FindByEmail(new e-mail) = %+v, %v; want Priya", found, ok)
		}
		return err
	})
}

// The impact check of an update that sends no teams does not treat the person's teams as removed.
func TestUpdatePerson_OnlySentFieldsChange_D210(t *testing.T) { //nolint:gocyclo // 11: one update and its impact check, then the fields, the log and the memberships in turn
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Lubricants Europe", OwnerID: priya.ID, TeamIDs: []string{europe.ID}},
	))
	update := PersonUpdate{JobTitle: text("Senior marketer"), Version: version(1)}

	impact, impactErr := f.service.PersonImpact(t.Context(), alex.Actor(), priya.ID, update)
	got, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, update)

	if impactErr != nil || len(impact.Warnings) != 0 {
		t.Errorf("PersonImpact = %+v, %v; want no warnings", impact, impactErr)
	}
	if err != nil || got.Version != 2 || got.Name != "Priya Raman" || got.Email != priya.Email ||
		got.JobTitle != "Senior marketer" || len(got.Teams) != 1 || got.Teams[0].ID != europe.ID {
		t.Fatalf("UpdatePerson = %+v, %v; want only the job title changed", got, err)
	}
	assertLog(t, f, []string{"Alex Morgan changed person 'Priya Raman' (U-1002): Job title Marketer → Senior marketer"})
	if memberships := f.memberships(t); !slices.Equal(memberships, []string{europe.ID + "/" + priya.ID}) {
		t.Errorf("memberships = %q, want Priya still in Lubricants Europe", memberships)
	}
}

// Without a version, or with one below 1, the save is refused next to "version" with the other problems.
func TestUpdatePerson_MissingVersion_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	reload := errs.FieldError{Field: "version", Message: "Reload the page and try again."}

	_, none := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{Name: text(""), Email: text(priya.Email)})
	_, zero := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{Name: text("Priya"), Email: text(priya.Email), Version: version(0)})
	_, gone := f.service.UpdatePerson(t.Context(), alex.Actor(), "U-9999", PersonUpdate{Name: text("X"), Email: text("x@y.z"), Version: version(1)})

	assertError(t, none, errs.Validation(errs.FieldError{Field: "name", Message: "Enter a name."}, reload))
	assertError(t, zero, errs.Validation(reload))
	assertError(t, gone, errs.NotFound("This person doesn't exist."))
}

func TestUpdatePerson_VersionConflict_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	f.clock.Advance(2 * time.Hour)
	update := PersonUpdate{Name: text("Priya Raman"), Email: text(priya.Email), JobTitle: text("Team leader"), Version: version(1)}
	if _, err := f.service.UpdatePerson(t.Context(), daniel.Actor(), priya.ID, update); err != nil {
		t.Fatalf("Daniel's update: %v", err)
	}

	update.JobTitle = text("Senior marketer")
	_, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, update)

	assertError(t, err, errs.VersionConflict("person", "Daniel Okoro", start.Add(2*time.Hour)))
	if stored := f.storedPerson(t, priya.ID); stored.JobTitle != "Team leader" || stored.Version != 2 {
		t.Errorf("stored person = %+v, want Daniel's change at version 2", stored)
	}
}

// The same e-mail in other letters and a repeated team are not a change.
func TestUpdatePerson_NoChangeNoVersionNoLog_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)

	got, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: text(" Priya Raman"), Email: text("PRIYA.RAMAN@shell.com"), JobTitle: text("Marketer"),
		TeamIDs: idList(europe.ID, europe.ID), Version: version(1),
	})

	if err != nil || got.Version != 1 {
		t.Errorf("UpdatePerson without a change = %+v, %v; want version 1", got, err)
	}
	assertLog(t, f, nil)
}

// An inactive team the person is already in is kept; one they are not in cannot be added.
func TestUpdatePerson_KeepsInactiveTeam_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	china := f.addTeam(t, "Mobility China", "violet", false, priya)
	ops := f.addTeam(t, "Global Marketing Ops", "teal", false)
	update := func(teams ...string) error {
		_, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
			Name: text("Priya Raman"), Email: text(priya.Email), JobTitle: text("Senior marketer"), TeamIDs: &teams, Version: version(1),
		})
		return err
	}

	assertError(t, update(china.ID, ops.ID), errs.Validation(errs.FieldError{Field: "teamIds",
		Message: "Global Marketing Ops is inactive, so people can't be added to it. Reactivate the team first."}))
	if err := update(china.ID); err != nil {
		t.Errorf("keeping the inactive team: %v", err)
	}
}
