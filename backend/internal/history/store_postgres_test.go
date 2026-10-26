package history_test

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
)

var (
	priya     = actor.Actor{ID: "U-1001", Name: "Priya Raman"}
	deletedAt = time.Date(2026, time.October, 26, 9, 30, 0, 0, time.UTC)
)

func newStore(t *testing.T) (*database.DB, *history.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	return db, history.NewPostgresStore(db)
}

func mustWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func logEntry(seq int, items []history.LogItem) history.Entry {
	return history.Entry{
		ID: "LOG-" + strconv.Itoa(seq), Seq: seq, At: deletedAt, Actor: priya, Action: history.ActionChanged,
		Verb: "changed", RecordType: "sampleFolder", RecordTypeLabel: "sample folder", RecordID: "SF-1001",
		RecordName: "Ideas", Path: []history.RecordRef{},
		Changes: []history.Change{{
			Field: "name", Label: "Name", Old: json.RawMessage(`"Idea"`), New: json.RawMessage(`"Ideas"`),
			OldText: "Idea", NewText: "Ideas",
		}},
		Items: items,
	}
}

func TestPostgresStoreEntriesReadBackInSequenceFR023(t *testing.T) {
	t.Parallel()
	db, s := newStore(t)
	withItems := logEntry(9999, []history.LogItem{{RecordType: "sampleNote", RecordTypeLabel: "sample note",
		RecordID: "SN-1001", RecordName: "Agenda", Details: ""}})
	emptyItems := logEntry(10000, []history.LogItem{})
	mustWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.AppendEntry(ctx, withItems), s.AppendEntry(ctx, emptyItems))
	})

	got, err := s.Entries(t.Context())

	if err != nil {
		t.Fatalf("Entries() error: %v", err)
	}
	emptyItems.Items = nil // the JSON store leaves an empty list out of the record
	if diff := cmp.Diff([]history.Entry{withItems, emptyItems}, got); diff != "" {
		t.Errorf("Entries() mismatch (-want +got):\n%s", diff)
	}
}

func binEntry(id string) history.BinEntry {
	return history.BinEntry{
		ID: id, RecordType: "sampleNote", RecordTypeLabel: "sample note", RecordID: "SN-1001",
		RecordName: "Agenda", Path: []history.RecordRef{{Type: "sampleFolder", ID: "SF-1001", Name: "Ideas"}},
		Parent:    &history.RecordRef{Type: "sampleFolder", ID: "SF-1001", Name: "Ideas"},
		DeletedAt: deletedAt, DeletedBy: priya, Contents: []history.ContentCount{},
		Snapshots: []history.Snapshot{{Kind: "sampleNote", ID: "SN-1001", Name: "Agenda", ParentID: "SF-1001",
			Data: json.RawMessage(`{"id":"SN-1001","folderId":"SF-1001","title":"Agenda"}`)}},
	}
}

func TestPostgresStoreBinEntriesReadBackAsSavedFR024(t *testing.T) {
	t.Parallel()
	db, s := newStore(t)
	open := binEntry("BIN-1000")
	resolved := binEntry("BIN-999")
	mustWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutBinEntry(ctx, open), s.PutBinEntry(ctx, resolved))
	})
	resolved.Snapshots = []history.Snapshot{}
	resolved.Resolution = &history.Resolution{How: "restored", By: priya, At: deletedAt.Add(time.Hour)}
	mustWrite(t, db, func(ctx context.Context) error { return s.PutBinEntry(ctx, resolved) })

	got, err := s.BinEntries(t.Context())

	if err != nil {
		t.Fatalf("BinEntries() error: %v", err)
	}
	if diff := cmp.Diff([]history.BinEntry{resolved, open}, got); diff != "" {
		t.Errorf("BinEntries() mismatch (-want +got):\n%s", diff)
	}
	one, found, err := s.GetBinEntry(t.Context(), "BIN-1000")
	if err != nil || !found || !cmp.Equal(open, one) {
		t.Errorf("GetBinEntry() = %+v, found %v, error %v; want %+v", one, found, err, open)
	}
	if _, found, err = s.GetBinEntry(t.Context(), "BIN-404"); err != nil || found {
		t.Errorf("GetBinEntry() of a missing entry = found %v, error %v; want not found and no error", found, err)
	}
}
