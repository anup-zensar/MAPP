package ids_test

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
)

// The first ID of every prefix is 1001, and each prefix counts on its own (FR-027).
func TestNext_StartsAt1001_FR027(t *testing.T) {
	db := openStore(t, t.TempDir())
	generator := ids.NewGenerator(ids.NewJSONCounters())

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

// Two IDs taken in one transaction differ, because the transaction reads its own counter change (FR-027).
func TestNext_TwiceInOneTransaction_FR027(t *testing.T) {
	db := openStore(t, t.TempDir())
	generator := ids.NewGenerator(ids.NewJSONCounters())

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

// An ID is never given out again: not after its record was deleted, and not after the API restarted
// (FR-027).
func TestNext_NeverReused_FR027(t *testing.T) {
	dir := t.TempDir()
	db := openStore(t, dir)
	generator := ids.NewGenerator(ids.NewJSONCounters())

	first := nextID(t, db, generator, "SF")
	err := db.Write(t.Context(), func(ctx context.Context) error {
		if err := store.Put(ctx, "sample_folders", first, map[string]string{"id": first}); err != nil {
			return err
		}
		return store.Delete(ctx, "sample_folders", first)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	afterDelete := nextID(t, db, generator, "SF")

	reopened := openStore(t, dir)
	afterRestart := nextID(t, reopened, ids.NewGenerator(ids.NewJSONCounters()), "SF")

	if first != "SF-1001" || afterDelete != "SF-1002" || afterRestart != "SF-1003" {
		t.Errorf("got %q, %q, %q; want SF-1001, SF-1002, SF-1003", first, afterDelete, afterRestart)
	}
}

// The counters table keeps the last number given per prefix (data model §4).
func TestJSONCounters_StoreTheLastNumberPerPrefix(t *testing.T) {
	db := openStore(t, t.TempDir())
	generator := ids.NewGenerator(ids.NewJSONCounters())
	for range 6 {
		nextID(t, db, generator, "U")
	}

	type counter struct {
		Prefix string `json:"prefix"`
		Last   int    `json:"last"`
	}
	err := db.Read(t.Context(), func(ctx context.Context) error {
		got, found, err := store.Get[counter](ctx, "counters", "U")
		if !found || got != (counter{Prefix: "U", Last: 1006}) {
			t.Errorf("counter U = %+v, %v; want {U 1006}", got, found)
		}
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// IDs are taken inside the caller's Write, so that the counter and the new record are saved together.
func TestNext_NeedsAWrite(t *testing.T) {
	db := openStore(t, t.TempDir())
	generator := ids.NewGenerator(ids.NewJSONCounters())

	if _, err := generator.Next(t.Context(), "P"); !errors.Is(err, store.ErrNoTransaction) {
		t.Errorf("Next outside a transaction = %v, want store.ErrNoTransaction", err)
	}
	err := db.Read(t.Context(), func(ctx context.Context) error {
		if _, err := generator.Next(ctx, "P"); !errors.Is(err, store.ErrReadOnly) {
			t.Errorf("Next inside a Read = %v, want store.ErrReadOnly", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

func openStore(t *testing.T, dir string) *store.DB {
	t.Helper()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return db
}

// nextID takes one ID in a Write of its own.
func nextID(t *testing.T, db *store.DB, generator *ids.Generator, prefix string) string {
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

// Every record type has its own prefix: capital letters only, and no two record types share one, so every
// counter belongs to one table (FR-027; the data models of features 002-008).
func TestPrefixes_UniquePerRecordType_FR027(t *testing.T) {
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
	want := map[string]string{
		ids.PrefixModel: "MOD", ids.PrefixActivityType: "AT", ids.PrefixShellAttribute: "SA", ids.PrefixShellOption: "SO",
		ids.PrefixFinanceBudget: "FB", ids.PrefixFinanceBudgetItem: "FBI", ids.PrefixTeam: "T", ids.PrefixTeamMembership: "TM",
		ids.PrefixPlan: "P", ids.PrefixCampaign: "C", ids.PrefixInitiative: "I", ids.PrefixActivity: "A", ids.PrefixLineItem: "L",
		ids.PrefixApprovalWorkflow: "WF", ids.PrefixApprovalRound: "AR", ids.PrefixApprovalAction: "AH", ids.PrefixNotification: "NT",
		ids.PrefixSavedView: "SV", ids.PrefixApprovalDelegation: "DLG",
	}
	for got, wantPrefix := range want {
		if got != wantPrefix {
			t.Errorf("prefix %q, want %q", got, wantPrefix)
		}
	}
	if len(ids.Prefixes) != 24 {
		t.Errorf("%d prefixes, want 24", len(ids.Prefixes))
	}
}

var prefixForm = regexp.MustCompile(`^[A-Z]{1,3}$`)
