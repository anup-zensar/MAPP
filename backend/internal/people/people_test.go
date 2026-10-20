package people

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/platform/errs"
	"northstar/internal/platform/record"
)

// A new person is saved active with the next U- id; the log gets their "created" entry and a "changed"
// entry for each of their teams (FR-001, FR-028, US1 scenario 1).
func TestCreatePerson_SavedActiveAndLogged_FR001_FR028(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)

	got, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{
		Name: "Jane Doe", Email: "jane.doe@northstar.demo", JobTitle: "Marketer", TeamIDs: []string{europe.ID},
	})
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	want := PersonDetails{
		PersonView: PersonView{
			ID: "U-1003", Name: "Jane Doe", Email: "jane.doe@northstar.demo", JobTitle: "Marketer", Initials: "JD",
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

// Every problem of a person form is reported at once (FR-002, SC-007), and nothing is saved.
func TestCreatePerson_AllProblemsAtOnce_FR002_SC007(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	longEmail := strings.Repeat("j", 240) + "@northstar.demo" // 255 characters

	tests := []struct {
		name  string
		input PersonInput
		want  []errs.FieldError
	}{
		{"empty name, e-mail without @", PersonInput{Name: " ", Email: "jane.doe.northstar.demo"}, []errs.FieldError{
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
	for _, email := range []string{"jane@", "@northstar.demo", "jane doe@northstar.demo", "jane@@northstar.demo",
		"jane@northstar", "jane@.northstar.demo", "jane@northstar.demo."} {
		_, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{Name: "Jane Doe", Email: email})
		assertError(t, err, errs.Validation(errs.FieldError{Field: "email", Message: "Enter a valid e-mail address."}))
	}
	if everyone, _ := f.service.ListPeople(t.Context(), alex.Actor()); len(everyone) != 1 {
		t.Errorf("%d people, want only Alex", len(everyone))
	}
	assertLog(t, f, nil)
}

// E-mails are stored without surrounding spaces and in lower case, the sign-in name feature 001 looks up
// (FR-002, spec edge case); the name and job title are trimmed.
func TestCreatePerson_EmailTrimmedAndLowerCase_FR002(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)

	got, err := f.service.CreatePerson(t.Context(), alex.Actor(),
		PersonInput{Name: "  Jane Doe ", Email: "  Jane.Doe@NorthStar.Demo ", JobTitle: " Marketer "})

	if err != nil || got.Name != "Jane Doe" || got.Email != "jane.doe@northstar.demo" || got.JobTitle != "Marketer" {
		t.Errorf("CreatePerson = %+v, %v; want the trimmed name and title and the lower-case e-mail", got, err)
	}
	if stored := f.storedPerson(t, got.ID); stored.Email != "jane.doe@northstar.demo" {
		t.Errorf("stored e-mail = %q", stored.Email)
	}
}

// An e-mail may belong to one person only, active or not, in any letter case; the message names who has it
// and, for an inactive person, suggests reactivating them (FR-002, US1 scenario 2).
func TestCreatePerson_DuplicateEmailNamesHolder_FR002(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	f.addPerson(t, "Priya Raman", "Marketer", true)
	f.addPerson(t, "Jane Doe", "Marketer", false)

	_, active := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{Name: "P", Email: " PRIYA.Raman@northstar.demo"})
	_, inactive := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{Name: "J", Email: "Jane.Doe@northstar.demo"})

	assertError(t, active, errs.Validation(errs.FieldError{Field: "email",
		Message: "priya.raman@northstar.demo already belongs to Priya Raman."}))
	assertError(t, inactive, errs.Validation(errs.FieldError{Field: "email",
		Message: "jane.doe@northstar.demo already belongs to Jane Doe (inactive) — reactivate them instead."}))
}

// A person created with teams shows in those teams' members at once (FR-012).
func TestCreatePerson_WithTeamsShowsOnBothSides_FR012(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true)
	china := f.addTeam(t, "Mobility China", "violet", true)

	jane, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{
		Name: "Jane Doe", Email: "jane.doe@northstar.demo", TeamIDs: []string{china.ID, europe.ID, china.ID},
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

// Only active teams can be added to a person, and only teams that exist (FR-011, D-070).
func TestCreatePerson_InactiveOrUnknownTeamRefused_FR011(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	china := f.addTeam(t, "Mobility China", "violet", false)

	_, err := f.service.CreatePerson(t.Context(), alex.Actor(), PersonInput{
		Name: "Jane Doe", Email: "jane.doe@northstar.demo", TeamIDs: []string{china.ID, "T-9999"},
	})

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "teamIds", Message: "Choose teams from the list."},
		errs.FieldError{Field: "teamIds", Message: "Mobility China is inactive, so people can't be added to it. Reactivate the team first."},
	))
}

// A change is logged with every changed field, old and new values, and the teams joined get their own entry;
// the next sign-in uses the new e-mail (FR-028, US1 scenario 4).
func TestUpdatePerson_ChangesLoggedOldAndNew_FR028(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	china := f.addTeam(t, "Mobility China", "violet", true)

	got, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: text("Priya Raman"), Email: text("priya.r@northstar.demo"), JobTitle: text("Senior marketer"),
		TeamIDs: idList(europe.ID, china.ID), Version: version(1),
	})

	if err != nil || got.Version != 2 || got.Email != "priya.r@northstar.demo" || len(got.Teams) != 2 {
		t.Fatalf("UpdatePerson = %+v, %v; want the new e-mail and two teams at version 2", got, err)
	}
	assertLog(t, f, []string{
		"Alex Morgan changed person 'Priya Raman' (U-1002): E-mail priya.raman@northstar.demo → priya.r@northstar.demo: " +
			"Job title Marketer → Senior marketer: Teams Lubricants Europe → Lubricants Europe, Mobility China",
		"Alex Morgan changed team 'Mobility China' (T-1002): Members No members → Priya Raman",
	})
	f.read(t, func(ctx context.Context, repo Repository) error {
		found, ok, err := repo.FindByEmail(ctx, "priya.r@northstar.demo")
		if !ok || found.ID != priya.ID {
			t.Errorf("FindByEmail(new e-mail) = %+v, %v; want Priya", found, ok)
		}
		return err
	})
}

// An update sends only the fields to change (D-203, D-210): the fields left out keep their values, the
// person keeps their teams, and the log lists only what changed. The impact check of such an update does not
// treat the teams that were not sent as removed.
func TestUpdatePerson_OnlySentFieldsChange_D210(t *testing.T) {
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

// Every update sends the version it started from; without one, or below 1, the save is refused next to
// the field "version", together with the other problems (FR-002, 001 FR-029).
func TestUpdatePerson_MissingVersion_FR002(t *testing.T) {
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

// A save from an out-of-date copy is refused, naming who changed the person and when; nothing is
// overwritten (001 FR-029).
func TestUpdatePerson_VersionConflict_FR029(t *testing.T) {
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

// Saving the form without a change — the same e-mail in other letters, a repeated team — is not a change:
// no new version, no log entry (FR-028).
func TestUpdatePerson_NoChangeNoVersionNoLog_FR028(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, priya)

	got, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: text(" Priya Raman"), Email: text("PRIYA.RAMAN@northstar.demo"), JobTitle: text("Marketer"),
		TeamIDs: idList(europe.ID, europe.ID), Version: version(1),
	})

	if err != nil || got.Version != 1 {
		t.Errorf("UpdatePerson without a change = %+v, %v; want version 1", got, err)
	}
	assertLog(t, f, nil)
}

// A person keeps a team that was set inactive after they joined it; an inactive team they are not in
// cannot be added (FR-014, D-070).
func TestUpdatePerson_KeepsInactiveTeam_FR014(t *testing.T) {
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

// Deactivate keeps the person's memberships, logs "deactivated" with the status change, and removes them
// from the people who can sign in (FR-005, FR-028, US1 scenario 5).
func TestSetPersonStatus_DeactivateKeepsMembershipsAndLogs_FR005_FR028(t *testing.T) {
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

// Nobody can deactivate themselves (FR-006); nothing changes.
func TestSetPersonStatus_Yourself_NotAllowed_FR006(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	inactive := false

	_, err := f.service.SetPersonStatus(t.Context(), alex.Actor(), alex.ID, StatusUpdate{Active: &inactive, Version: version(1)})

	assertError(t, err, errs.NotAllowed("You can't deactivate yourself — ask someone else to do it."))
	if stored := f.storedPerson(t, alex.ID); !stored.Active || stored.Version != 1 {
		t.Errorf("Alex = %+v, want active at version 1", stored)
	}
}

// Deactivating an inactive person changes nothing: no new version, no log entry (FR-005).
func TestSetPersonStatus_AlreadyInactiveUnchanged_FR005(t *testing.T) {
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

// Reactivate lets the person sign in and be chosen again, still in their teams (FR-007, US1 scenario 7).
func TestSetPersonStatus_ReactivateMembershipsIntact_FR007(t *testing.T) {
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

// The status action needs both fields of its body, checks the version and answers not_found for an
// unknown person.
func TestSetPersonStatus_MissingActiveAndVersion(t *testing.T) {
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
	assertError(t, conflict, errs.VersionConflict("person", "NorthStar", start))
	assertError(t, gone, errs.NotFound("This person doesn't exist."))
}

// The People list and every picker show everyone, active or not, with their teams; people are sorted by
// name ignoring letter case, two people with the same name by e-mail; teams by name (FR-001, FR-012).
func TestListPeople_EveryoneSortedWithTeams_FR001(t *testing.T) {
	f := newFixture(t)
	sara := f.addPerson(t, "Sara Novak", "Local finance advisor", true)
	f.addPerson(t, "Priya Raman", "Marketer", false)
	f.addPerson(t, "daniel okoro", "", true)
	mei := f.addPerson(t, "Mei Lin", "Marketer", true)
	otherMei := f.addPerson(t, "Mei Lin", "Business champion", true)
	otherMei.Email = "a.mei.lin@northstar.demo"
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
		"daniel okoro <daniel.okoro@northstar.demo>",
		"Mei Lin <a.mei.lin@northstar.demo>",
		"Mei Lin <mei.lin@northstar.demo>",
		"Priya Raman <priya.raman@northstar.demo>",
		"Sara Novak <sara.novak@northstar.demo>",
	}
	if !slices.Equal(order, want) {
		t.Fatalf("people = %q, want %q", order, want)
	}
	wantSara := PersonListItem{PersonView: PersonView{
		ID: sara.ID, Name: "Sara Novak", Email: "sara.novak@northstar.demo", JobTitle: "Local finance advisor",
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

// A person's details count what they own and approve: the plans they own from feature 004 (plans in the
// recycle bin left out), then the counts of later features in the order they were added, owners before
// approvers; zero counts are left out, and the label is singular for 1 (FR-004, D-067).
func TestGetPerson_UsageCountsFromHooks_FR004(t *testing.T) {
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

// People are never deleted (FR-008), so an unknown id is a wrong id: not_found with this package's message.
func TestGetPerson_NotFound_FR008(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)

	_, err := f.service.GetPerson(t.Context(), alex.Actor(), "U-9999")

	assertError(t, err, errs.NotFound("This person doesn't exist."))
}

// putPerson stores p through the repository, replacing the person with the same id.
func (f *fixture) putPerson(t *testing.T, p Person) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error { return NewJSONRepository().Put(ctx, p) })
	if err != nil {
		t.Fatalf("Put(%s): %v", p.ID, err)
	}
}
