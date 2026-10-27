package history

import (
	"context"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
)

var (
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	mei    = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
)

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

type fixture struct {
	service *Service
	db      *fake.DB
	clock   *testClock
	removed []string
}

func newBareFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{db: fake.NewDB(), clock: &testClock{now: start}}
	f.open()
	return f
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := newBareFixture(t)
	f.register(t, f.standardKinds()...)
	return f
}

func (f *fixture) open() {
	f.service = NewService(f.db, newFakeStore(f.db), ids.NewGenerator(fake.NewCounters(f.db)), f.clock.Now)
}

// reopen builds a new service on the stored data, as the API does when it starts again.
func (f *fixture) reopen(t *testing.T) {
	t.Helper()
	f.open()
	f.register(t, f.standardKinds()...)
}

func (f *fixture) register(t *testing.T, kinds ...Kind) {
	t.Helper()
	for _, k := range kinds {
		if err := f.service.Register(k); err != nil {
			t.Fatalf("Register(%s): %v", k.Name, err)
		}
	}
}

func (f *fixture) addTree(t *testing.T) {
	t.Helper()
	f.put(t, parentsTable, newRecord("P-1", "Parent one", ""))
	f.put(t, childrenTable, newRecord("C-1", "Child one", "P-1"), newRecord("C-2", "Child two", "P-1"))
	f.put(t, grandchildrenTable, newRecord("G-1", "Grandchild one", "C-1"),
		newRecord("G-2", "Grandchild two", "C-1"), newRecord("G-3", "Grandchild three", "C-2"))
}

func (f *fixture) records(table string) *fake.Table[testRecord] {
	return fake.TableOf(f.db, table, func(r testRecord) string { return r.ID })
}

func (f *fixture) put(t *testing.T, table string, records ...testRecord) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		return f.records(table).Put(ctx, records...)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func (f *fixture) stored(t *testing.T, table, id string) (testRecord, bool) {
	t.Helper()
	r, found, err := f.records(table).Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return r, found
}

func (f *fixture) allRecords(t *testing.T) map[string][]testRecord {
	t.Helper()
	all := map[string][]testRecord{}
	for _, table := range []string{parentsTable, childrenTable, grandchildrenTable} {
		records, err := f.records(table).List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		all[table] = records
	}
	return all
}

func (f *fixture) entries(t *testing.T) []Entry {
	t.Helper()
	entries, err := newFakeStore(f.db).Entries(t.Context())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return entries
}

func (f *fixture) binEntry(t *testing.T, id string) BinEntry {
	t.Helper()
	entry, found, err := newFakeStore(f.db).GetBinEntry(t.Context(), id)
	if err != nil || !found {
		t.Fatalf("GetBinEntry(%s) = found %v, %v", id, found, err)
	}
	return entry
}

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// appendRaw appends e as Purge and EmptyBin do, in a Write of its own.
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

// removeForGood removes a record as a permanent delete leaves it: not stored and in no entry of the bin.
func (f *fixture) removeForGood(t *testing.T, table, id string) {
	t.Helper()
	if err := f.records(table).Delete(t.Context(), id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func (f *fixture) lastEntry(t *testing.T) Entry {
	t.Helper()
	entries := f.entries(t)
	return entries[len(entries)-1]
}
