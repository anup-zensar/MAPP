package people

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// The sign-in page lists only the people who can sign in, sorted by name (FR-001).
func TestActivePeople_ExcludesInactive_FR001(t *testing.T) {
	t.Parallel()
	service, db := newTestService(t)
	addPeople(t, db,
		newPerson("U-1001", "Sara Novak", true),
		newPerson("U-1002", "Alex Morgan", true),
		newPerson("U-1003", "Mei Lin", false),
		newPerson("U-1004", "Daniel Okoro", true),
	)

	got, err := service.ActivePeople(t.Context())
	if err != nil {
		t.Fatalf("ActivePeople: %v", err)
	}

	var names []string
	for _, p := range got {
		names = append(names, p.Name)
	}
	if want := []string{"Alex Morgan", "Daniel Okoro", "Sara Novak"}; !slices.Equal(names, want) {
		t.Fatalf("ActivePeople names = %q, want %q", names, want)
	}
	alex := DemoPerson{ID: "U-1002", Name: "Alex Morgan", Email: "alex.morgan@shell.com", JobTitle: "Marketer", Initials: "AM"}
	if got[0] != alex {
		t.Errorf("ActivePeople()[0] = %+v, want %+v", got[0], alex)
	}
}

// Without demo data the list is empty, never nil, so the API answers "people": [] and the sign-in page can
// say what to do (FR-028).
func TestActivePeople_NoPeopleGivesAnEmptyList_FR028(t *testing.T) {
	t.Parallel()
	service, _ := newTestService(t)

	got, err := service.ActivePeople(t.Context())

	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("ActivePeople = %v, %v; want an empty list", got, err)
	}
}

// The signed-in person comes with everything the sidebar and the account menu show (FR-012), and with
// what My profile needs: their version and teams (003 FR-021, FR-026).
func TestCurrentUser_PersonWithPreferences_FR012(t *testing.T) {
	t.Parallel()
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)

	got, err := service.CurrentUser(t.Context(), priya.Actor())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}

	want := CurrentUser{
		ID:        "U-1002",
		Name:      "Priya Raman",
		FirstName: "Priya",
		Email:     "priya.raman@shell.com",
		JobTitle:  "Marketer",
		Initials:  "PR",
		Preferences: PreferencesView{Theme: "system", SidebarCollapsed: false, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY",
			HomeScope: "myTeams", PinnedPlanIDs: []string{}},
		Version: 1,
		Teams:   []TeamRef{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CurrentUser = %+v, want %+v", got, want)
	}
}

// A signed-in person who no longer exists must sign in again (FR-007).
func TestCurrentUser_PersonGoneMustSignInAgain_FR007(t *testing.T) {
	t.Parallel()
	service, _ := newTestService(t)

	_, err := service.CurrentUser(t.Context(), actor.Actor{ID: "U-9999", Name: "Nobody"})

	assertError(t, err, errs.SignInRequired())
}

// The signed-in person comes with their version and their teams, active or not, by name: the "My teams"
// of other features and My profile (003 FR-026, FR-021).
func TestCurrentUser_VersionAndTeams_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	f.addTeam(t, "Mobility China", "violet", false, priya)
	f.addTeam(t, "Global Marketing Ops", "teal", true, alex)
	f.addTeam(t, "lubricants Europe", "blue", true, priya)
	_, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: text(priya.Name), Email: text(priya.Email), JobTitle: text("Senior marketer"), TeamIDs: idList("T-1001", "T-1003"), Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdatePerson: %v", err)
	}

	got, err := f.service.CurrentUser(t.Context(), priya.Actor())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}

	wantTeams := []TeamRef{
		{ID: "T-1003", Name: "lubricants Europe", Colour: "blue", Active: true},
		{ID: "T-1001", Name: "Mobility China", Colour: "violet", Active: false},
	}
	if got.Version != 2 || !reflect.DeepEqual(got.Teams, wantTeams) {
		t.Errorf("CurrentUser version %d, teams %+v; want version 2 and %+v", got.Version, got.Teams, wantTeams)
	}
}

// My profile changes the signed-in person's name and job title with the People form's rules: trimmed,
// saved with a new version and one "changed" entry in the log, shown at once in the answer (initials
// included) and wherever the person is named. Saving the same values again writes nothing (003 FR-022,
// FR-028).
func TestUpdateProfile_NameAndTitleLogged_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	f.addTeam(t, "Lubricants Europe", "blue", true, priya)
	f.clock.Advance(time.Hour)

	got, err := f.service.UpdateProfile(t.Context(), priya.Actor(), ProfileUpdate{
		Name: "  Priya Sharma ", JobTitle: "Senior marketer ", Version: version(1),
	})

	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if got.Name != "Priya Sharma" || got.FirstName != "Priya" || got.Initials != "PS" || got.JobTitle != "Senior marketer" ||
		got.Email != priya.Email || got.Version != 2 || len(got.Teams) != 1 {
		t.Errorf("UpdateProfile = %+v; want the new name and job title at version 2, e-mail and team kept", got)
	}
	stored := f.storedPerson(t, priya.ID)
	if stored.Name != "Priya Sharma" || stored.UpdatedAt != start.Add(time.Hour) || stored.UpdatedBy.ID != priya.ID {
		t.Errorf("stored person = %+v, want the change made by Priya an hour later", stored)
	}
	refs, err := f.service.PersonRefs(t.Context(), []string{priya.ID})
	if err != nil || refs[priya.ID].Name != "Priya Sharma" {
		t.Errorf("PersonRefs = %+v, %v; want the new name everywhere she is named", refs, err)
	}

	again, err := f.service.UpdateProfile(t.Context(), priya.Actor(), ProfileUpdate{
		Name: "Priya Sharma", JobTitle: "Senior marketer", Version: version(2),
	})
	if err != nil || again.Version != 2 {
		t.Errorf("saving the same values = %+v, %v; want no new version", again, err)
	}
	assertLog(t, f, []string{
		"Priya Raman changed person 'Priya Sharma' (U-1001): Job title Marketer → Senior marketer: Name Priya Raman → Priya Sharma",
	})
}

// An empty name, a job title over 120 characters and a missing version are reported together, and nothing
// is saved (003 FR-022, SC-007); an out-of-date version is refused with version_conflict.
func TestUpdateProfile_EmptyName_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)

	_, problems := f.service.UpdateProfile(t.Context(), priya.Actor(), ProfileUpdate{Name: "   ", JobTitle: strings.Repeat("j", 121)})
	_, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: text(priya.Name), Email: text(priya.Email), JobTitle: text("Team leader"), Version: version(1),
	})
	if err != nil {
		t.Fatalf("Alex's update: %v", err)
	}
	_, outOfDate := f.service.UpdateProfile(t.Context(), priya.Actor(), ProfileUpdate{Name: "Priya", Version: version(1)})

	assertError(t, problems, errs.Validation(
		errs.FieldError{Field: "name", Message: "Enter a name."},
		errs.FieldError{Field: "jobTitle", Message: "Use 120 characters or fewer."},
		errs.FieldError{Field: "version", Message: "Reload the page and try again."},
	))
	assertError(t, outOfDate, errs.VersionConflict("person", "Alex Morgan", start))
	if stored := f.storedPerson(t, priya.ID); stored.Name != "Priya Raman" || stored.JobTitle != "Team leader" || stored.Version != 2 {
		t.Errorf("stored person = %+v, want only Alex's change", stored)
	}
}

// A signed-in person who no longer exists cannot change their profile: they must sign in again (003
// FR-022, 001 FR-007).
func TestUpdateProfile_PersonGoneMustSignInAgain_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.UpdateProfile(t.Context(), actor.Actor{ID: "U-9999", Name: "Nobody"},
		ProfileUpdate{Name: "Nobody", Version: version(1)})

	assertError(t, err, errs.SignInRequired())
}
