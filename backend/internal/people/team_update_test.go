package people

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

// A team may be renamed to a name no other team has, in any letter case, and keeps its own name in other
// letters; the log shows the old and new names (FR-011, US2 scenario 4).
func TestUpdateTeam_RenameUnique_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true)
	f.addTeam(t, "Mobility China", "violet", true)
	rename := func(name string, from int) (TeamView, error) {
		return f.service.UpdateTeam(t.Context(), alex.Actor(), europe.ID,
			TeamUpdate{Name: text(name), Colour: text("blue"), MemberIDs: idList(), Version: version(from)})
	}

	_, err := rename(" MOBILITY china", 1)
	assertError(t, err, errs.Validation(errs.FieldError{Field: "name", Message: "A team called 'Mobility China' already exists."}))
	renamed, err := rename(" Lubricants Europe East ", 1)
	if err != nil || renamed.Name != "Lubricants Europe East" || renamed.Version != 2 {
		t.Fatalf("rename = %+v, %v; want Lubricants Europe East at version 2", renamed, err)
	}
	if again, err := rename("LUBRICANTS EUROPE EAST", 2); err != nil || again.Version != 3 {
		t.Errorf("rename to its own name in capitals = %+v, %v; want version 3", again, err)
	}
	assertLog(t, f, []string{
		"Alex Morgan changed team 'Lubricants Europe East' (T-1001): Name Lubricants Europe → Lubricants Europe East",
		"Alex Morgan changed team 'LUBRICANTS EUROPE EAST' (T-1001): Name Lubricants Europe East → LUBRICANTS EUROPE EAST",
	})
}

// Members who were made inactive may stay in the team; only inactive people who are not members yet are
// refused (FR-011, D-070).
func TestUpdateTeam_KeepsInactiveMember_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	mei := f.addPerson(t, "Mei Lin", "Marketer", false)
	daniel := f.addPerson(t, "Daniel Okoro", "Marketing team leader", true)
	carlos := f.addPerson(t, "Carlos Mendes", "Business champion", false)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, mei)
	update := func(from int, members ...Person) (TeamView, error) {
		var memberIDs []string
		for _, p := range members {
			memberIDs = append(memberIDs, p.ID)
		}
		return f.service.UpdateTeam(t.Context(), alex.Actor(), europe.ID,
			TeamUpdate{Name: text("Lubricants Europe"), Colour: text("blue"), MemberIDs: &memberIDs, Version: version(from)})
	}

	got, err := update(1, mei, daniel)
	if err != nil || !reflect.DeepEqual(got.Members, []PersonRef{daniel.Ref(), mei.Ref()}) {
		t.Errorf("keep Mei, add Daniel = %+v, %v; want Daniel and Mei (inactive)", got.Members, err)
	}
	_, err = update(2, mei, daniel, carlos)
	assertError(t, err, errs.Validation(errs.FieldError{Field: "memberIds",
		Message: "Carlos Mendes is inactive and can't be added. Reactivate them first."}))
}

// A save from an out-of-date copy is refused, naming who changed the team and when, and nothing is
// overwritten (001 FR-029). A missing version is reported with the other problems; an unknown team answers
// not_found.
func TestUpdateTeam_VersionConflict_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true)
	f.clock.Advance(5*time.Hour + 32*time.Minute)
	update := TeamUpdate{Name: text("Daniel's name"), Colour: text("blue"), Version: version(1)}
	if _, err := f.service.UpdateTeam(t.Context(), daniel.Actor(), europe.ID, update); err != nil {
		t.Fatalf("Daniel's update: %v", err)
	}

	update.Name = text("Priya's name")
	_, conflict := f.service.UpdateTeam(t.Context(), priya.Actor(), europe.ID, update)
	_, missing := f.service.UpdateTeam(t.Context(), priya.Actor(), europe.ID, TeamUpdate{Name: text(" "), Colour: text("blue")})
	_, gone := f.service.UpdateTeam(t.Context(), priya.Actor(), "T-9999", update)

	assertError(t, conflict, errs.VersionConflict("team", "Daniel Okoro", start.Add(5*time.Hour+32*time.Minute)))
	assertError(t, missing, errs.Validation(
		errs.FieldError{Field: "name", Message: "Enter a name."},
		errs.FieldError{Field: "version", Message: "Reload the page and try again."},
	))
	assertError(t, gone, errs.NotFound("This team no longer exists — someone may have deleted it."))
	if stored, _ := f.storedTeam(t, europe.ID); stored.Name != "Daniel's name" || stored.Version != 2 {
		t.Errorf("stored team = %+v, want Daniel's name at version 2", stored)
	}
}

// Saving a team form that changes nothing is not a change: no new version, no log entry (FR-028).
func TestUpdateTeam_NoChangeNoVersionNoLog_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)

	got, err := f.service.UpdateTeam(t.Context(), alex.Actor(), europe.ID, TeamUpdate{
		Name: text(" Lubricants Europe "), Colour: text("blue"), MemberIDs: idList(priya.ID, priya.ID), Version: version(1),
	})

	if err != nil || got.Version != 1 || len(got.Members) != 1 {
		t.Errorf("UpdateTeam without a change = %+v, %v; want version 1 with Priya", got, err)
	}
	assertLog(t, f, nil)
}

// An update sends only the fields to change (D-203, D-210): a new colour alone keeps the name, the
// description and every member, and the impact check of it warns about nobody.
func TestUpdateTeam_OnlySentFieldsChange_D210(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Lubricants Europe", OwnerID: priya.ID, TeamIDs: []string{europe.ID}},
	))
	update := TeamUpdate{Colour: text("teal"), Version: version(1)}

	impact, impactErr := f.service.TeamImpact(t.Context(), alex.Actor(), europe.ID, update)
	got, err := f.service.UpdateTeam(t.Context(), alex.Actor(), europe.ID, update)

	if impactErr != nil || len(impact.Warnings) != 0 {
		t.Errorf("TeamImpact = %+v, %v; want no warnings", impact, impactErr)
	}
	if err != nil || got.Version != 2 || got.Name != "Lubricants Europe" || got.Colour != "teal" ||
		len(got.Members) != 1 || got.Members[0].ID != priya.ID {
		t.Fatalf("UpdateTeam = %+v, %v; want only the colour changed", got, err)
	}
	if memberships := f.memberships(t); !slices.Equal(memberships, []string{europe.ID + "/" + priya.ID}) {
		t.Errorf("memberships = %q, want Priya still a member", memberships)
	}
}

// Members changed on a team show on the person at once (FR-012).
func TestTeamEdit_MembershipShowsOnPerson_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	sara := f.addPerson(t, "Sara Novak", "Local finance advisor", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)

	_, err := f.service.UpdateTeam(t.Context(), alex.Actor(), europe.ID,
		TeamUpdate{Name: text("Lubricants Europe"), Colour: text("blue"), MemberIDs: idList(sara.ID), Version: version(1)})
	if err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}

	saraNow, err := f.service.GetPerson(t.Context(), alex.Actor(), sara.ID)
	if err != nil || !slices.Equal(saraNow.Teams, []TeamRef{{ID: europe.ID, Name: "Lubricants Europe", Colour: "blue", Active: true}}) {
		t.Errorf("Sara's teams = %+v, %v; want Lubricants Europe", saraNow.Teams, err)
	}
	priyaNow, err := f.service.GetPerson(t.Context(), alex.Actor(), priya.ID)
	if err != nil || len(priyaNow.Teams) != 0 {
		t.Errorf("Priya's teams = %+v, %v; want none", priyaNow.Teams, err)
	}
}

// Teams changed on a person show in the teams' members at once (FR-012, US2 scenario 2).
func TestPersonEdit_MembershipShowsOnTeam_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	asia := f.addTeam(t, "Lubricants Asia", "teal", true)

	_, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: text("Priya Raman"), Email: text(priya.Email), JobTitle: text("Marketer"), TeamIDs: idList(asia.ID), Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdatePerson: %v", err)
	}

	teams, err := f.service.ListTeams(t.Context(), alex.Actor())
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	members := map[string][]PersonRef{}
	for _, team := range teams {
		members[team.ID] = team.Members
	}
	want := map[string][]PersonRef{asia.ID: {f.storedPerson(t, priya.ID).Ref()}, europe.ID: {}}
	if !reflect.DeepEqual(members, want) {
		t.Errorf("members = %+v, want Priya in Lubricants Asia only", members)
	}
}
