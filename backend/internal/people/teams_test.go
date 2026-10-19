package people

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/record"
)

// A new team is saved active, with its name and description trimmed, its members (a repeated id counts
// once) and the least-used colour; the log gets the team's "created" entry and a "changed" entry for each
// member, whose teams changed (FR-010, FR-012, FR-028, D-057).
func TestCreateTeam_SavedWithMembersAndLogged_FR010_FR028(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "Marketing administrator", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "Marketing team leader", true)

	got, err := f.service.CreateTeam(t.Context(), alex.Actor(), TeamInput{
		Name: "  Lubricants Asia ", Description: " Marketing for lubricants across Asia. ",
		MemberIDs: []string{priya.ID, daniel.ID, priya.ID},
	})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	want := TeamView{
		ID: "T-1001", Name: "Lubricants Asia", Description: "Marketing for lubricants across Asia.", Colour: "blue",
		Active: true, Members: []PersonRef{daniel.Ref(), priya.Ref()}, Plans: []TeamPlanUse{},
		Meta: record.NewMeta(alex.Actor(), start),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateTeam =\n%+v\nwant\n%+v", got, want)
	}
	if got := f.memberships(t); !slices.Equal(got, []string{"T-1001/" + priya.ID, "T-1001/" + daniel.ID}) {
		t.Errorf("memberships = %q, want Priya's and Daniel's", got)
	}
	assertLog(t, f, []string{
		"Alex Morgan created team 'Lubricants Asia' (T-1001)",
		"Alex Morgan changed person 'Priya Raman' (U-1002): Teams No team → Lubricants Asia",
		"Alex Morgan changed person 'Daniel Okoro' (U-1003): Teams No team → Lubricants Asia",
	})
	if stored := f.storedPerson(t, priya.ID); stored.Version != 2 || stored.UpdatedBy != alex.Actor() {
		t.Errorf("Priya = version %d by %v, want version 2 by Alex", stored.Version, stored.UpdatedBy)
	}
}

// Every problem of a team form is reported at once (FR-011, SC-007): a name taken by another team in any
// letter case, a description over 300 characters, a colour outside the palette, an id that is no person and
// an inactive person. Nothing is saved or logged.
func TestCreateTeam_AllProblemsAtOnce_FR011(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	mei := f.addPerson(t, "Mei Lin", "Marketer", false)
	f.addTeam(t, "Mobility China", "violet", true)

	_, err := f.service.CreateTeam(t.Context(), alex.Actor(), TeamInput{
		Name: " mobility CHINA ", Description: strings.Repeat("d", 301), Colour: "pink",
		MemberIDs: []string{mei.ID, "U-9999"},
	})

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "name", Message: "A team called 'Mobility China' already exists."},
		errs.FieldError{Field: "description", Message: "Use 300 characters or fewer."},
		errs.FieldError{Field: "colour", Message: "Choose one of the colours."},
		errs.FieldError{Field: "memberIds", Message: "Choose people from the list."},
		errs.FieldError{Field: "memberIds", Message: "Mei Lin is inactive and can't be added. Reactivate them first."},
	))
	for _, name := range []string{"", "   "} {
		_, err := f.service.CreateTeam(t.Context(), alex.Actor(), TeamInput{Name: name})
		assertError(t, err, errs.Validation(errs.FieldError{Field: "name", Message: "Enter a name."}))
	}
	_, err = f.service.CreateTeam(t.Context(), alex.Actor(), TeamInput{Name: strings.Repeat("é", 81)})
	assertError(t, err, errs.Validation(errs.FieldError{Field: "name", Message: "Use 80 characters or fewer."}))
	if _, created := f.storedTeam(t, "T-1002"); created {
		t.Error("a team was saved although the form was refused")
	}
	assertLog(t, f, nil)
}

// Without a colour, a new team gets the palette colour the fewest teams have, the first in palette order
// on a tie; a chosen colour is kept (D-025, D-065).
func TestCreateTeam_ColourLeastUsed_D025(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	f.addTeam(t, "One", "blue", true)
	f.addTeam(t, "Two", "violet", false)
	f.addTeam(t, "Three", "blue", true)

	var colours []string
	for _, input := range []TeamInput{{Name: "Four"}, {Name: "Five", Colour: "gold"}, {Name: "Six"}} {
		team, err := f.service.CreateTeam(t.Context(), alex.Actor(), input)
		if err != nil {
			t.Fatalf("CreateTeam(%s): %v", input.Name, err)
		}
		colours = append(colours, team.Colour)
	}

	if want := []string{"teal", "gold", "magenta"}; !slices.Equal(colours, want) {
		t.Errorf("colours = %q, want %q", colours, want)
	}
}

// A team may be renamed to a name no other team has, in any letter case, and keeps its own name in other
// letters; the log shows the old and new names (FR-011, US2 scenario 4).
func TestUpdateTeam_RenameUnique_FR011(t *testing.T) {
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

// Set team inactive keeps the team's members and the plans that use it; the log says "deactivated" with the
// status change. Asking again for the same status changes nothing. Both fields of the body are needed
// (FR-014, FR-028).
func TestSetTeamStatus_DeactivateKeepsMembersAndLogs_FR014(t *testing.T) {
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

// assertLog checks the whole activity log, written as sentences.
func assertLog(t *testing.T, f *fixture, want []string) {
	t.Helper()
	if got := sentences(f.entries(t)); !slices.Equal(got, want) {
		t.Errorf("log =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The Teams list shows every team, active or not, sorted by name ignoring letter case, with its members
// (inactive ones included, by name) and every plan that uses it, plans in the recycle bin included and
// marked (FR-010, FR-014, D-067).
func TestListTeams_MembersAndPlans_FR010(t *testing.T) {
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", false)
	daniel := f.addPerson(t, "Daniel Okoro", "Marketing team leader", true)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya, daniel)
	ops := f.addTeam(t, "global Marketing Ops", "teal", false)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1002", Name: "FY2027 Lubricants Europe", OwnerID: daniel.ID, TeamIDs: []string{europe.ID}, InRecycleBin: true},
		PlanUse{ID: "P-1001", Name: "FY2026 Lubricants Europe", OwnerID: priya.ID, TeamIDs: []string{ops.ID, europe.ID}},
		PlanUse{ID: "P-1003", Name: "FY2026 Mobility China", TeamIDs: []string{"T-9999"}},
	))

	got, err := f.service.ListTeams(t.Context(), alex.Actor())
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}

	want := []TeamView{
		{ID: ops.ID, Name: "global Marketing Ops", Colour: "teal", Active: false, Meta: ops.Meta,
			Members: []PersonRef{},
			Plans:   []TeamPlanUse{{ID: "P-1001", Name: "FY2026 Lubricants Europe"}}},
		{ID: europe.ID, Name: "Lubricants Europe", Colour: "blue", Active: true, Meta: europe.Meta,
			Members: []PersonRef{
				{ID: daniel.ID, Name: "Daniel Okoro", Initials: "DO", JobTitle: "Marketing team leader", Active: true},
				{ID: priya.ID, Name: "Priya Raman", Initials: "PR", JobTitle: "Marketer", Active: false},
			},
			Plans: []TeamPlanUse{
				{ID: "P-1001", Name: "FY2026 Lubricants Europe"},
				{ID: "P-1002", Name: "FY2027 Lubricants Europe", InRecycleBin: true},
			}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListTeams =\n%+v\nwant\n%+v", got, want)
	}
}

// Before feature 004 no plan uses a team: every list is empty, never null (FR-010).
func TestListTeams_NoPlansYet_FR010(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	f.addTeam(t, "Lubricants Europe", "blue", true)

	got, err := f.service.ListTeams(t.Context(), alex.Actor())

	if err != nil || len(got) != 1 || got[0].Plans == nil || len(got[0].Plans) != 0 || got[0].Members == nil {
		t.Errorf("ListTeams = %+v, %v; want one team with empty lists", got, err)
	}
}

// A team's details show its members and the plans using it; a team that no longer exists answers
// not_found with the message the web app shows (US2 scenario 3).
func TestGetTeam_DetailsOrNotFound_FR010(t *testing.T) {
	f := newFixture(t)
	sara := f.addPerson(t, "Sara Novak", "Local finance advisor", true)
	china := f.addTeam(t, "Mobility China", "violet", true, sara)

	got, err := f.service.GetTeam(t.Context(), sara.Actor(), china.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.Name != "Mobility China" || len(got.Members) != 1 || got.Members[0].ID != sara.ID {
		t.Errorf("GetTeam = %+v, want Mobility China with Sara", got)
	}
	_, err = f.service.GetTeam(t.Context(), sara.Actor(), "T-9999")
	assertError(t, err, errs.NotFound("This team no longer exists — someone may have deleted it."))
}
