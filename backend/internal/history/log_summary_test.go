package history

import (
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// The summary cards count over the whole log, for the boundaries the browser sends in the person's time
// zone (011 FR-002, D-179): changes since midnight, since Monday, the different people this week, and
// deletes, permanent deletes and emptied recycle bins this week.
func TestLogSummary_Counts_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	weekStart := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC) // Monday: 12 Mar 2026 is a Thursday
	todayStart := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	f.clock.now = weekStart.Add(-time.Minute) // Sunday 23:59, last week
	f.logAs(t, mei, parentOne(ActionCreated, ""))
	f.logAs(t, mei, LogInput{Kind: "grandchild", RecordID: "G-9", RecordName: "Gone", ParentID: "C-1", Action: ActionDeleted})
	f.clock.now = weekStart // Monday 00:00
	f.logAs(t, priya, parentOne(ActionChanged, ""))
	f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-9", RecordName: "Gone", ParentID: "C-1", Action: ActionDeletedPermanently})
	f.clock.now = todayStart.Add(-time.Millisecond) // yesterday
	f.logAs(t, daniel, parentOne(ActionChanged, ""))
	f.clock.now = todayStart
	f.logAs(t, daniel, childOne(ActionCreated))
	f.moveToBin(t, "grandchild", "G-3")
	f.appendRaw(t, priya, Entry{Action: ActionEmptiedRecycleBin, Verb: verbEmptiedRecycleBin,
		RecordType: emptiedBinType, RecordTypeLabel: "recycle bin", RecordName: "Recycle bin", Details: "1 item"})

	got := f.summary(t, todayStart, weekStart)

	want := [5]int{3, 6, 2, 3, 8} // today, this week, people this week, deletions this week, total
	if counts := [5]int{got.ChangesToday, got.ChangesThisWeek, got.PeopleActiveThisWeek, got.DeletionsThisWeek, got.Total}; counts != want {
		t.Errorf("today, week, people, deletions, total = %v, want %v", counts, want)
	}
}

// The summary does not follow the filters: it always counts the whole log, every person, kind and action
// (011 FR-002), so the cards stay the same while the list below them is filtered.
func TestLogSummary_IgnoresFilters_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.logAs(t, priya, parentOne(ActionCreated, ""))
	f.logAs(t, daniel, childOne(ActionCreated))
	f.logAs(t, mei, LogInput{Kind: "attachment", RecordID: "X-1", RecordName: "Unknown kind", Action: ActionCreated, ParentID: "P-1"}, f.withKind(t, "attachment"))
	filtered := f.query(t, LogQuery{ActorIDs: []string{priya.ID}, Limit: 50})

	got := f.summary(t, start.Add(-9*time.Hour), start.Add(-81*time.Hour))

	if filtered.Total != 1 || got.ChangesToday != 3 || got.ChangesThisWeek != 3 || got.PeopleActiveThisWeek != 3 || got.Total != 3 {
		t.Errorf("summary = %+v while the filtered list has %d; want every entry counted", got, filtered.Total)
	}
}

// The summary offers the filter choices (011 FR-003): everyone with an entry, the record types in the log
// with their latest label and count (not the emptied recycle bin), and the plans the log names, as a record
// or at the top of a path, with their latest names — each sorted by name.
func TestLogSummary_Facets_FR003(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	plans := f.testKind("plan", "plans", "", "plans")
	campaigns := f.testKind("campaign", "campaigns", "plan", "campaigns")
	f.register(t, plans, campaigns)
	f.put(t, "plans", newRecord("P-1001", "FY2026 Lubricants Europe", ""), newRecord("P-1002", "a small plan", ""))
	f.put(t, "campaigns", newRecord("C-1001", "Helix launch", "P-1001"))
	f.logAs(t, priya, LogInput{Kind: "plan", RecordID: "P-1001", RecordName: "FY2026 Lubricants", Action: ActionCreated})
	f.logAs(t, daniel, LogInput{Kind: "plan", RecordID: "P-1002", RecordName: "a small plan", Action: ActionCreated})
	f.logAs(t, mei, LogInput{Kind: "campaign", RecordID: "C-1001", RecordName: "Helix launch", Action: ActionCreated})
	f.logAs(t, actor.Actor{ID: daniel.ID, Name: "Daniel O."}, LogInput{Kind: "campaign", RecordID: "C-1001", RecordName: "Helix launch", Action: ActionChanged})
	f.appendRaw(t, priya, Entry{Action: ActionEmptiedRecycleBin, Verb: verbEmptiedRecycleBin,
		RecordType: emptiedBinType, RecordTypeLabel: "recycle bin", RecordName: "Recycle bin"})
	f.appendRaw(t, priya, Entry{Action: ActionDeleted, Verb: "deleted", RecordType: "campaign", RecordTypeLabel: "marketing campaign",
		RecordID: "C-1002", RecordName: "Pilot", Path: []RecordRef{{Type: "plan", ID: "P-1003", Name: "Old plan"}}})

	got := f.summary(t, start, start)

	wantPeople := []actor.Actor{{ID: daniel.ID, Name: "Daniel O."}, mei, priya}
	wantTypes := []RecordTypeCount{{Type: "campaign", Label: "marketing campaign", Count: 3}, {Type: "plan", Label: "plan", Count: 2}}
	wantPlans := []RecordRef{
		{Type: "plan", ID: "P-1002", Name: "a small plan"},
		{Type: "plan", ID: "P-1001", Name: "FY2026 Lubricants Europe"},
		{Type: "plan", ID: "P-1003", Name: "Old plan"},
	}
	if !reflect.DeepEqual(got.People, wantPeople) || !reflect.DeepEqual(got.RecordTypes, wantTypes) || !reflect.DeepEqual(got.Plans, wantPlans) {
		t.Errorf("facets =\n%s\n%s\n%s\nwant\n%s\n%s\n%s", asJSON(got.People), asJSON(got.RecordTypes), asJSON(got.Plans),
			asJSON(wantPeople), asJSON(wantTypes), asJSON(wantPlans))
	}
}

// Both boundaries are required, and the week cannot start after today; both problems are reported at once
// (011 FR-002).
func TestLogSummary_Validation(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	reload := "Reload the page and try again."

	_, missing := f.service.LogSummary(t.Context(), priya, time.Time{}, time.Time{})
	_, reversed := f.service.LogSummary(t.Context(), priya, start, start.Add(time.Second))

	assertError(t, missing, errs.Validation(errs.FieldError{Field: "todayStart", Message: reload}, errs.FieldError{Field: "weekStart", Message: reload}))
	assertError(t, reversed, errs.Validation(errs.FieldError{Field: "weekStart", Message: reload}))
}
