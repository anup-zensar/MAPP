package people

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

// The sign-in page lists only the people who can sign in, sorted by name (FR-001).
func TestActivePeople_ExcludesInactive_FR001(t *testing.T) {
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
	alex := DemoPerson{ID: "U-1002", Name: "Alex Morgan", Email: "alex.morgan@northstar.demo", JobTitle: "Marketer", Initials: "AM"}
	if got[0] != alex {
		t.Errorf("ActivePeople()[0] = %+v, want %+v", got[0], alex)
	}
}

// Without demo data the list is empty, never nil, so the API answers "people": [] and the sign-in page can
// say what to do (FR-028).
func TestActivePeople_NoPeopleGivesAnEmptyList_FR028(t *testing.T) {
	service, _ := newTestService(t)

	got, err := service.ActivePeople(t.Context())

	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("ActivePeople = %v, %v; want an empty list", got, err)
	}
}

// The signed-in person comes with everything the sidebar and the account menu show (FR-012), and with
// what My profile needs: their version and teams (003 FR-021, FR-026).
func TestCurrentUser_PersonWithPreferences_FR012(t *testing.T) {
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)

	got, err := service.CurrentUser(t.Context(), priya.Actor())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}

	want := CurrentUser{
		ID:          "U-1002",
		Name:        "Priya Raman",
		FirstName:   "Priya",
		Email:       "priya.raman@northstar.demo",
		JobTitle:    "Marketer",
		Initials:    "PR",
		Preferences: PreferencesView{Theme: "system", SidebarCollapsed: false, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY"},
		Version:     1,
		Teams:       []TeamRef{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CurrentUser = %+v, want %+v", got, want)
	}
}

// A signed-in person who no longer exists must sign in again (FR-007).
func TestCurrentUser_PersonGoneMustSignInAgain_FR007(t *testing.T) {
	service, _ := newTestService(t)

	_, err := service.CurrentUser(t.Context(), actor.Actor{ID: "U-9999", Name: "Nobody"})

	assertError(t, err, errs.SignInRequired())
}

// A person who has changed nothing follows the computer's light or dark setting, sees the sidebar expanded,
// amounts in US dollars and dates as "12 Mar 2026" (FR-011, FR-014, D-038; 003 FR-023, FR-024). Sending no
// change saves nothing.
func TestPreferences_Defaults_FR011_FR014(t *testing.T) {
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)
	defaults := PreferencesView{Theme: "system", SidebarCollapsed: false, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY"}

	user, err := service.CurrentUser(t.Context(), priya.Actor())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if user.Preferences != defaults {
		t.Errorf("preferences = %+v, want the defaults %+v", user.Preferences, defaults)
	}

	got, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{})
	if err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	if got != defaults {
		t.Errorf("UpdatePreferences with nothing sent = %+v, want the defaults %+v", got, defaults)
	}
	if _, found := storedPreferences(t, db, priya.ID); found {
		t.Error("a preferences record was saved although nothing changed")
	}
}

// Only the preferences that were sent change; the others stay as they were (FR-011, FR-014).
func TestUpdatePreferences_ChangesOnlySentFields_FR011(t *testing.T) {
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)
	collapsed, dark := true, "dark"

	afterCollapse, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{SidebarCollapsed: &collapsed})
	if err != nil {
		t.Fatalf("UpdatePreferences(sidebarCollapsed): %v", err)
	}
	afterTheme, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{Theme: &dark})
	if err != nil {
		t.Fatalf("UpdatePreferences(theme): %v", err)
	}

	if want := (PreferencesView{Theme: "system", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY"}); afterCollapse != want {
		t.Errorf("after collapsing = %+v, want %+v", afterCollapse, want)
	}
	if want := (PreferencesView{Theme: "dark", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY"}); afterTheme != want {
		t.Errorf("after choosing dark = %+v, want %+v", afterTheme, want)
	}
	stored, _ := storedPreferences(t, db, priya.ID)
	want := Preferences{PersonID: priya.ID, Theme: "dark", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY"}
	if stored != want {
		t.Errorf("stored preferences = %+v, want %+v", stored, want)
	}
}

// Preferences belong to one person: demo people often share a browser (FR-014, D-038).
func TestUpdatePreferences_EachPersonHasTheirOwn_FR014(t *testing.T) {
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	daniel := newPerson("U-1003", "Daniel Okoro", true)
	addPeople(t, db, priya, daniel)
	dark := "dark"

	if _, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{Theme: &dark}); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}

	user, err := service.CurrentUser(t.Context(), daniel.Actor())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if user.Preferences.Theme != "system" {
		t.Errorf("Daniel's theme = %q after Priya chose dark, want system", user.Preferences.Theme)
	}
}

// A theme other than system, light or dark is refused with a message next to the field, and nothing
// changes (FR-014, FR-020).
func TestUpdatePreferences_InvalidTheme_FR014(t *testing.T) {
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)
	collapsed := true

	for _, theme := range []string{"blue", "", "Dark"} {
		_, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{Theme: &theme, SidebarCollapsed: &collapsed})

		assertError(t, err, errs.Validation(errs.FieldError{Field: "theme", Message: "Choose system, light or dark."}))
	}
	if _, found := storedPreferences(t, db, priya.ID); found {
		t.Error("preferences were saved although the theme was refused")
	}
}

// A preferences record saved by feature 001 has no display currency or date format: the person sees the
// defaults, and the record needs no migration (003 FR-024, data model §4).
func TestPreferences_Old001RecordGetsDefaults_FR024(t *testing.T) {
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	old := Preferences{PersonID: priya.ID, Theme: "dark", SidebarCollapsed: true} // as feature 001 saved it
	f.savePreferences(t, old)

	user, err := f.service.CurrentUser(t.Context(), priya.Actor())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	unchanged, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{})
	if err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}

	want := PreferencesView{Theme: "dark", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY"}
	if user.Preferences != want || unchanged != want {
		t.Errorf("preferences = %+v and %+v, want %+v", user.Preferences, unchanged, want)
	}
	if stored, _ := storedPreferences(t, f.db, priya.ID); stored != old {
		t.Errorf("stored preferences = %+v, want feature 001's record %+v unchanged", stored, old)
	}
}

// The display currency is chosen from the active currencies of feature 002 (003 FR-023). A display currency
// set inactive after the person chose it stays theirs, also when they change another preference or send it
// again (spec edge case, D-063); nobody can choose an inactive or unknown currency.
func TestUpdatePreferences_DisplayCurrencyActiveOnly_FR023(t *testing.T) {
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	currencies := []Currency{usd,
		{Code: "EUR", Name: "Euro", Rate: "0.92", Active: true},
		{Code: "GBP", Name: "Pound sterling", Rate: "0.79", Active: false},
	}
	f.service.UseCurrencies(func(context.Context) ([]Currency, error) { return currencies, nil })
	eur, gbp, xyz, dark := "EUR", "GBP", "XYZ", "dark"
	refused := errs.Validation(errs.FieldError{Field: "displayCurrency", Message: "Choose one of the active currencies."})

	chosen, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{DisplayCurrency: &eur})
	if err != nil || chosen.DisplayCurrency != "EUR" {
		t.Fatalf("choose EUR = %+v, %v; want EUR", chosen, err)
	}
	currencies[1].Active = false // Settings sets the euro inactive

	afterTheme, themeErr := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{Theme: &dark})
	sentAgain, againErr := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{DisplayCurrency: &eur})
	_, inactiveOther := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{DisplayCurrency: &gbp})
	_, unknown := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{DisplayCurrency: &xyz})
	_, inactiveForOthers := f.service.UpdatePreferences(t.Context(), daniel.Actor(), PreferencesUpdate{DisplayCurrency: &eur})

	if themeErr != nil || afterTheme.DisplayCurrency != "EUR" || againErr != nil || sentAgain.DisplayCurrency != "EUR" {
		t.Errorf("Priya's inactive EUR = %+v, %v and %+v, %v; want it kept", afterTheme, themeErr, sentAgain, againErr)
	}
	assertError(t, inactiveOther, refused)
	assertError(t, unknown, refused)
	assertError(t, inactiveForOthers, refused)
	if stored, _ := storedPreferences(t, f.db, priya.ID); stored.DisplayCurrency != "EUR" {
		t.Errorf("Priya's stored display currency = %q, want EUR", stored.DisplayCurrency)
	}
	if _, found := storedPreferences(t, f.db, daniel.ID); found {
		t.Error("Daniel's refused choice was saved")
	}
}

// The date format is one of DD MMM YYYY, DD/MM/YYYY, MM/DD/YYYY and YYYY-MM-DD (003 FR-024); anything else
// is refused next to the field, together with every other problem of the request (SC-007).
func TestUpdatePreferences_DateFormatChoices_FR024(t *testing.T) {
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)

	for _, format := range []string{"DD/MM/YYYY", "MM/DD/YYYY", "YYYY-MM-DD", "DD MMM YYYY"} {
		got, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{DateFormat: &format})
		if err != nil || got.DateFormat != format {
			t.Errorf("choose %q = %+v, %v", format, got, err)
		}
	}
	for _, format := range []string{"DD.MM.YYYY", "dd/mm/yyyy", ""} {
		_, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{DateFormat: &format})
		assertError(t, err, errs.Validation(errs.FieldError{Field: "dateFormat", Message: "Choose one of the date formats."}))
	}

	blue, xyz, wrong := "blue", "XYZ", "DD.MM.YYYY"
	_, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{Theme: &blue, DisplayCurrency: &xyz, DateFormat: &wrong})

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "theme", Message: "Choose system, light or dark."},
		errs.FieldError{Field: "displayCurrency", Message: "Choose one of the active currencies."},
		errs.FieldError{Field: "dateFormat", Message: "Choose one of the date formats."},
	))
	if stored, _ := storedPreferences(t, f.db, priya.ID); stored.DateFormat != "DD MMM YYYY" {
		t.Errorf("stored date format = %q, want the last valid choice DD MMM YYYY", stored.DateFormat)
	}
}

// Preferences are not business records: changing the display currency or the date format writes nothing
// to the activity log (D-038, D-062; 003 data model §6).
func TestUpdatePreferences_NotLogged_D038(t *testing.T) {
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	f.service.UseCurrencies(func(context.Context) ([]Currency, error) {
		return []Currency{usd, {Code: "EUR", Name: "Euro", Rate: "0.92", Active: true}}, nil
	})
	eur, format := "EUR", "DD/MM/YYYY"

	if _, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{DisplayCurrency: &eur, DateFormat: &format}); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}

	if entries := f.entries(t); len(entries) != 0 {
		t.Errorf("log = %q, want no entry", sentences(entries))
	}
}

// The signed-in person comes with their version and their teams, active or not, by name: the "My teams"
// of other features and My profile (003 FR-026, FR-021).
func TestCurrentUser_VersionAndTeams_FR026(t *testing.T) {
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	f.addTeam(t, "Mobility China", "violet", false, priya)
	f.addTeam(t, "Global Marketing Ops", "teal", true, alex)
	f.addTeam(t, "lubricants Europe", "blue", true, priya)
	_, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: priya.Name, Email: priya.Email, JobTitle: "Senior marketer", TeamIDs: []string{"T-1001", "T-1003"}, Version: version(1),
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
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)

	_, problems := f.service.UpdateProfile(t.Context(), priya.Actor(), ProfileUpdate{Name: "   ", JobTitle: strings.Repeat("j", 121)})
	_, err := f.service.UpdatePerson(t.Context(), alex.Actor(), priya.ID, PersonUpdate{
		Name: priya.Name, Email: priya.Email, JobTitle: "Team leader", Version: version(1),
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
	f := newFixture(t)

	_, err := f.service.UpdateProfile(t.Context(), actor.Actor{ID: "U-9999", Name: "Nobody"},
		ProfileUpdate{Name: "Nobody", Version: version(1)})

	assertError(t, err, errs.SignInRequired())
}

// The display currencies offered are the active currencies, sorted by code, plus the person's current one
// when it has been set inactive since (003 FR-023, D-063).
func TestDisplayCurrencies_ActivePlusCurrent_FR023(t *testing.T) {
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	f.service.UseCurrencies(func(context.Context) ([]Currency, error) {
		return []Currency{usd,
			{Code: "GBP", Name: "Pound sterling", Rate: "0.79", Active: true},
			{Code: "EUR", Name: "Euro", Rate: "0.92", Active: false},
			{Code: "CNY", Name: "Chinese yuan", Rate: "7.10", Active: true},
		}, nil
	})
	f.savePreferences(t, Preferences{PersonID: priya.ID, Theme: "system", DisplayCurrency: "EUR"})

	priyas, priyaErr := f.service.DisplayCurrencies(t.Context(), priya.Actor())
	daniels, danielErr := f.service.DisplayCurrencies(t.Context(), daniel.Actor())

	active := []DisplayCurrencyOption{{"CNY", "Chinese yuan", true}, {"GBP", "Pound sterling", true}, {"USD", "US dollar", true}}
	withEUR := []DisplayCurrencyOption{active[0], {"EUR", "Euro", false}, active[1], active[2]}
	if priyaErr != nil || !reflect.DeepEqual(priyas, withEUR) {
		t.Errorf("Priya's choices = %+v, %v; want %+v", priyas, priyaErr, withEUR)
	}
	if danielErr != nil || !reflect.DeepEqual(daniels, active) {
		t.Errorf("Daniel's choices = %+v, %v; want %+v", daniels, danielErr, active)
	}
}

// newTestService returns a people service on a new, empty data folder, and the store behind it.
func newTestService(t *testing.T) (*Service, *store.DB) {
	t.Helper()
	f := newFixture(t)
	return f.service, f.db
}

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// fixture is the people service and the history service it logs to, on a new, empty data folder, with the
// people kinds registered as internal/app registers them, and a clock that moves only when the test says
// so. No hooks are set: tests that need plans or usage counts set fakes with UsePlans and AddPersonUsage.
type fixture struct {
	service *Service
	history *history.Service
	db      *store.DB
	ids     *ids.Generator
	clock   *testClock
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONRepository(), generator, clock.Now)
	service := NewService(db, NewJSONRepository(), generator, hist, clock.Now)
	for _, kind := range service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return &fixture{service: service, history: hist, db: db, ids: generator, clock: clock}
}

// addPerson stores a person the way the seed does — an id from the counter U, made by NorthStar itself,
// no log entry — with the e-mail "first.last@northstar.demo", and returns them.
func (f *fixture) addPerson(t *testing.T, name, jobTitle string, active bool) Person {
	t.Helper()
	var p Person
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		id, err := f.ids.Next(ctx, ids.PrefixPerson)
		if err != nil {
			return err
		}
		p = newPerson(id, name, active)
		p.JobTitle = jobTitle
		return NewJSONRepository().Put(ctx, p)
	})
	if err != nil {
		t.Fatalf("addPerson(%q): %v", name, err)
	}
	return p
}

// addTeam stores a team with its members the way the seed does, and returns it.
func (f *fixture) addTeam(t *testing.T, name, colour string, active bool, members ...Person) Team {
	t.Helper()
	var team Team
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		repo := NewJSONRepository()
		id, err := f.ids.Next(ctx, ids.PrefixTeam)
		if err != nil {
			return err
		}
		team = Team{ID: id, Name: name, Colour: colour, Active: active, Meta: record.NewMeta(actor.System, start)}
		if err := repo.PutTeam(ctx, team); err != nil {
			return err
		}
		for _, p := range members {
			membershipID, err := f.ids.Next(ctx, ids.PrefixTeamMembership)
			if err != nil {
				return err
			}
			m := Membership{ID: membershipID, TeamID: team.ID, PersonID: p.ID, Meta: record.NewMeta(actor.System, start)}
			if err := repo.PutMembership(ctx, m); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("addTeam(%q): %v", name, err)
	}
	return team
}

// read runs fn in a Read of the fixture's store with its repository, and stops the test when it fails.
func (f *fixture) read(t *testing.T, fn func(ctx context.Context, repo Repository) error) {
	t.Helper()
	err := f.db.Read(t.Context(), func(ctx context.Context) error { return fn(ctx, NewJSONRepository()) })
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// storedPerson returns the stored person id and stops the test when there is none.
func (f *fixture) storedPerson(t *testing.T, id string) Person {
	t.Helper()
	var p Person
	var found bool
	f.read(t, func(ctx context.Context, repo Repository) (err error) {
		p, found, err = repo.Get(ctx, id)
		return err
	})
	if !found {
		t.Fatalf("person %s is not stored", id)
	}
	return p
}

// storedTeam returns the stored team id, and false when there is none.
func (f *fixture) storedTeam(t *testing.T, id string) (Team, bool) {
	t.Helper()
	var team Team
	var found bool
	f.read(t, func(ctx context.Context, repo Repository) (err error) {
		team, found, err = repo.GetTeam(ctx, id)
		return err
	})
	return team, found
}

// memberships returns every stored membership as "<team id>/<person id>", ordered by membership id.
func (f *fixture) memberships(t *testing.T) []string {
	t.Helper()
	var list []string
	f.read(t, func(ctx context.Context, repo Repository) error {
		stored, err := repo.ListMemberships(ctx)
		for _, m := range stored {
			list = append(list, m.TeamID+"/"+m.PersonID)
		}
		return err
	})
	return list
}

// entries returns the whole activity log, oldest first.
func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	var entries []history.Entry
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entries, err = history.NewJSONRepository().Entries(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return entries
}

// sentences writes each log entry as a sentence: "<person> <verb> <record type> '<name>' (<id>)", then
// its changes and details, the way feature 011 will show it.
func sentences(entries []history.Entry) []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		text := fmt.Sprintf("%s %s %s '%s' (%s)", e.Actor.Name, e.Verb, e.RecordTypeLabel, e.RecordName, e.RecordID)
		for _, c := range e.Changes {
			text += fmt.Sprintf(": %s %s → %s", c.Label, c.OldText, c.NewText)
		}
		if e.Details != "" {
			text += " — " + e.Details
		}
		list = append(list, text)
	}
	return list
}

// plansHook returns a PlanSource that lists plans.
func plansHook(plans ...PlanUse) PlanSource {
	return func(context.Context) ([]PlanUse, error) { return plans, nil }
}

// version returns a pointer to v, for the Version of updates.
func version(v int) *int {
	return &v
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// newPerson returns a person with the e-mail "first.last@northstar.demo" and the job title Marketer.
func newPerson(id, name string, active bool) Person {
	email := strings.ToLower(strings.ReplaceAll(name, " ", ".")) + "@northstar.demo"
	created := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	return Person{ID: id, Name: name, Email: email, JobTitle: "Marketer", Active: active, Meta: record.NewMeta(actor.System, created)}
}

// addPeople saves people through the repository, in one Write.
func addPeople(t *testing.T, db *store.DB, list ...Person) {
	t.Helper()
	repo := NewJSONRepository()
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for _, p := range list {
			if err := repo.Put(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// storedPreferences returns the preferences record of a person, and false when there is none.
func storedPreferences(t *testing.T, db *store.DB, personID string) (Preferences, bool) {
	t.Helper()
	var prefs Preferences
	var found bool
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		prefs, found, err = NewJSONRepository().GetPreferences(ctx, personID)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return prefs, found
}

// assertError checks that err is exactly want: the same status, code, message and field problems.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v, want %#v", err, want)
	}
}
