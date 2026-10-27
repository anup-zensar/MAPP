package history

import (
	"context"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
)

// The people of these tests.
var (
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	mei    = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// fixture is one test's history service on a temporary data folder, with a clock that moves only when the
// test says so.
type fixture struct {
	service *Service
	db      *store.DB
	dir     string
	clock   *testClock
	removed []string // the ids the test kinds removed, in order
}

// newBareFixture returns a fixture whose service knows no kinds yet.
func newBareFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{dir: t.TempDir(), clock: &testClock{now: start}}
	f.open(t)
	return f
}

// newFixture returns a fixture whose service knows the kinds parent, child and grandchild.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := newBareFixture(t)
	f.register(t, f.standardKinds()...)
	return f
}

// open opens the data folder and builds a new service on it, as the API does when it starts.
func (f *fixture) open(t *testing.T) {
	t.Helper()
	db, err := store.Open(f.dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	f.db = db
	f.service = NewService(db, NewJSONStore(), ids.NewGenerator(ids.NewJSONCounters()), f.clock.Now)
}

// reopen starts again from the data folder with the standard kinds, as after a restart of the API.
func (f *fixture) reopen(t *testing.T) {
	t.Helper()
	f.open(t)
	f.register(t, f.standardKinds()...)
}

// register registers kinds in order and stops the test when one is refused.
func (f *fixture) register(t *testing.T, kinds ...Kind) {
	t.Helper()
	for _, k := range kinds {
		if err := f.service.Register(k); err != nil {
			t.Fatalf("Register(%s): %v", k.Name, err)
		}
	}
}

// addTree saves the records most tests use: the parent P-1 "Parent one" with the children C-1 and C-2;
// the grandchildren G-1 and G-2 under C-1, and G-3 under C-2.
func (f *fixture) addTree(t *testing.T) {
	t.Helper()
	f.put(t, parentsTable, newRecord("P-1", "Parent one", ""))
	f.put(t, childrenTable, newRecord("C-1", "Child one", "P-1"), newRecord("C-2", "Child two", "P-1"))
	f.put(t, grandchildrenTable, newRecord("G-1", "Grandchild one", "C-1"),
		newRecord("G-2", "Grandchild two", "C-1"), newRecord("G-3", "Grandchild three", "C-2"))
}

// put saves records in table, in one Write.
func (f *fixture) put(t *testing.T, table string, records ...testRecord) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		for _, r := range records {
			if err := store.Put(ctx, table, r.ID, r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// stored returns the record id of table, and false when it is not stored.
func (f *fixture) stored(t *testing.T, table, id string) (testRecord, bool) {
	t.Helper()
	var r testRecord
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		r, found, err = store.Get[testRecord](ctx, table, id)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return r, found
}

// allRecords returns every stored record of the test kinds, by table.
func (f *fixture) allRecords(t *testing.T) map[string][]testRecord {
	t.Helper()
	all := map[string][]testRecord{}
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		for _, table := range []string{parentsTable, childrenTable, grandchildrenTable} {
			records, err := store.List[testRecord](ctx, table)
			if err != nil {
				return err
			}
			all[table] = records
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return all
}

// entries returns the whole activity log, oldest first.
func (f *fixture) entries(t *testing.T) []Entry {
	t.Helper()
	var entries []Entry
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entries, err = NewJSONStore().Entries(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return entries
}

// binEntry returns the stored recycle-bin entry id, with its snapshots and resolution.
func (f *fixture) binEntry(t *testing.T, id string) BinEntry {
	t.Helper()
	var entry BinEntry
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entry, found, err = NewJSONStore().GetBinEntry(ctx, id)
		return err
	})
	if err != nil || !found {
		t.Fatalf("GetBinEntry(%s) = found %v, %v", id, found, err)
	}
	return entry
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// appendRaw appends e as a in one Write, as Purge and EmptyBin do, and stops the test when that fails.
func (f *fixture) appendRaw(t *testing.T, a actor.Actor, e Entry) Entry {
	t.Helper()
	var entry Entry
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		entry, err = f.service.appendEntry(ctx, a, e)
		return err
	})
	if err != nil {
		t.Fatalf("appendEntry: %v", err)
	}
	return entry
}

// removeForGood removes a record from its table without the recycle bin, as a permanent delete leaves it:
// not stored and in no entry of the bin.
func (f *fixture) removeForGood(t *testing.T, table, id string) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error { return store.Delete(ctx, table, id) })
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// lastEntry returns the newest entry of the log.
func (f *fixture) lastEntry(t *testing.T) Entry {
	t.Helper()
	entries := f.entries(t)
	return entries[len(entries)-1]
}
