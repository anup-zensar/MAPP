package database_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/database/sqlcdb"
)

var (
	errStop = errors.New("stop")
	savedAt = time.Date(2026, time.October, 26, 9, 30, 0, 0, time.UTC)
)

func folder(id string, at time.Time) sqlcdb.PutSampleFolderParams {
	return sqlcdb.PutSampleFolderParams{
		ID: id, Name: "Folder " + id, Version: 1,
		CreatedAt: at, CreatedByID: "U-1001", CreatedByName: "Priya Raman",
		UpdatedAt: at, UpdatedByID: "U-1001", UpdatedByName: "Priya Raman",
	}
}

func folderIDs(t *testing.T, db *database.DB) []string {
	t.Helper()
	rows, err := db.Queries(t.Context()).ListSampleFolders(t.Context())
	if err != nil {
		t.Fatalf("ListSampleFolders() error: %v", err)
	}
	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func TestWriteKeepsNothingWhenFnFailsFR006(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		write func(db *database.DB) func(ctx context.Context) error
	}{
		{
			name: "the failing function's own change",
			write: func(db *database.DB) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					if err := db.Queries(ctx).PutSampleFolder(ctx, folder("SF-1001", savedAt)); err != nil {
						return err
					}
					return errStop
				}
			},
		},
		{
			name: "a nested Write's change, which joined the failing one",
			write: func(db *database.DB) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					err := db.Write(ctx, func(ctx context.Context) error {
						return db.Queries(ctx).PutSampleFolder(ctx, folder("SF-1001", savedAt))
					})
					return errors.Join(err, errStop)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := dbtest.NewDB(t)

			err := db.Write(t.Context(), tt.write(db))

			if !errors.Is(err, errStop) {
				t.Errorf("Write() error = %v, want %v", err, errStop)
			}
			if ids := folderIDs(t, db); len(ids) != 0 {
				t.Errorf("folders after the failed Write = %v, want none", ids)
			}
		})
	}
}

func TestWriteInsideReadIsRefusedFR007(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)

	err := db.Read(t.Context(), func(ctx context.Context) error {
		return db.Write(ctx, func(context.Context) error { return nil })
	})

	if !errors.Is(err, database.ErrReadOnly) {
		t.Errorf("Write() inside Read error = %v, want %v", err, database.ErrReadOnly)
	}
}

func TestTimestampsComeBackInUTCFR005(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	inIndia := time.Date(2026, time.October, 26, 15, 0, 0, 0, time.FixedZone("IST", 5*3600+1800))
	if err := db.Queries(t.Context()).PutSampleFolder(t.Context(), folder("SF-1001", inIndia)); err != nil {
		t.Fatalf("PutSampleFolder() error: %v", err)
	}

	row, err := db.Queries(t.Context()).GetSampleFolder(t.Context(), "SF-1001")

	if err != nil {
		t.Fatalf("GetSampleFolder() error: %v", err)
	}
	if !row.CreatedAt.Equal(inIndia) || row.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt = %v, want %v in UTC", row.CreatedAt, inIndia.UTC())
	}
}

func TestMigrateRunsConcurrentlyWithoutHarmFR002(t *testing.T) {
	t.Parallel()
	db := dbtest.NewEmptyDB(t)

	var runs sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		runs.Go(func() { errs[i] = database.Migrate(t.Context(), db) })
	}
	runs.Wait()

	if err := errors.Join(errs...); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	if ids := folderIDs(t, db); len(ids) != 0 {
		t.Errorf("folders in a freshly migrated database = %v, want none", ids)
	}
}
