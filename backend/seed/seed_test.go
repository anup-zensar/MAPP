package seed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
	"northstar/seed"
)

// loadedAt is the time the tests pass to Load.
var loadedAt = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The seed loads the six demo people of docs/product/demo-users-and-data.md in the order of that table, all
// active and created by NorthStar at the time given (FR-028, data model §1).
func TestLoad_SixDemoPeople_FR028(t *testing.T) {
	db := openStore(t)

	if err := seed.Load(t.Context(), db, fixedClock); err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []people.Person{
		demoPerson("U-1001", "Alex Morgan", "alex.morgan@northstar.demo", "Marketing administrator"),
		demoPerson("U-1002", "Priya Raman", "priya.raman@northstar.demo", "Marketer"),
		demoPerson("U-1003", "Daniel Okoro", "daniel.okoro@northstar.demo", "Marketing team leader"),
		demoPerson("U-1004", "Mei Lin", "mei.lin@northstar.demo", "Marketer"),
		demoPerson("U-1005", "Carlos Mendes", "carlos.mendes@northstar.demo", "Business champion"),
		demoPerson("U-1006", "Sara Novak", "sara.novak@northstar.demo", "Local finance advisor"),
	}
	got := storedPeople(t, db)
	if len(got) != len(want) {
		t.Fatalf("loaded %d people, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("person %d =\n%+v\nwant\n%+v", i+1, got[i], want[i])
		}
	}
}

// The people's IDs come from the readable-ID counter, so the next person gets U-1007 (research R4, R22).
func TestLoad_IDsComeFromTheCounter_FR027(t *testing.T) {
	db := openStore(t)
	if err := seed.Load(t.Context(), db, fixedClock); err != nil {
		t.Fatalf("Load: %v", err)
	}

	var next string
	err := db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		next, err = ids.NewGenerator(ids.NewJSONCounters()).Next(ctx, "U")
		return err
	})

	if err != nil || next != "U-1007" {
		t.Errorf("next person ID = %q, %v; want U-1007", next, err)
	}
}

// The demo data is loaded only into an empty store; a store with any data is refused and left as it was
// (FR-028).
func TestLoad_RefusesAStoreWithData_FR028(t *testing.T) {
	db := openStore(t)
	err := db.Write(t.Context(), func(ctx context.Context) error {
		return store.Put(ctx, "sample_folders", "SF-1001", map[string]string{"id": "SF-1001"})
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = seed.Load(t.Context(), db, fixedClock)

	if !errors.Is(err, seed.ErrNotEmpty) {
		t.Errorf("Load = %v, want ErrNotEmpty", err)
	}
	if got := storedPeople(t, db); len(got) != 0 {
		t.Errorf("people = %+v, want none", got)
	}
}

// Loading twice is refused the second time, so the demo people are never doubled (FR-028).
func TestLoad_Twice_FR028(t *testing.T) {
	db := openStore(t)
	if err := seed.Load(t.Context(), db, fixedClock); err != nil {
		t.Fatalf("first Load: %v", err)
	}

	err := seed.Load(t.Context(), db, fixedClock)

	if !errors.Is(err, seed.ErrNotEmpty) {
		t.Errorf("second Load = %v, want ErrNotEmpty", err)
	}
	if got := storedPeople(t, db); len(got) != 6 {
		t.Errorf("%d people after loading twice, want 6", len(got))
	}
}

func fixedClock() time.Time {
	return loadedAt
}

func openStore(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return db
}

// demoPerson is a seeded person: active, created by NorthStar itself at loadedAt.
func demoPerson(id, name, email, jobTitle string) people.Person {
	return people.Person{
		ID:       id,
		Name:     name,
		Email:    email,
		JobTitle: jobTitle,
		Active:   true,
		Meta:     record.NewMeta(actor.System, loadedAt),
	}
}

// storedPeople returns every stored person, ordered by id.
func storedPeople(t *testing.T, db *store.DB) []people.Person {
	t.Helper()
	var list []people.Person
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		list, err = people.NewJSONRepository().List(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return list
}
