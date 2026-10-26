package people

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/errs"
)

// A membership change is a change of both records (D-057): the edited team and every person who joined or
// left get a new version and one "changed" entry; a member who stays is not touched (FR-012, FR-028).
func TestMembershipChange_BothSidesNewVersionAndLog_FR012_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	sara := f.addPerson(t, "Sara Novak", "Local finance advisor", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya, daniel)
	f.addTeam(t, "Mobility China", "violet", true, sara)

	got, err := f.service.UpdateTeam(t.Context(), alex.Actor(), europe.ID, TeamUpdate{
		Name: text("Lubricants Europe"), Colour: text("blue"), MemberIDs: idList(daniel.ID, sara.ID), Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}

	if got.Version != 2 {
		t.Errorf("team version = %d, want 2", got.Version)
	}
	versions := map[string]int{}
	for _, p := range []Person{priya, daniel, sara} {
		versions[p.Name] = f.storedPerson(t, p.ID).Version
	}
	if want := map[string]int{"Priya Raman": 2, "Daniel Okoro": 1, "Sara Novak": 2}; !reflect.DeepEqual(versions, want) {
		t.Errorf("versions = %v, want %v", versions, want)
	}
	assertLog(t, f, []string{
		"Alex Morgan changed team 'Lubricants Europe' (T-1001): Members Daniel Okoro, Priya Raman → Daniel Okoro, Sara Novak",
		"Alex Morgan changed person 'Priya Raman' (U-1002): Teams Lubricants Europe → No team",
		"Alex Morgan changed person 'Sara Novak' (U-1004): Teams Mobility China → Lubricants Europe, Mobility China",
	})
}

// Removing a member who owns plans that use only this team of theirs warns first, with the plans and the
// exact sentence, counted by plan; the check saves nothing (FR-013, US2 scenario 5).
func TestTeamImpact_OwnerLosesLastPlanTeam_Warns_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya, daniel)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Lubricants Europe", OwnerID: priya.ID, TeamIDs: []string{europe.ID}},
		PlanUse{ID: "P-1002", Name: "FY2026 Lubricants Asia", OwnerID: daniel.ID, TeamIDs: []string{europe.ID}},
		PlanUse{ID: "P-1003", Name: "FY2027 Lubricants Europe", OwnerID: daniel.ID, TeamIDs: []string{europe.ID}},
	))
	keepNobody := TeamUpdate{Name: text("Lubricants Europe"), Colour: text("blue"), MemberIDs: idList()}

	got, err := f.service.TeamImpact(t.Context(), alex.Actor(), europe.ID, keepNobody)
	if err != nil {
		t.Fatalf("TeamImpact: %v", err)
	}

	want := []ImpactWarning{
		{Kind: "ownerNeedsAttention", Count: 2, Message: "Daniel Okoro owns 2 plans that use this team: FY2026 Lubricants Asia, " +
			"FY2027 Lubricants Europe. Daniel will no longer belong to their teams, so their owner will need attention."},
		{Kind: "ownerNeedsAttention", Count: 1, Message: "Priya Raman owns 1 plan that uses this team: FY2026 Lubricants Europe. " +
			"Priya will no longer belong to its teams, so its owner will need attention."},
	}
	if !reflect.DeepEqual(got.Warnings, want) {
		t.Errorf("warnings =\n%+v\nwant\n%+v", got.Warnings, want)
	}
	if stored, _ := f.storedTeam(t, europe.ID); stored.Version != 1 || len(f.memberships(t)) != 2 {
		t.Errorf("the impact check changed the team: version %d, memberships %q", stored.Version, f.memberships(t))
	}
	assertLog(t, f, nil)
}

// An owner who still belongs to another of the plan's teams is no concern (R1 still holds).
func TestTeamImpact_OwnerStillInAnotherPlanTeam_NoWarning_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	china := f.addTeam(t, "Mobility China", "violet", true, priya)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Joint plan", OwnerID: priya.ID, TeamIDs: []string{europe.ID, china.ID}},
	))

	got, err := f.service.TeamImpact(t.Context(), alex.Actor(), europe.ID,
		TeamUpdate{Name: text("Lubricants Europe"), Colour: text("blue"), MemberIDs: idList()})

	if err != nil || got.Warnings == nil || len(got.Warnings) != 0 {
		t.Errorf("TeamImpact = %+v, %v; want no warnings", got, err)
	}
}

// Members of an inactive team still count for R1, so an owner who stays in the plan's inactive team needs
// no attention (FR-014).
func TestTeamImpact_InactiveTeamStillCounts_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	china := f.addTeam(t, "Mobility China", "violet", false, priya)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Joint plan", OwnerID: priya.ID, TeamIDs: []string{europe.ID, china.ID}},
	))

	got, err := f.service.TeamImpact(t.Context(), alex.Actor(), europe.ID,
		TeamUpdate{Name: text("Lubricants Europe"), Colour: text("blue"), MemberIDs: idList()})

	if err != nil || len(got.Warnings) != 0 {
		t.Errorf("TeamImpact = %+v, %v; want no warnings", got, err)
	}
}

// Plans in the recycle bin are not part of the owner warning (D-058, D-067).
func TestImpact_IgnoresPlansInRecycleBin_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2025 Lubricants Europe", OwnerID: priya.ID, TeamIDs: []string{europe.ID}, InRecycleBin: true},
	))

	got, err := f.service.TeamImpact(t.Context(), alex.Actor(), europe.ID,
		TeamUpdate{Name: text("Lubricants Europe"), Colour: text("blue"), MemberIDs: idList()})

	if err != nil || len(got.Warnings) != 0 {
		t.Errorf("TeamImpact = %+v, %v; want no warnings", got, err)
	}
}

// The impact check reports the problems of the form exactly as the save would, all at once, but does not
// need the version (SC-007, D-058); an unknown team or person answers not_found.
func TestImpact_ValidatesLikeUpdate_SC007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	mei := f.addPerson(t, "Mei Lin", "", false)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true)
	f.addTeam(t, "Mobility China", "violet", false)

	_, team := f.service.TeamImpact(t.Context(), alex.Actor(), europe.ID, TeamUpdate{
		Name: text("mobility china"), Description: text(strings.Repeat("d", 301)), Colour: text(""), MemberIDs: idList(mei.ID),
	})
	_, person := f.service.PersonImpact(t.Context(), alex.Actor(), mei.ID, PersonUpdate{
		Name: text(""), Email: text("ALEX.MORGAN@shell.com"), TeamIDs: idList("T-1002"),
	})
	_, teamGone := f.service.TeamImpact(t.Context(), alex.Actor(), "T-9999", TeamUpdate{Name: text("X"), Colour: text("blue")})
	_, personGone := f.service.PersonImpact(t.Context(), alex.Actor(), "U-9999", PersonUpdate{Name: text("X"), Email: text("x@y.z")})

	assertError(t, team, errs.Validation(
		errs.FieldError{Field: "name", Message: "A team called 'Mobility China' already exists."},
		errs.FieldError{Field: "description", Message: "Use 300 characters or fewer."},
		errs.FieldError{Field: "colour", Message: "Choose one of the colours."},
		errs.FieldError{Field: "memberIds", Message: "Mei Lin is inactive and can't be added. Reactivate them first."},
	))
	assertError(t, person, errs.Validation(
		errs.FieldError{Field: "name", Message: "Enter a name."},
		errs.FieldError{Field: "email", Message: "alex.morgan@shell.com already belongs to Alex Morgan."},
		errs.FieldError{Field: "teamIds", Message: "Mobility China is inactive, so people can't be added to it. Reactivate the team first."},
	))
	assertError(t, teamGone, errs.NotFound("This team no longer exists — someone may have deleted it."))
	assertError(t, personGone, errs.NotFound("This person doesn't exist."))
}

// A person who leaves the teams of plans they own is warned about first: with one removed team the
// sentence names it, with several it says "the removed teams"; plans that keep one of the person's teams
// are left out, and nothing is saved (FR-013, D-058).
func TestPersonImpact_OneAndTwoRemovedTeams_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	china := f.addTeam(t, "Mobility China", "violet", true, priya)
	ops := f.addTeam(t, "Global Marketing Ops", "teal", true, priya)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Lubricants Europe", OwnerID: priya.ID, TeamIDs: []string{europe.ID}},
		PlanUse{ID: "P-1002", Name: "FY2026 Mobility China", OwnerID: priya.ID, TeamIDs: []string{china.ID}},
		PlanUse{ID: "P-1003", Name: "FY2026 Global Brand", OwnerID: priya.ID, TeamIDs: []string{europe.ID, ops.ID}},
	))
	impact := func(teams ...string) []ImpactWarning {
		t.Helper()
		got, err := f.service.PersonImpact(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
			Name: text("Priya Raman"), Email: text(priya.Email), JobTitle: text("Marketer"), TeamIDs: &teams,
		})
		if err != nil {
			t.Fatalf("PersonImpact: %v", err)
		}
		return got.Warnings
	}

	oneRemoved := impact(china.ID, ops.ID)
	twoRemoved := impact(ops.ID)
	noneRemoved := impact(europe.ID, china.ID, ops.ID)

	if want := []ImpactWarning{{Kind: "ownerNeedsAttention", Count: 1, Message: "Priya Raman owns 1 plan that uses " +
		"Lubricants Europe: FY2026 Lubricants Europe. Priya will no longer belong to its teams, so its owner will need attention."}}; !reflect.DeepEqual(oneRemoved, want) {
		t.Errorf("one team removed: warnings =\n%+v\nwant\n%+v", oneRemoved, want)
	}
	if want := []ImpactWarning{{Kind: "ownerNeedsAttention", Count: 2, Message: "Priya Raman owns 2 plans that use the " +
		"removed teams: FY2026 Lubricants Europe, FY2026 Mobility China. Priya will no longer belong to their teams, so their " +
		"owner will need attention."}}; !reflect.DeepEqual(twoRemoved, want) {
		t.Errorf("two teams removed: warnings =\n%+v\nwant\n%+v", twoRemoved, want)
	}
	if noneRemoved == nil || len(noneRemoved) != 0 {
		t.Errorf("nothing removed: warnings = %#v, want an empty list", noneRemoved)
	}
	if stored := f.storedPerson(t, priya.ID); stored.Version != 1 || len(f.memberships(t)) != 3 {
		t.Errorf("the impact check changed Priya: version %d, memberships %q", stored.Version, f.memberships(t))
	}
}

// Removing a plan's owner from its team is allowed: the warning is only a confirmation (FR-013).
func TestUpdateTeam_RemovingOwnerIsAllowed_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Lubricants Europe", OwnerID: priya.ID, TeamIDs: []string{europe.ID}},
	))

	got, err := f.service.UpdateTeam(t.Context(), alex.Actor(), europe.ID,
		TeamUpdate{Name: text("Lubricants Europe"), Colour: text("blue"), MemberIDs: idList(), Version: version(1)})

	if err != nil || len(got.Members) != 0 || !slices.Equal(got.Plans, []TeamPlanUse{{ID: "P-1001", Name: "FY2026 Lubricants Europe"}}) {
		t.Errorf("UpdateTeam = %+v, %v; want no members, the plan kept", got, err)
	}
}
