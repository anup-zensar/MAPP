package people_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/record"
)

var savedMeta = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))

func newPeopleStore(t *testing.T) (*database.DB, *people.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	return db, people.NewPostgresStore(db)
}

func mustSave(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func storedPerson(id, email, managerID string) people.Person {
	return people.Person{ID: id, Name: "Person " + id, Email: email, JobTitle: "Planner", Active: true,
		ManagerID: managerID, Meta: savedMeta}
}

func storedTeam(id, name string) people.Team {
	return people.Team{ID: id, Name: name, Description: "Marketing for " + name, Colour: "blue", Active: true,
		Meta: savedMeta}
}

func TestPostgresStorePeopleReadBackAsSavedFR008(t *testing.T) {
	t.Parallel()
	db, s := newPeopleStore(t)
	priya := storedPerson("U-1001", "priya@example.com", "")
	daniel := storedPerson("U-1002", "daniel@example.com", "U-1001")
	mustSave(t, db, func(ctx context.Context) error { return errors.Join(s.Put(ctx, daniel), s.Put(ctx, priya)) })

	everyone, err := s.List(t.Context())

	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if diff := cmp.Diff([]people.Person{priya, daniel}, everyone); diff != "" {
		t.Errorf("List() mismatch (-want +got):\n%s", diff)
	}
	if got, found, err := s.FindByEmail(t.Context(), "daniel@example.com"); err != nil || !found || got != daniel {
		t.Errorf("FindByEmail() = %+v, found %v, error %v; want %+v", got, found, err, daniel)
	}
}

func TestPostgresStoreUnknownPersonIsNotFoundFR008(t *testing.T) {
	t.Parallel()
	_, s := newPeopleStore(t)

	_, foundByEmail, emailErr := s.FindByEmail(t.Context(), "nobody@example.com")
	_, foundByID, idErr := s.Get(t.Context(), "U-9999")

	if err := errors.Join(emailErr, idErr); err != nil || foundByEmail || foundByID {
		t.Errorf("FindByEmail() found %v, Get() found %v, error %v; want neither found and no error", foundByEmail, foundByID, err)
	}
}

func TestPostgresStorePreferencesReadBackAsSavedFR008(t *testing.T) {
	t.Parallel()
	db, s := newPeopleStore(t)
	pinned := people.Preferences{PersonID: "U-1002", Theme: "dark", DisplayCurrency: "EUR",
		DateFormat: "DD/MM/YYYY", HomeScope: "everything", PinnedPlanIDs: []string{"P-1002", "P-1001"}}
	unpinned := people.Preferences{PersonID: "U-1001", Theme: "light", SidebarCollapsed: true,
		PinnedPlanIDs: []string{}}
	mustSave(t, db, func(ctx context.Context) error {
		return errors.Join(s.Put(ctx, storedPerson("U-1001", "a@example.com", "")),
			s.Put(ctx, storedPerson("U-1002", "b@example.com", "")),
			s.PutPreferences(ctx, pinned), s.PutPreferences(ctx, unpinned))
	})

	all, err := s.ListPreferences(t.Context())

	if err != nil {
		t.Fatalf("ListPreferences() error: %v", err)
	}
	unpinned.PinnedPlanIDs = nil // the JSON store leaves an empty list out of the record
	if diff := cmp.Diff([]people.Preferences{unpinned, pinned}, all); diff != "" {
		t.Errorf("ListPreferences() mismatch (-want +got):\n%s", diff)
	}
	if _, found, err := s.GetPreferences(t.Context(), "U-1003"); err != nil || found {
		t.Errorf("GetPreferences() of someone without any = found %v, error %v; want not found", found, err)
	}
}

func TestPostgresStoreTeamsAndMembershipsFR008(t *testing.T) {
	t.Parallel()
	db, s := newPeopleStore(t)
	europe, china := storedTeam("T-1001", "Lubricants Europe"), storedTeam("T-1002", "Mobility China")
	memberships := []people.Membership{
		{ID: "TM-1001", TeamID: "T-1001", PersonID: "U-1001", Meta: savedMeta},
		{ID: "TM-1002", TeamID: "T-1001", PersonID: "U-1002", Meta: savedMeta},
		{ID: "TM-1003", TeamID: "T-1002", PersonID: "U-1002", Meta: savedMeta},
	}
	mustSave(t, db, func(ctx context.Context) error {
		return errors.Join(s.Put(ctx, storedPerson("U-1001", "a@example.com", "")),
			s.Put(ctx, storedPerson("U-1002", "b@example.com", "")), s.Put(ctx, storedPerson("U-1003", "c@example.com", "")),
			s.PutTeam(ctx, europe), s.PutTeam(ctx, china), s.PutMembership(ctx, memberships[0]),
			s.PutMembership(ctx, memberships[1]), s.PutMembership(ctx, memberships[2]))
	})

	ofTeam, teamErr := s.MembershipsOfTeam(t.Context(), "T-1001")
	ofPerson, personErr := s.MembershipsOfPerson(t.Context(), "U-1002")
	none, noneErr := s.MembershipsOfPerson(t.Context(), "U-1003")

	if err := errors.Join(teamErr, personErr, noneErr); err != nil {
		t.Fatalf("Memberships…() error: %v", err)
	}
	if diff := cmp.Diff(memberships[:2], ofTeam); diff != "" {
		t.Errorf("MembershipsOfTeam() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(memberships[1:], ofPerson); diff != "" {
		t.Errorf("MembershipsOfPerson() mismatch (-want +got):\n%s", diff)
	}
	if none == nil || len(none) != 0 {
		t.Errorf("MembershipsOfPerson() of someone in no team = %#v, want an empty list", none)
	}
	mustSave(t, db, func(ctx context.Context) error {
		return errors.Join(s.DeleteMembership(ctx, "TM-1003"), s.DeleteTeam(ctx, "T-1002"))
	})
	if teams, err := s.ListTeams(t.Context()); err != nil || !cmp.Equal([]people.Team{europe}, teams) {
		t.Errorf("ListTeams() after the delete = %+v, error %v; want only %s", teams, err, europe.ID)
	}
}

func TestPostgresStoreRefusesBrokenPeopleDataFR004(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		write func(ctx context.Context, s *people.PostgresStore) error
	}{
		{name: "an e-mail address that differs only in letter case", write: func(ctx context.Context, s *people.PostgresStore) error {
			return s.Put(ctx, storedPerson("U-1003", "PRIYA@example.com", ""))
		}},
		{name: "a manager who does not exist", write: func(ctx context.Context, s *people.PostgresStore) error {
			return s.Put(ctx, storedPerson("U-1003", "c@example.com", "U-9999"))
		}},
		{name: "a person who is their own manager", write: func(ctx context.Context, s *people.PostgresStore) error {
			return s.Put(ctx, storedPerson("U-1001", "priya@example.com", "U-1001"))
		}},
		{name: "a team name that differs only in letter case", write: func(ctx context.Context, s *people.PostgresStore) error {
			return s.PutTeam(ctx, storedTeam("T-1002", "LUBRICANTS EUROPE"))
		}},
		{name: "the same person twice in one team", write: func(ctx context.Context, s *people.PostgresStore) error {
			return s.PutMembership(ctx, people.Membership{ID: "TM-1002", TeamID: "T-1001", PersonID: "U-1001", Meta: savedMeta})
		}},
		{name: "a team deleted while it still has members", write: func(ctx context.Context, s *people.PostgresStore) error {
			return s.DeleteTeam(ctx, "T-1001")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := newPeopleStore(t)
			mustSave(t, db, func(ctx context.Context) error {
				return errors.Join(s.Put(ctx, storedPerson("U-1001", "priya@example.com", "")),
					s.PutTeam(ctx, storedTeam("T-1001", "Lubricants Europe")),
					s.PutMembership(ctx, people.Membership{ID: "TM-1001", TeamID: "T-1001", PersonID: "U-1001", Meta: savedMeta}))
			})

			err := db.Write(t.Context(), func(ctx context.Context) error { return tt.write(ctx, s) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
