package ids_test

import (
	"context"
	"regexp"
	"testing"

	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
)

// The first ID of every prefix is 1001, and each prefix counts on its own.
func TestNext_StartsAt1001_FR027(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	generator := ids.NewGenerator(fake.NewCounters(db))

	var plan, folder string
	var logNumber int
	err := db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		if plan, err = generator.Next(ctx, "P"); err != nil {
			return err
		}
		if folder, err = generator.Next(ctx, "SF"); err != nil {
			return err
		}
		logNumber, err = generator.NextNumber(ctx, "LOG")
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if plan != "P-1001" || folder != "SF-1001" || logNumber != 1001 {
		t.Errorf("got %q, %q, %d; want P-1001, SF-1001, 1001", plan, folder, logNumber)
	}
}

// Two IDs taken in one transaction differ, because the transaction reads its own counter change.
func TestNext_TwiceInOneTransaction_FR027(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	generator := ids.NewGenerator(fake.NewCounters(db))

	var first, second string
	err := db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		if first, err = generator.Next(ctx, "P"); err != nil {
			return err
		}
		second, err = generator.Next(ctx, "P")
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if first != "P-1001" || second != "P-1002" {
		t.Errorf("got %q and %q, want P-1001 and P-1002", first, second)
	}
}

// An ID is never given out again: not after its record was deleted, and not after the API restarted.
func TestNext_NeverReused_FR027(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	generator := ids.NewGenerator(fake.NewCounters(db))
	folders := fake.TableOf(db, "sample_folders", func(id string) string { return id })

	first := nextID(t, db, generator, "SF")
	err := db.Write(t.Context(), func(ctx context.Context) error {
		if err := folders.Put(ctx, first); err != nil {
			return err
		}
		return folders.Delete(ctx, first)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	afterDelete := nextID(t, db, generator, "SF")

	afterRestart := nextID(t, db, ids.NewGenerator(fake.NewCounters(db)), "SF")

	if first != "SF-1001" || afterDelete != "SF-1002" || afterRestart != "SF-1003" {
		t.Errorf("got %q, %q, %q; want SF-1001, SF-1002, SF-1003", first, afterDelete, afterRestart)
	}
}

func nextID(t *testing.T, db *fake.DB, generator *ids.Generator, prefix string) string {
	t.Helper()
	var id string
	err := db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		id, err = generator.Next(ctx, prefix)
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	return id
}

// Every record type has its own prefix, so every counter belongs to one table.
func TestPrefixes_UniquePerRecordType_FR027(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, prefix := range ids.Prefixes {
		if !prefixForm.MatchString(prefix) {
			t.Errorf("prefix %q is not 1 to 3 capital letters", prefix)
		}
		if seen[prefix] {
			t.Errorf("prefix %q is listed twice", prefix)
		}
		seen[prefix] = true
	}
}

var prefixForm = regexp.MustCompile(`^[A-Z]{1,3}$`)
