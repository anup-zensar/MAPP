package people

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

func TestCreateTeam_SavedWithMembersAndLogged_FR010_FR028(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

// A team name with a tab, a line break or a text-direction character is refused.
func TestCreateTeam_SingleLineName_038_FR055_L11(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	for _, name := range []string{"Mobility\tChina", "Mobility\nChina", "Mobility \u202eChina"} {
		_, err := f.service.CreateTeam(t.Context(), alex.Actor(), TeamInput{Name: name})
		assertError(t, err, errs.Validation(errs.FieldError{Field: "name",
			Message: "Names can't contain tabs, line breaks or text-direction characters."}))
	}
	assertLog(t, f, nil)
}

// Without a colour, a new team gets the palette colour the fewest teams have, the first in palette order
// on a tie; a chosen colour is kept (D-025, D-065).
func TestCreateTeam_ColourLeastUsed_D025(t *testing.T) {
	t.Parallel()
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

func TestListTeams_MembersAndPlans_FR010(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
