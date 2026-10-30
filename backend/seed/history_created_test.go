package seed_test

import (
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/record"
)

// Every seeded planning record has one "created" entry by its owner (Alex Morgan when it has none), 14 to 10
// days before the seed time and after its parents', which its created line matches; the
// deleted initiative's three records, 12 days before.
func TestSeed_History_CreatedEntries_011_038_FR060_L8(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	entries, _ := demoLog(t, loadedStore(t))

	created := createdEntries(t, entries)
	records := seededRecords(d)
	for _, r := range records {
		checkCreatedEntry(t, created, r)
	}
	for _, id := range []string{"I-1011", "A-1013", "L-1011"} {
		e, found := created[id]
		if !found || e.Actor.Name != "Priya Raman" || e.At.Sub(loadedAt.Add(-12*day)) > 2*time.Minute || e.At.Before(loadedAt.Add(-12*day)) {
			t.Errorf("created entry of %s = %+v, want Priya Raman 12 days before", id, e)
		}
	}
	if want := len(records) + 3; len(created) != want {
		t.Errorf("%d created entries, want %d", len(created), want)
	}
}

type seededRecord struct {
	kind, id, owner string
	meta            record.Meta
}

// createdEntries are the created entries by record id, each one checked against the entries before it.
func createdEntries(t *testing.T, entries []history.Entry) map[string]history.Entry {
	t.Helper()
	created := map[string]history.Entry{}
	for _, e := range entries {
		if e.Action != history.ActionCreated {
			continue
		}
		if _, twice := created[e.RecordID]; twice {
			t.Errorf("%s: a second created entry", entryText(e))
		}
		created[e.RecordID] = e
		checkCreatedShape(t, created, e)
	}
	return created
}

func checkCreatedShape(t *testing.T, created map[string]history.Entry, e history.Entry) {
	t.Helper()
	if e.Verb != "created" || len(e.Changes) != 0 || e.Details != "" {
		t.Errorf("%s: verb %q, changes %v, details %q; want created, none, none", entryText(e), e.Verb, e.Changes, e.Details)
	}
	for _, parent := range e.Path {
		if p, found := created[parent.ID]; !found || p.Seq > e.Seq {
			t.Errorf("%s comes before the created entry of its %s %s", entryText(e), parent.Type, parent.ID)
		}
	}
}

// seededRecords are the seeded planning records with their owners' names, Alex Morgan for a record without one.
func seededRecords(d demoData) []seededRecord {
	owners := map[string]string{"": "Alex Morgan"}
	for _, p := range d.people {
		owners[p.ID] = p.Name
	}
	var records []seededRecord
	for _, p := range d.plans {
		records = append(records, seededRecord{"plan", p.ID, owners[p.OwnerID], p.Meta})
	}
	for _, c := range d.campaigns {
		records = append(records, seededRecord{"campaign", c.ID, owners[*c.OwnerID], c.Meta})
	}
	for _, i := range d.initiatives {
		records = append(records, seededRecord{"initiative", i.ID, owners[*i.OwnerID], i.Meta})
	}
	for _, a := range d.activities {
		records = append(records, seededRecord{"activity", a.ID, owners[a.OwnerID], a.Meta})
	}
	for _, l := range d.lineItems {
		records = append(records, seededRecord{"lineItem", l.ID, owners[l.OwnerID], l.Meta})
	}
	return records
}

func checkCreatedEntry(t *testing.T, created map[string]history.Entry, r seededRecord) {
	t.Helper()
	e, found := created[r.id]
	switch {
	case !found:
		t.Errorf("%s %s has no created entry", r.kind, r.id)
	case e.RecordType != r.kind || e.Actor.Name != r.owner:
		t.Errorf("%s, want a %s created by %s", entryText(e), r.kind, r.owner)
	case !withinCreationDays(e.At):
		t.Errorf("%s is not between 14 and 10 days before the seed time", entryText(e))
	case !r.meta.CreatedAt.Equal(e.At) || r.meta.CreatedBy.ID != e.Actor.ID:
		t.Errorf("%s %s: created %s by %s, its first History entry says %s", r.kind, r.id, r.meta.CreatedAt,
			r.meta.CreatedBy.Name, entryText(e))
	}
}

func withinCreationDays(at time.Time) bool {
	return !at.Before(loadedAt.Add(-14*day)) && !at.After(loadedAt.Add(-10*day))
}
