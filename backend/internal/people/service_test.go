package people

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
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

// The signed-in person comes with everything the sidebar and the account menu show (FR-012).
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
		Preferences: PreferencesView{Theme: "system", SidebarCollapsed: false},
	}
	if got != want {
		t.Errorf("CurrentUser = %+v, want %+v", got, want)
	}
}

// A signed-in person who no longer exists must sign in again (FR-007).
func TestCurrentUser_PersonGoneMustSignInAgain_FR007(t *testing.T) {
	service, _ := newTestService(t)

	_, err := service.CurrentUser(t.Context(), actor.Actor{ID: "U-9999", Name: "Nobody"})

	assertError(t, err, errs.SignInRequired())
}

// A person who has changed nothing follows the computer's light or dark setting and sees the sidebar
// expanded (FR-011, FR-014, D-038). Sending no change saves nothing.
func TestPreferences_Defaults_FR011_FR014(t *testing.T) {
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)
	defaults := PreferencesView{Theme: "system", SidebarCollapsed: false}

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

	if want := (PreferencesView{Theme: "system", SidebarCollapsed: true}); afterCollapse != want {
		t.Errorf("after collapsing = %+v, want %+v", afterCollapse, want)
	}
	if want := (PreferencesView{Theme: "dark", SidebarCollapsed: true}); afterTheme != want {
		t.Errorf("after choosing dark = %+v, want %+v", afterTheme, want)
	}
	stored, _ := storedPreferences(t, db, priya.ID)
	if want := (Preferences{PersonID: priya.ID, Theme: "dark", SidebarCollapsed: true}); stored != want {
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

// newTestService returns a people service on a new, empty data folder, and the store behind it.
func newTestService(t *testing.T) (*Service, *store.DB) {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return NewService(db, NewJSONRepository()), db
}

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
