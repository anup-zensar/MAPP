package fake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
)

type note struct {
	ID   string   `json:"id"`
	Tags []string `json:"tags"`
}

var errRefused = errors.New("refused")

func notesOf(db *fake.DB) *fake.Table[note] {
	return fake.TableOf(db, "notes", func(n note) string { return n.ID })
}

func listed(t *testing.T, notes *fake.Table[note]) []note {
	t.Helper()
	all, err := notes.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	return all
}

func TestWrite_FailedWritePutsEverythingBack_029FR003(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	notes := notesOf(db)
	counters := fake.NewCounters(db)
	if err := notes.Put(t.Context(), note{ID: "N-1", Tags: []string{"kept"}}, note{ID: "N-2"}); err != nil {
		t.Fatalf("Put() error: %v", err)
	}
	before := listed(t, notes)

	err := db.Write(t.Context(), func(ctx context.Context) error {
		if _, err := counters.Advance(ctx, "N", 1); err != nil {
			return err
		}
		if err := notes.Put(ctx, note{ID: "N-1", Tags: []string{"changed"}}, note{ID: "N-3"}); err != nil {
			return err
		}
		if err := db.Write(ctx, func(ctx context.Context) error { return notes.Delete(ctx, "N-2") }); err != nil {
			return err
		}
		return errRefused
	})

	if !errors.Is(err, errRefused) {
		t.Fatalf("Write() error = %v, want %v", err, errRefused)
	}
	if diff := cmp.Diff(before, listed(t, notes)); diff != "" {
		t.Errorf("records after the failed Write (-want +got):\n%s", diff)
	}
	if counters.Used() {
		t.Error("Used() = true after the failed Write, want its number taken back")
	}
}

func TestRead_RefusesChanges_029FR003(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	notes := notesOf(db)

	err := db.Read(t.Context(), func(ctx context.Context) error {
		putErr := notes.Put(ctx, note{ID: "N-1"})
		writeErr := db.Write(ctx, func(context.Context) error { return nil })
		return errors.Join(putErr, writeErr)
	})

	if !errors.Is(err, fake.ErrReadOnly) {
		t.Errorf("changes inside a Read error = %v, want %v", err, fake.ErrReadOnly)
	}
	if got := listed(t, notes); len(got) != 0 {
		t.Errorf("records = %v, want none", got)
	}
}

func TestTable_ListsByIDAndSharesNothing_029FR003(t *testing.T) {
	t.Parallel()
	notes := notesOf(fake.NewDB())
	if err := notes.Put(t.Context(), note{ID: "N-1000"}, note{ID: "N-999", Tags: []string{"a"}}); err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	read, _, err := notes.Get("N-999")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	read.Tags[0] = "changed by the caller"

	want := []note{{ID: "N-999", Tags: []string{"a"}}, {ID: "N-1000"}}
	if diff := cmp.Diff(want, listed(t, notes)); diff != "" {
		t.Errorf("List() (-want +got):\n%s", diff)
	}
}

func TestCopy_ChangesStayInTheCopy_029FR004(t *testing.T) {
	t.Parallel()
	original := fake.NewDB()
	if err := notesOf(original).Put(t.Context(), note{ID: "N-1"}); err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	copied := original.Copy()
	if err := notesOf(copied).Delete(t.Context(), "N-1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	if got := listed(t, notesOf(original)); len(got) != 1 {
		t.Errorf("original records = %v, want N-1 kept", got)
	}
	if got := listed(t, notesOf(copied)); len(got) != 0 {
		t.Errorf("copied records = %v, want none", got)
	}
}

func TestCounters_GiveOutReadableIDs_029FR003(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	generator := ids.NewGenerator(fake.NewCounters(db))
	var got []string

	err := db.Write(t.Context(), func(ctx context.Context) error {
		for range 2 {
			id, err := generator.Next(ctx, "P")
			if err != nil {
				return err
			}
			got = append(got, id)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if diff := cmp.Diff([]string{"P-1001", "P-1002"}, got); diff != "" {
		t.Errorf("Next() (-want +got):\n%s", diff)
	}
}
