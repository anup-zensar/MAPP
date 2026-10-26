package allwork_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/allwork"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/record"
)

var pgSavedAt = time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC)

// pgStore gives a store whose database already holds the people U-1001 and U-1002.
func pgStore(t *testing.T) (*database.DB, *allwork.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory := people.NewPostgresStore(db)
	meta := record.NewMeta(actor.System, pgSavedAt)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(
			directory.Put(ctx, people.Person{ID: "U-1001", Name: "Priya Raman", Email: "priya@example.com", Meta: meta}),
			directory.Put(ctx, people.Person{ID: "U-1002", Name: "Daniel Okoro", Email: "daniel@example.com", Meta: meta}))
	})
	return db, allwork.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgView(id, personID, name string, isDefault bool) allwork.StoredView {
	return allwork.StoredView{ID: id, PersonID: personID, Name: name, IsDefault: isDefault, CreatedAt: pgSavedAt,
		UpdatedAt: pgSavedAt, State: allwork.ViewState{View: "grid", Year: 2026, Columns: []string{"name", "budget"},
			Filters: allwork.Filters{Teams: []allwork.FilterValue{{ID: "T-1001", Label: "Lubricants Europe"}},
				PlanStatuses: []string{}},
			Sort: &allwork.Sort{Column: "name", Direction: "asc"}, Expansion: allwork.Expansion{Mode: "default"}}}
}

func TestPostgresStoreSavedViewsAreThePersonsOwnFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	mine := []allwork.StoredView{pgView("SV-1000", "U-1001", "My plans", true), pgView("SV-999", "U-1001", "Europe", false)}
	theirs := pgView("SV-1001", "U-1002", "My plans", true)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutView(ctx, mine[0]), s.PutView(ctx, theirs), s.PutView(ctx, mine[1]),
			s.PutView(ctx, pgView("SV-1002", "U-1001", "Deleted", false)))
	})
	pgWrite(t, db, func(ctx context.Context) error { return s.DeleteView(ctx, "SV-1002") })

	got, err := s.ListViews(t.Context(), "U-1001")

	if err != nil {
		t.Fatalf("ListViews() error: %v", err)
	}
	if diff := cmp.Diff([]allwork.StoredView{mine[1], mine[0]}, got); diff != "" {
		t.Errorf("ListViews() of U-1001 mismatch (-want +got):\n%s", diff)
	}
	if one, found, err := s.GetView(t.Context(), "SV-1001"); err != nil || !found || !cmp.Equal(theirs, one) {
		t.Errorf("GetView() = %+v, found %v, error %v; want %+v", one, found, err, theirs)
	}
}

func TestPostgresStoreAllWorkPreferencesFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	sorted := allwork.PreferencesRecord{PersonID: "U-1001", Columns: []string{"name", "budget"},
		Sort: &allwork.Sort{Column: "budget", Direction: "desc"}, UpdatedAt: pgSavedAt}
	unsorted := allwork.PreferencesRecord{PersonID: "U-1002", UpdatedAt: pgSavedAt}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutPreferences(ctx, sorted), s.PutPreferences(ctx, unsorted))
	})

	gotSorted, _, sortedErr := s.GetPreferences(t.Context(), "U-1001")
	gotUnsorted, _, unsortedErr := s.GetPreferences(t.Context(), "U-1002")

	if err := errors.Join(sortedErr, unsortedErr); err != nil {
		t.Fatalf("GetPreferences() error: %v", err)
	}
	if diff := cmp.Diff([]allwork.PreferencesRecord{sorted, unsorted}, []allwork.PreferencesRecord{gotSorted, gotUnsorted}); diff != "" {
		t.Errorf("GetPreferences() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreRefusesBrokenViewsFR004(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		view allwork.StoredView
	}{
		{name: "a second view of one person whose name differs only in letter case", view: pgView("SV-1002", "U-1001", "MY PLANS", false)},
		{name: "a second default view of one person", view: pgView("SV-1002", "U-1001", "Europe", true)},
		{name: "a view of a person who does not exist", view: pgView("SV-1002", "U-9999", "Europe", false)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := pgStore(t)
			pgWrite(t, db, func(ctx context.Context) error { return s.PutView(ctx, pgView("SV-1001", "U-1001", "My plans", true)) })

			err := db.Write(t.Context(), func(ctx context.Context) error { return s.PutView(ctx, tt.view) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
