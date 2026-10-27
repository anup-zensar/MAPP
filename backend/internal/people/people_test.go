package people

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// The log gets a "created" entry, then a "changed" entry for each of the person's teams.
func TestCreatePerson_SavedActiveAndLogged_FR001_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)

	got, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{
		Name: "Jane Doe", Email: "jane.doe@shell.com", JobTitle: "Marketer", TeamIDs: []string{europe.ID},
	})
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	want := PersonDetails{
		PersonView: PersonView{
			ID: "U-1003", Name: "Jane Doe", Email: "jane.doe@shell.com", JobTitle: "Marketer", Initials: "JD",
			Active: true, Teams: []TeamRef{europe.Ref()}, Meta: record.NewMeta(alex.Actor(), start),
		},
		Usage: []UsageCount{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreatePerson =\n%+v\nwant\n%+v", got, want)
	}
	assertLog(t, f, []string{
		"Alex Morgan created person 'Jane Doe' (U-1003)",
		"Alex Morgan changed team 'Lubricants Europe' (T-1001): Members Priya Raman → Jane Doe, Priya Raman",
	})
	if active, err := f.service.ActivePeople(t.Context()); err != nil || len(active) != 3 {
		t.Errorf("ActivePeople = %+v, %v; want Jane among the 3 people who can sign in (FR-003)", active, err)
	}
}

func TestCreatePerson_AllProblemsAtOnce_FR002_SC007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	longEmail := strings.Repeat("j", 245) + "@shell.com" // 255 characters

	tests := []struct {
		name  string
		input PersonInput
		want  []errs.FieldError
	}{
		{"empty name, e-mail without @", PersonInput{Name: " ", Email: "jane.doe.shell.com"}, []errs.FieldError{
			{Field: "name", Message: "Enter a name."},
			{Field: "email", Message: "Enter a valid e-mail address."},
		}},
		{"too long", PersonInput{Name: strings.Repeat("n", 121), Email: longEmail, JobTitle: strings.Repeat("é", 121)}, []errs.FieldError{
			{Field: "name", Message: "Use 120 characters or fewer."},
			{Field: "email", Message: "Use 254 characters or fewer."},
			{Field: "jobTitle", Message: "Use 120 characters or fewer."},
		}},
		{"no e-mail", PersonInput{Name: "Jane Doe", Email: "  "}, []errs.FieldError{
			{Field: "email", Message: "Enter an e-mail address."},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.CreatePerson(t.Context(), alex.Actor(), tt.input)
			assertError(t, err, errs.Validation(tt.want...))
		})
	}
	for _, email := range []string{"jane@", "@shell.com", "jane doe@shell.com", "jane@@shell.com",
		"jane@shell", "jane@.shell.com", "jane@shell.com."} {
		_, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{Name: "Jane Doe", Email: email})
		assertError(t, err, errs.Validation(errs.FieldError{Field: "email", Message: "Enter a valid e-mail address."}))
	}
	if everyone, _ := f.service.ListPeople(t.Context(), alex.Actor()); len(everyone) != 1 {
		t.Errorf("%d people, want only Alex", len(everyone))
	}
	assertLog(t, f, nil)
}

// The e-mail is the sign-in name: stored trimmed and in lower case; the name and job title are trimmed.
func TestCreatePerson_EmailTrimmedAndLowerCase_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)

	got, err := f.service.CreatePerson(t.Context(), alex.Actor(),
		PersonInput{Name: "  Jane Doe ", Email: "  Jane.Doe@Shell.Com ", JobTitle: " Marketer "})

	if err != nil || got.Name != "Jane Doe" || got.Email != "jane.doe@shell.com" || got.JobTitle != "Marketer" {
		t.Errorf("CreatePerson = %+v, %v; want the trimmed name and title and the lower-case e-mail", got, err)
	}
	if stored := f.storedPerson(t, got.ID); stored.Email != "jane.doe@shell.com" {
		t.Errorf("stored e-mail = %q", stored.Email)
	}
}

// The message names who has the e-mail and, for an inactive person, suggests reactivating them.
func TestCreatePerson_DuplicateEmailNamesHolder_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	f.addPerson(t, "Priya Raman", "Marketer", true)
	f.addPerson(t, "Jane Doe", "Marketer", false)

	_, active := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{Name: "P", Email: " PRIYA.Raman@shell.com"})
	_, inactive := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{Name: "J", Email: "Jane.Doe@shell.com"})

	assertError(t, active, errs.Validation(errs.FieldError{Field: "email",
		Message: "priya.raman@shell.com already belongs to Priya Raman."}))
	assertError(t, inactive, errs.Validation(errs.FieldError{Field: "email",
		Message: "jane.doe@shell.com already belongs to Jane Doe (inactive) — reactivate them instead."}))
}

func TestCreatePerson_WithTeamsShowsOnBothSides_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true)
	china := f.addTeam(t, "Mobility China", "violet", true)

	jane, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{
		Name: "Jane Doe", Email: "jane.doe@shell.com", TeamIDs: []string{china.ID, europe.ID, china.ID},
	})
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	if !slices.Equal(jane.Teams, []TeamRef{europe.Ref(), china.Ref()}) {
		t.Errorf("Jane's teams = %+v, want Lubricants Europe and Mobility China", jane.Teams)
	}
	for _, team := range []Team{europe, china} {
		view, err := f.service.GetTeam(t.Context(), alex.Actor(), team.ID)
		if err != nil || len(view.Members) != 1 || view.Members[0].ID != jane.ID || view.Version != 2 {
			t.Errorf("%s = %+v, %v; want Jane as its member at version 2", team.Name, view, err)
		}
	}
}

func TestCreatePerson_InactiveOrUnknownTeamRefused_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	china := f.addTeam(t, "Mobility China", "violet", false)

	_, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{
		Name: "Jane Doe", Email: "jane.doe@shell.com", TeamIDs: []string{china.ID, "T-9999"},
	})

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "teamIds", Message: "Choose teams from the list."},
		errs.FieldError{Field: "teamIds", Message: "Mobility China is inactive, so people can't be added to it. Reactivate the team first."},
	))
}

// People sort by name ignoring letter case, then by e-mail; their teams sort by name.
func TestListPeople_EveryoneSortedWithTeams_FR001(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	sara := f.addPerson(t, "Sara Novak", "Local finance advisor", true)
	f.addPerson(t, "Priya Raman", "Marketer", false)
	f.addPerson(t, "daniel okoro", "", true)
	mei := f.addPerson(t, "Mei Lin", "Marketer", true)
	otherMei := f.addPerson(t, "Mei Lin", "Business champion", true)
	otherMei.Email = "a.mei.lin@shell.com"
	f.putPerson(t, otherMei)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, sara)
	ops := f.addTeam(t, "global Marketing Ops", "teal", false, sara, mei)

	got, err := f.service.ListPeople(t.Context(), sara.Actor())
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}

	var order []string
	for _, p := range got {
		order = append(order, p.Name+" <"+p.Email+">")
	}
	want := []string{
		"daniel okoro <daniel.okoro@shell.com>",
		"Mei Lin <a.mei.lin@shell.com>",
		"Mei Lin <mei.lin@shell.com>",
		"Priya Raman <priya.raman@shell.com>",
		"Sara Novak <sara.novak@shell.com>",
	}
	if !slices.Equal(order, want) {
		t.Fatalf("people = %q, want %q", order, want)
	}
	wantSara := PersonListItem{PersonView: PersonView{
		ID: sara.ID, Name: "Sara Novak", Email: "sara.novak@shell.com", JobTitle: "Local finance advisor",
		Initials: "SN", Active: true, Meta: sara.Meta,
		Teams: []TeamRef{
			{ID: ops.ID, Name: "global Marketing Ops", Colour: "teal", Active: false},
			{ID: europe.ID, Name: "Lubricants Europe", Colour: "blue", Active: true},
		},
	}}
	if !reflect.DeepEqual(got[4], wantSara) {
		t.Errorf("Sara = %+v\nwant %+v", got[4], wantSara)
	}
	if got[0].Teams == nil || len(got[0].Teams) != 0 {
		t.Errorf("teams of a person in no team = %#v, want an empty list", got[0].Teams)
	}
	if got[3].Active {
		t.Error("Priya is listed as active, want inactive")
	}
}

// Owned plans in the recycle bin are left out; then the hooks' counts in the order added, without zeros.
func TestGetPerson_UsageCountsFromHooks_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	f.service.UsePlans(plansHook(
		PlanUse{ID: "P-1001", Name: "FY2026 Lubricants Europe", OwnerID: priya.ID},
		PlanUse{ID: "P-1002", Name: "FY2027 Lubricants Europe", OwnerID: priya.ID},
		PlanUse{ID: "P-1003", Name: "Old plan", OwnerID: priya.ID, InRecycleBin: true},
		PlanUse{ID: "P-1004", Name: "FY2026 Mobility China", OwnerID: daniel.ID},
	))
	counts := map[string]map[string]int{ // record type → person → count
		"approvalWorkflow": {priya.ID: 1},
		"campaign":         {},
		"activity":         {priya.ID: 5},
	}
	count := func(recordType string) func(context.Context, string) (int, error) {
		return func(_ context.Context, personID string) (int, error) { return counts[recordType][personID], nil }
	}
	f.service.AddPersonUsage(PersonUsage{Role: "approver", RecordType: "approvalWorkflow",
		Singular: "approval workflow", Plural: "approval workflows", Count: count("approvalWorkflow")})
	f.service.AddPersonUsage(PersonUsage{Role: "owner", RecordType: "campaign",
		Singular: "campaign", Plural: "campaigns", Count: count("campaign")})
	f.service.AddPersonUsage(PersonUsage{Role: "owner", RecordType: "activity",
		Singular: "activity", Plural: "activities", Count: count("activity")})

	tests := []struct {
		person Person
		want   []UsageCount
	}{
		{priya, []UsageCount{
			{Role: "owner", RecordType: "plan", Label: "plans", Count: 2},
			{Role: "owner", RecordType: "activity", Label: "activities", Count: 5},
			{Role: "approver", RecordType: "approvalWorkflow", Label: "approval workflow", Count: 1},
		}},
		{daniel, []UsageCount{{Role: "owner", RecordType: "plan", Label: "plan", Count: 1}}},
		{alex, []UsageCount{}},
	}
	for _, tt := range tests {
		got, err := f.service.GetPerson(t.Context(), alex.Actor(), tt.person.ID)
		if err != nil {
			t.Fatalf("GetPerson(%s): %v", tt.person.Name, err)
		}
		if !reflect.DeepEqual(got.Usage, tt.want) {
			t.Errorf("usage of %s = %+v, want %+v", tt.person.Name, got.Usage, tt.want)
		}
		if got.ID != tt.person.ID || got.Name != tt.person.Name {
			t.Errorf("GetPerson(%s) = %+v", tt.person.ID, got.PersonView)
		}
	}
}

// People are never deleted, so an unknown id is a wrong id: not_found with this package's message.
func TestGetPerson_NotFound_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)

	_, err := f.service.GetPerson(t.Context(), alex.Actor(), "U-9999")

	assertError(t, err, errs.NotFound("This person doesn't exist."))
}

func (f *fixture) putPerson(t *testing.T, p Person) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error { return f.store.Put(ctx, p) })
	if err != nil {
		t.Fatalf("Put(%s): %v", p.ID, err)
	}
}
