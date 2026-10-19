package people

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
)

// Every change of a person or a team is logged, and a team is deleted with its memberships, so history must
// know the three kinds — the team before its memberships (FR-028, FR-015).
func TestKinds_Register_FR028(t *testing.T) {
	f := newFixture(t)
	hist := history.NewService(f.db, history.NewJSONRepository(), ids.NewGenerator(ids.NewJSONCounters()), f.clock.Now)

	var got []string
	for _, kind := range f.service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
		got = append(got, kind.Name+" ("+kind.Singular+" / "+kind.Plural+") under '"+kind.Parent+"'")
	}

	want := []string{
		"person (person / people) under ''",
		"team (team / teams) under ''",
		"teamMembership (team membership / team memberships) under 'team'",
	}
	if len(got) != len(want) {
		t.Fatalf("kinds = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("kind %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// People are never deleted (FR-008): even history cannot move one to the recycle bin, and nothing changes.
func TestPersonKind_RemoveRefused_FR008(t *testing.T) {
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	alex := f.addPerson(t, "Alex Morgan", "", true)

	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		_, err := f.history.MoveToBin(ctx, alex.Actor(), "person", priya.ID)
		return err
	})

	if !errors.Is(err, errNeverDeleted) {
		t.Errorf("MoveToBin(person) = %v, want %v", err, errNeverDeleted)
	}
	if stored := f.storedPerson(t, priya.ID); stored != priya {
		t.Errorf("stored person = %+v, want %+v unchanged", stored, priya)
	}
	if entries := f.entries(t); len(entries) != 0 {
		t.Errorf("log = %q, want no entry", sentences(entries))
	}
}

// Undo puts a deleted team and its memberships back exactly as they were (FR-015).
func TestTeamDelete_UndoRestoresMembershipsExactly_FR015(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	asia := f.createTeam(t, alex, TeamInput{Name: "Lubricants Asia", MemberIDs: []string{priya.ID, daniel.ID}})
	teamBefore, _ := f.storedTeam(t, asia.ID)
	membershipsBefore := f.storedMemberships(t)
	entry, err := f.service.DeleteTeam(t.Context(), alex.Actor(), asia.ID)
	if err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	f.clock.Advance(time.Minute)

	result, err := f.history.Undo(t.Context(), alex.Actor(), entry.ID)
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}

	if teamAfter, _ := f.storedTeam(t, asia.ID); teamAfter != teamBefore || result.RecordName != "Lubricants Asia" {
		t.Errorf("team after Undo = %+v (%+v), want %+v", teamAfter, result, teamBefore)
	}
	if membershipsAfter := f.storedMemberships(t); !slices.Equal(membershipsAfter, membershipsBefore) {
		t.Errorf("memberships after Undo = %+v, want %+v", membershipsAfter, membershipsBefore)
	}
	priyaNow, err := f.service.GetPerson(t.Context(), alex.Actor(), priya.ID)
	if err != nil || len(priyaNow.Teams) != 1 || priyaNow.Teams[0].ID != asia.ID {
		t.Errorf("Priya's teams after Undo = %+v, %v; want Lubricants Asia", priyaNow.Teams, err)
	}
}

// Undo must put the team back exactly, so it is refused when another team has meanwhile taken the name
// (FR-015, 001 D-037); the team stays in the recycle bin.
func TestTeamUndo_NameTakenRefused_FR015(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	asia := f.createTeam(t, alex, TeamInput{Name: "Lubricants Asia"})
	entry, err := f.service.DeleteTeam(t.Context(), alex.Actor(), asia.ID)
	if err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	f.createTeam(t, alex, TeamInput{Name: "LUBRICANTS ASIA"})

	_, err = f.history.Undo(t.Context(), alex.Actor(), entry.ID)

	assertError(t, err, errs.NameTaken("Can't undo — there is now another team called 'LUBRICANTS ASIA'. "+
		"Restore it from the recycle bin instead."))
	if _, stored := f.storedTeam(t, asia.ID); stored {
		t.Error("the team came back although Undo was refused")
	}
}

// A Restore from the recycle bin renames a team whose name was taken to "<name> (restored)", with a new
// version and a note, and brings its memberships back (FR-015, feature 011 FR-022).
func TestTeamRestore_NameTakenGetsSuffix_FR015(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	asia := f.createTeam(t, alex, TeamInput{Name: "Lubricants Asia", MemberIDs: []string{priya.ID}})
	entry, err := f.service.DeleteTeam(t.Context(), alex.Actor(), asia.ID)
	if err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	f.createTeam(t, alex, TeamInput{Name: "Lubricants Asia"})
	f.clock.Advance(time.Hour)

	result, err := f.history.Restore(t.Context(), alex.Actor(), entry.ID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if result.RecordName != "Lubricants Asia (restored)" ||
		!strings.Contains(result.Details, "Renamed to 'Lubricants Asia (restored)' because the name was taken.") {
		t.Errorf("Restore = %+v, want the renamed team with the note", result)
	}
	restored, _ := f.storedTeam(t, asia.ID)
	if restored.Name != "Lubricants Asia (restored)" || restored.Version != 2 || !restored.UpdatedAt.Equal(start.Add(time.Hour)) {
		t.Errorf("restored team = %+v, want the new name at version 2", restored)
	}
	if got := f.memberships(t); !slices.Equal(got, []string{asia.ID + "/" + priya.ID}) {
		t.Errorf("memberships = %q, want Priya's in the restored team", got)
	}
}

// The log names a team's members, colour and status in words, and a person's teams by name (FR-028, 003
// data model §6).
func TestDescribe_TeamsMembersStatusColour_FR028(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	europe := f.createTeam(t, alex, TeamInput{Name: "Lubricants Europe", Colour: "blue", MemberIDs: []string{priya.ID}})
	_, err := f.service.UpdateTeam(t.Context(), alex.Actor(), europe.ID, TeamUpdate{
		Name: "Lubricants Europe", Description: "Europe", Colour: "teal", MemberIDs: []string{priya.ID, daniel.ID}, Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}
	inactive := false
	if _, err := f.service.SetTeamStatus(t.Context(), alex.Actor(), europe.ID, StatusUpdate{Active: &inactive, Version: version(2)}); err != nil {
		t.Fatalf("SetTeamStatus: %v", err)
	}

	assertLog(t, f, []string{
		"Alex Morgan created team 'Lubricants Europe' (T-1001)",
		"Alex Morgan changed person 'Priya Raman' (U-1002): Teams No team → Lubricants Europe",
		"Alex Morgan changed team 'Lubricants Europe' (T-1001): Colour Blue → Teal: Description Not filled in → Europe: " +
			"Members Priya Raman → Daniel Okoro, Priya Raman",
		"Alex Morgan changed person 'Daniel Okoro' (U-1003): Teams No team → Lubricants Europe",
		"Alex Morgan deactivated team 'Lubricants Europe' (T-1001): Status Active → Inactive",
	})
}

// createTeam creates a team through the service as a and stops the test when that fails.
func (f *fixture) createTeam(t *testing.T, a Person, input TeamInput) TeamView {
	t.Helper()
	team, err := f.service.CreateTeam(t.Context(), a.Actor(), input)
	if err != nil {
		t.Fatalf("CreateTeam(%q): %v", input.Name, err)
	}
	return team
}

// storedMemberships returns every stored membership, ordered by id.
func (f *fixture) storedMemberships(t *testing.T) []Membership {
	t.Helper()
	var list []Membership
	f.read(t, func(ctx context.Context, repo Repository) (err error) {
		list, err = repo.ListMemberships(ctx)
		return err
	})
	return list
}
