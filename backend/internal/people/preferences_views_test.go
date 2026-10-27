package people

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/platform/errs"
)

// Home's My teams / Everything switch is remembered per person: My teams by default, Everything once chosen,
// back to My teams, which is stored as no value (so choosing it again saves nothing); any other value is
// refused next to the field with the other problems, and nothing changes (009 FR-003, D-172).
func TestUpdatePreferences_HomeScope_FR003(t *testing.T) {
	t.Parallel()
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)
	everything, myTeams := "everything", "myTeams"

	unchanged, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{HomeScope: &myTeams})
	if err != nil || unchanged.HomeScope != "myTeams" {
		t.Fatalf("choosing My teams = %+v, %v; want myTeams", unchanged, err)
	}
	if _, found := storedPreferences(t, db, priya.ID); found {
		t.Error("a preferences record was saved although My teams is the default")
	}

	wide, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{HomeScope: &everything})
	if err != nil || wide.HomeScope != "everything" {
		t.Fatalf("choosing Everything = %+v, %v; want everything", wide, err)
	}
	user, _ := service.CurrentUser(t.Context(), priya.Actor())
	if user.Preferences.HomeScope != "everything" {
		t.Errorf("GET /api/v1/me home scope = %q, want everything (remembered)", user.Preferences.HomeScope)
	}

	back, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{HomeScope: &myTeams})
	stored, _ := storedPreferences(t, db, priya.ID)
	if err != nil || back.HomeScope != "myTeams" || stored.HomeScope != "" {
		t.Errorf("back to My teams = %+v (stored %q), %v; want myTeams stored as no value", back, stored.HomeScope, err)
	}

	for _, scope := range []string{"all", "", "Everything"} {
		theme := "blue"
		_, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{Theme: &theme, HomeScope: &scope})

		assertError(t, err, errs.Validation(
			errs.FieldError{Field: "theme", Message: "Choose system, light or dark."},
			errs.FieldError{Field: "homeScope", Message: "Choose My teams or Everything."}))
	}
	if after, _ := storedPreferences(t, db, priya.ID); !reflect.DeepEqual(after, stored) {
		t.Errorf("stored preferences = %+v after refused changes, want %+v", after, stored)
	}
}

// TestUpdatePreferences_HistoryIncludeInside_FR012: the History section's include choice (feature 011) is
// false by default, remembered for the person, saved only when it changes, and leaves the activity log alone.
func TestUpdatePreferences_HistoryIncludeInside_FR012(t *testing.T) {
	t.Parallel()
	service, db := newTestService(t)
	priya := newPerson("U-1002", "Priya Raman", true)
	addPeople(t, db, priya)
	yes, no := true, false

	user, err := service.CurrentUser(t.Context(), priya.Actor())
	if err != nil || user.Preferences.HistoryIncludeInside {
		t.Fatalf("default = %+v, %v; want false", user.Preferences, err)
	}
	unchanged, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{HistoryIncludeInside: &no})
	if err != nil || unchanged.HistoryIncludeInside {
		t.Fatalf("choosing false = %+v, %v; want false", unchanged, err)
	}
	if _, found := storedPreferences(t, db, priya.ID); found {
		t.Error("a preferences record was saved although nothing changed")
	}

	on, err := service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{HistoryIncludeInside: &yes})
	if err != nil || !on.HistoryIncludeInside {
		t.Fatalf("choosing true = %+v, %v; want true", on, err)
	}
	user, _ = service.CurrentUser(t.Context(), priya.Actor())
	if !user.Preferences.HistoryIncludeInside {
		t.Error("GET /api/v1/me historyIncludeInside = false, want true (remembered)")
	}
	if stored, _ := storedPreferences(t, db, priya.ID); !stored.HistoryIncludeInside || stored.Theme != "system" {
		t.Errorf("stored preferences = %+v, want historyIncludeInside true and the other defaults", stored)
	}
}

// Pinned plans are a display preference like the theme (018 FR-004, D-038, D-317): empty — never null —
// until the person pins, kept per person in the order pinned (unknown ids included: a deleted plan's id is
// harmless, 018 FR-007), shown by GET /api/v1/me from any browser, and never written to the activity log.
func TestUpdatePreferences_PinnedPlans_RoundTrip_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)

	user, err := f.service.CurrentUser(t.Context(), priya.Actor())
	if err != nil || user.Preferences.PinnedPlanIDs == nil || len(user.Preferences.PinnedPlanIDs) != 0 {
		t.Fatalf("default pinned plans = %#v, %v; want an empty list, never nil", user.Preferences.PinnedPlanIDs, err)
	}

	pinned, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{PinnedPlanIDs: idList("P-1002", "P-9999", "P-1001")})
	if err != nil {
		t.Fatalf("UpdatePreferences(pin): %v", err)
	}

	want := []string{"P-1002", "P-9999", "P-1001"} // the order pinned, the unknown P-9999 kept (FR-007)
	if !slices.Equal(pinned.PinnedPlanIDs, want) {
		t.Errorf("pinned = %q, want %q in the order pinned", pinned.PinnedPlanIDs, want)
	}
	user, _ = f.service.CurrentUser(t.Context(), priya.Actor())
	if !slices.Equal(user.Preferences.PinnedPlanIDs, want) {
		t.Errorf("GET /api/v1/me pinned plans = %q, want %q (remembered)", user.Preferences.PinnedPlanIDs, want)
	}
	if daniels, _ := f.service.CurrentUser(t.Context(), daniel.Actor()); len(daniels.Preferences.PinnedPlanIDs) != 0 {
		t.Errorf("Daniel's pinned plans = %q after Priya pinned, want none", daniels.Preferences.PinnedPlanIDs)
	}

	unpinned, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{PinnedPlanIDs: idList()})
	if err != nil || unpinned.PinnedPlanIDs == nil || len(unpinned.PinnedPlanIDs) != 0 {
		t.Errorf("after unpinning all = %#v, %v; want an empty list, never nil", unpinned.PinnedPlanIDs, err)
	}
	if entries := f.entries(t); len(entries) != 0 {
		t.Errorf("log = %q, want no entry (D-038)", sentences(entries))
	}
}

// A repeated id pins the plan once: duplicates are dropped, keeping the order sent (018 FR-005).
func TestUpdatePreferences_PinnedPlans_DuplicatesDropped_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)

	got, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{PinnedPlanIDs: idList("P-1002", "P-1001", "P-1002", "P-1001")})

	if err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	want := []string{"P-1002", "P-1001"}
	if !slices.Equal(got.PinnedPlanIDs, want) {
		t.Errorf("pinned = %q, want %q (each once, order kept)", got.PinnedPlanIDs, want)
	}
	if stored, _ := storedPreferences(t, f.db, priya.ID); !slices.Equal(stored.PinnedPlanIDs, want) {
		t.Errorf("stored pinned plans = %q, want %q", stored.PinnedPlanIDs, want)
	}
}

// Only ids of the plan form P-<number> can be pinned; anything else is refused next to the field, with
// the other problems of the request, and nothing changes (018 FR-005, FR-020 of 001).
func TestUpdatePreferences_PinnedPlans_InvalidIDRefused_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	refused := errs.Validation(errs.FieldError{Field: "pinnedPlanIds", Message: "Pin plans by their id, like P-1001."})

	for _, id := range []string{"p-1001", "C-1001", "P-", "P-12x", ""} {
		_, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{PinnedPlanIDs: idList(id)})

		assertError(t, err, refused)
	}

	blue := "blue"
	_, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{Theme: &blue, PinnedPlanIDs: idList("plan-7")})

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "theme", Message: "Choose system, light or dark."},
		errs.FieldError{Field: "pinnedPlanIds", Message: "Pin plans by their id, like P-1001."},
	))
	if _, found := storedPreferences(t, f.db, priya.ID); found {
		t.Error("preferences were saved although the pinned plans were refused")
	}
}

// At most 100 plans can be pinned; the cap counts the list after duplicates are dropped (018 FR-006).
func TestUpdatePreferences_PinnedPlans_CapAt100_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	ids := make([]string, 0, 101)
	for i := range 101 {
		ids = append(ids, fmt.Sprintf("P-%d", 1000+i))
	}

	_, tooMany := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{PinnedPlanIDs: &ids})

	assertError(t, tooMany, errs.Validation(errs.FieldError{Field: "pinnedPlanIds", Message: "Pin at most 100 plans."}))
	if _, found := storedPreferences(t, f.db, priya.ID); found {
		t.Error("preferences were saved although the cap was refused")
	}

	hundred := slices.Clone(ids[:100])
	full, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{PinnedPlanIDs: &hundred})
	if err != nil || len(full.PinnedPlanIDs) != 100 {
		t.Errorf("pinning 100 plans = %d ids, %v; want all 100 saved", len(full.PinnedPlanIDs), err)
	}

	// 101 sent with one repeat is 100 once deduped, so it is accepted (FR-005 before FR-006).
	withRepeat := append(slices.Clone(ids[:100]), ids[0])
	deduped, err := f.service.UpdatePreferences(t.Context(), priya.Actor(), PreferencesUpdate{PinnedPlanIDs: &withRepeat})
	if err != nil || len(deduped.PinnedPlanIDs) != 100 {
		t.Errorf("pinning 100 plans with a repeat = %d ids, %v; want the repeat dropped and all 100 saved", len(deduped.PinnedPlanIDs), err)
	}
}
