package samples_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
	"mapp/internal/samples"
)

var (
	priya     = actor.Actor{ID: "U-1001", Name: "Priya Raman"}
	daniel    = actor.Actor{ID: "U-1002", Name: "Daniel Okoro"}
	createdAt = time.Date(2026, time.October, 26, 9, 30, 0, 0, time.UTC)
	changedAt = createdAt.Add(time.Hour)
)

func newFolder(id, name string) samples.Folder {
	return samples.Folder{ID: id, Name: name, Meta: record.NewMeta(priya, createdAt)}
}

func newNote(id, folderID string) samples.Note {
	return samples.Note{ID: id, FolderID: folderID, Title: "Note " + id, Meta: record.NewMeta(priya, createdAt)}
}

func newStore(t *testing.T) (*database.DB, *samples.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	return db, samples.NewPostgresStore(db)
}

func mustWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func putFolders(t *testing.T, db *database.DB, s *samples.PostgresStore, folders ...samples.Folder) {
	t.Helper()
	mustWrite(t, db, func(ctx context.Context) error {
		for _, f := range folders {
			if err := s.PutFolder(ctx, f); err != nil {
				return err
			}
		}
		return nil
	})
}

func TestPostgresStoreFolderReadsBackAsSavedFR008(t *testing.T) {
	t.Parallel()
	db, s := newStore(t)
	want := newFolder("SF-1001", "Ideas")
	putFolders(t, db, s, want)

	got, found, err := s.GetFolder(t.Context(), want.ID)

	if err != nil || !found {
		t.Fatalf("GetFolder() = found %v, error %v; want the folder", found, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetFolder() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreMissingFolderIsNotFoundFR008(t *testing.T) {
	t.Parallel()
	_, s := newStore(t)

	_, found, err := s.GetFolder(t.Context(), "SF-9999")

	if err != nil || found {
		t.Errorf("GetFolder() = found %v, error %v; want not found and no error", found, err)
	}
}

func TestPostgresStorePutFolderReplacesFR008(t *testing.T) {
	t.Parallel()
	db, s := newStore(t)
	renamed := newFolder("SF-1001", "Plans")
	renamed.Touch(daniel, changedAt)
	putFolders(t, db, s, newFolder("SF-1001", "Ideas"))
	putFolders(t, db, s, renamed)

	got, err := s.ListFolders(t.Context())

	if err != nil {
		t.Fatalf("ListFolders() error: %v", err)
	}
	if diff := cmp.Diff([]samples.Folder{renamed}, got); diff != "" {
		t.Errorf("ListFolders() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreListFoldersShorterIDsFirstFR008(t *testing.T) {
	t.Parallel()
	db, s := newStore(t)
	putFolders(t, db, s, newFolder("SF-1000", "B"), newFolder("SF-999", "A"), newFolder("SF-1001", "C"))

	got, err := s.ListFolders(t.Context())

	if err != nil {
		t.Fatalf("ListFolders() error: %v", err)
	}
	if diff := cmp.Diff([]string{"SF-999", "SF-1000", "SF-1001"}, folderIDs(got)); diff != "" {
		t.Errorf("ListFolders() order mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreDeleteFolderFR008(t *testing.T) {
	t.Parallel()
	db, s := newStore(t)
	putFolders(t, db, s, newFolder("SF-1001", "Ideas"), newFolder("SF-1002", "Plans"))

	mustWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.DeleteFolder(ctx, "SF-1001"), s.DeleteFolder(ctx, "SF-9999"))
	})

	got, err := s.ListFolders(t.Context())
	if err != nil {
		t.Fatalf("ListFolders() error: %v", err)
	}
	if diff := cmp.Diff([]string{"SF-1002"}, folderIDs(got)); diff != "" {
		t.Errorf("folders after DeleteFolder mismatch (-want +got):\n%s", diff)
	}
}

func withNotes(t *testing.T) (*database.DB, *samples.PostgresStore) {
	t.Helper()
	db, s := newStore(t)
	putFolders(t, db, s, newFolder("SF-1001", "Ideas"), newFolder("SF-1002", "Plans"))
	mustWrite(t, db, func(ctx context.Context) error {
		return errors.Join(
			s.PutNote(ctx, newNote("SN-1000", "SF-1001")),
			s.PutNote(ctx, newNote("SN-999", "SF-1002")),
			s.PutNote(ctx, newNote("SN-1001", "SF-1001")),
		)
	})
	return db, s
}

func TestPostgresStoreNotesInFolderFR008(t *testing.T) {
	t.Parallel()
	_, s := withNotes(t)

	got, err := s.NotesInFolder(t.Context(), "SF-1001")

	if err != nil {
		t.Fatalf("NotesInFolder() error: %v", err)
	}
	want := []samples.Note{newNote("SN-1000", "SF-1001"), newNote("SN-1001", "SF-1001")}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("NotesInFolder() mismatch (-want +got):\n%s", diff)
	}
	if one, found, err := s.GetNote(t.Context(), "SN-1001"); err != nil || !found || !cmp.Equal(want[1], one) {
		t.Errorf("GetNote() = %+v, found %v, error %v; want %+v", one, found, err, want[1])
	}
}

func TestPostgresStoreDeleteNoteFR008(t *testing.T) {
	t.Parallel()
	db, s := withNotes(t)

	mustWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.DeleteNote(ctx, "SN-1000"), s.DeleteNote(ctx, "SN-9999"))
	})

	if _, found, err := s.GetNote(t.Context(), "SN-1000"); err != nil || found {
		t.Errorf("GetNote() of the deleted note = found %v, error %v; want not found and no error", found, err)
	}
	got, err := s.ListNotes(t.Context())
	if err != nil {
		t.Fatalf("ListNotes() error: %v", err)
	}
	if diff := cmp.Diff([]string{"SN-999", "SN-1001"}, noteIDs(got)); diff != "" {
		t.Errorf("ListNotes() after DeleteNote mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreChecksParentsAtCommitFR004(t *testing.T) {
	t.Parallel()
	db, s := withNotes(t)

	err := db.Write(t.Context(), func(ctx context.Context) error {
		return errors.Join(s.DeleteFolder(ctx, "SF-1001"), s.DeleteNote(ctx, "SN-1000"), s.DeleteNote(ctx, "SN-1001"))
	})

	if err != nil {
		t.Errorf("Write() removing a folder before its notes error = %v, want nil", err)
	}
}

func TestPostgresStoreRefusesBrokenDataFR004(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		write func(ctx context.Context, s *samples.PostgresStore) error
	}{
		{
			name: "a note in a folder that does not exist",
			write: func(ctx context.Context, s *samples.PostgresStore) error {
				return s.PutNote(ctx, newNote("SN-1001", "SF-404"))
			},
		},
		{
			name:  "a folder deleted while it still holds a note",
			write: func(ctx context.Context, s *samples.PostgresStore) error { return s.DeleteFolder(ctx, "SF-1001") },
		},
		{
			name: "a second folder whose name differs only in letter case",
			write: func(ctx context.Context, s *samples.PostgresStore) error {
				return s.PutFolder(ctx, newFolder("SF-1002", "IDEAS"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := newStore(t)
			putFolders(t, db, s, newFolder("SF-1001", "Ideas"))
			mustWrite(t, db, func(ctx context.Context) error { return s.PutNote(ctx, newNote("SN-1000", "SF-1001")) })

			err := db.Write(t.Context(), func(ctx context.Context) error { return tt.write(ctx, s) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}

// TestPostgresStoreRacingRenamesFR006 lets two people save a rename of the same folder from version 1 at
// once. One save wins; the other must be refused with version_conflict, never overwrite it.
func TestPostgresStoreRacingRenamesFR006(t *testing.T) {
	t.Parallel()
	db, s := newStore(t)
	putFolders(t, db, s, newFolder("SF-1001", "Ideas"))

	var renames sync.WaitGroup
	names := []string{"Plans", "Notes"}
	results := make([]error, len(names))
	for i, name := range names {
		renames.Go(func() { results[i] = renameFromVersion1(t.Context(), db, s, name) })
	}
	renames.Wait()

	conflicts := 0
	for _, err := range results {
		if appErr, ok := errors.AsType[*errs.Error](err); ok && appErr.Code == errs.CodeVersionConflict {
			conflicts++
		} else if err != nil {
			t.Fatalf("rename error = %v, want nil or version_conflict", err)
		}
	}
	got, _, err := s.GetFolder(t.Context(), "SF-1001")
	if conflicts != 1 || err != nil || got.Version != 2 {
		t.Errorf("conflicts = %d, stored version = %d (error %v); want 1 conflict and version 2", conflicts, got.Version, err)
	}
}

// renameFromVersion1 renames the folder as the samples service does, in a transaction of any store.Transactor.
func renameFromVersion1(ctx context.Context, tx store.Transactor, repo samples.Repository, name string) error {
	return tx.Write(ctx, func(ctx context.Context) error {
		folder, _, err := repo.GetFolder(ctx, "SF-1001")
		if err != nil {
			return err
		}
		if conflict := folder.CheckVersion(1, "sample folder"); conflict != nil {
			return conflict
		}
		folder.Name = name
		folder.Touch(daniel, changedAt)
		return repo.PutFolder(ctx, folder)
	})
}

func folderIDs(folders []samples.Folder) []string {
	ids := make([]string, 0, len(folders))
	for _, f := range folders {
		ids = append(ids, f.ID)
	}
	return ids
}

func noteIDs(notes []samples.Note) []string {
	ids := make([]string, 0, len(notes))
	for _, n := range notes {
		ids = append(ids, n.ID)
	}
	return ids
}
