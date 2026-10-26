package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/auth"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/record"
)

var signedInAt = time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC)

func newSessionStore(t *testing.T) (*database.DB, *auth.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	err := db.Write(t.Context(), func(ctx context.Context) error {
		return people.NewPostgresStore(db).Put(ctx, people.Person{ID: "U-1001", Name: "Priya Raman",
			Email: "priya@example.com", Active: true, Meta: record.NewMeta(actor.System, signedInAt)})
	})
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	return db, auth.NewPostgresStore(db)
}

func session(hashDigit, personID string) auth.Session {
	return auth.Session{ID: strings.Repeat(hashDigit, 64), PersonID: personID, CreatedAt: signedInAt,
		LastUsedAt: signedInAt}
}

func TestPostgresStoreSessionsFR008(t *testing.T) {
	t.Parallel()
	db, s := newSessionStore(t)
	later, earlier := session("b", "U-1001"), session("a", "U-1001")
	err := db.Write(t.Context(), func(ctx context.Context) error {
		return errors.Join(s.Put(ctx, later), s.Put(ctx, earlier), s.Put(ctx, session("c", "U-1001")))
	})
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	later.LastUsedAt = signedInAt.Add(time.Hour)

	err = db.Write(t.Context(), func(ctx context.Context) error {
		return errors.Join(s.Put(ctx, later), s.Delete(ctx, session("c", "").ID), s.Delete(ctx, session("d", "").ID))
	})

	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if all, err := s.List(t.Context()); err != nil || !cmp.Equal([]auth.Session{earlier, later}, all) {
		t.Errorf("List() = %+v, error %v; want %+v", all, err, []auth.Session{earlier, later})
	}
	if _, found, err := s.Get(t.Context(), session("c", "").ID); err != nil || found {
		t.Errorf("Get() of a deleted session = found %v, error %v; want not found", found, err)
	}
}

func TestPostgresStoreRefusesASessionOfNoOneFR004(t *testing.T) {
	t.Parallel()
	db, s := newSessionStore(t)

	err := db.Write(t.Context(), func(ctx context.Context) error { return s.Put(ctx, session("a", "U-9999")) })

	if err == nil {
		t.Errorf("Write() error = nil, want the database to refuse a session of a person who does not exist")
	}
}
