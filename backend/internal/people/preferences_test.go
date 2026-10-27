package people

import (
	"context"
	"reflect"
	"testing"

	"mapp/internal/platform/errs"
)

// A person who has changed nothing follows the computer's light or dark setting, sees the sidebar expanded,
// amounts in US dollars and dates as "12 Mar 2026" (FR-011, FR-014, D-038; 003 FR-023, FR-024). Sending no
// change saves nothing.
func TestPreferences_Defaults_FR011_FR014(t *testing.T) {
	t.Parallel()
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)
	defaults := PreferencesView{Theme: "system", SidebarCollapsed: false, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY", HomeScope: "myTeams",
		PinnedPlanIDs: []string{}}

	user, err := service.CurrentUser(t.Context(), priya.Actor())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if !reflect.DeepEqual(user.Preferences, defaults) {
		t.Errorf("preferences = %+v, want the defaults %+v", user.Preferences, defaults)
	}

	got, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{})
	if err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	if !reflect.DeepEqual(got, defaults) {
		t.Errorf("UpdatePreferences with nothing sent = %+v, want the defaults %+v", got, defaults)
	}
	if _, found := storedPreferences(t, db, priya.ID); found {
		t.Error("a preferences record was saved although nothing changed")
	}
}

// Only the preferences that were sent change; the others stay as they were (FR-011, FR-014).
func TestUpdatePreferences_ChangesOnlySentFields_FR011(t *testing.T) {
	t.Parallel()
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

	if want := (PreferencesView{Theme: "system", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY", HomeScope: "myTeams",
		PinnedPlanIDs: []string{}}); !reflect.DeepEqual(afterCollapse, want) {
		t.Errorf("after collapsing = %+v, want %+v", afterCollapse, want)
	}
	if want := (PreferencesView{Theme: "dark", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY", HomeScope: "myTeams",
		PinnedPlanIDs: []string{}}); !reflect.DeepEqual(afterTheme, want) {
		t.Errorf("after choosing dark = %+v, want %+v", afterTheme, want)
	}
	stored, _ := storedPreferences(t, db, priya.ID)
	// The stored record keeps no pinned list while none is pinned ("pinnedPlanIds" is omitted when empty).
	want := Preferences{PersonID: priya.ID, Theme: "dark", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY"}
	if !reflect.DeepEqual(stored, want) {
		t.Errorf("stored preferences = %+v, want %+v", stored, want)
	}
}

// Preferences belong to one person: demo people often share a browser (FR-014, D-038).
func TestUpdatePreferences_EachPersonHasTheirOwn_FR014(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

	want := PreferencesView{Theme: "dark", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY", HomeScope: "myTeams",
		PinnedPlanIDs: []string{}}
	if !reflect.DeepEqual(user.Preferences, want) || !reflect.DeepEqual(unchanged, want) {
		t.Errorf("preferences = %+v and %+v, want %+v", user.Preferences, unchanged, want)
	}
	if stored, _ := storedPreferences(t, f.db, priya.ID); !reflect.DeepEqual(stored, old) {
		t.Errorf("stored preferences = %+v, want feature 001's record %+v unchanged", stored, old)
	}
}

// The display currency is chosen from the active currencies of feature 002 (003 FR-023). A display currency
// set inactive after the person chose it stays theirs, also when they change another preference or send it
// again (spec edge case, D-063); nobody can choose an inactive or unknown currency.
func TestUpdatePreferences_DisplayCurrencyActiveOnly_FR023(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// The display currencies offered are the active currencies, sorted by code, plus the person's current one
// when it has been set inactive since (003 FR-023, D-063).
func TestDisplayCurrencies_ActivePlusCurrent_FR023(t *testing.T) {
	t.Parallel()
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
