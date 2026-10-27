package history

import (
	"slices"
	"testing"
)

// Entries come newest first by their sequence number, which also orders entries saved in the same
// millisecond (011 FR-004).
func TestQueryLog_NewestFirstBySeq_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	first := f.logAs(t, priya, parentOne(ActionCreated, ""))
	second := f.logAs(t, priya, parentOne(ActionChanged, ""))
	third := f.logAs(t, priya, parentOne(ActionStatusChanged, "published"))

	assertPage(t, f.query(t, LogQuery{Limit: 50}), []Entry{third, second, first}, 3)
}

// Load more asks for the entries older than the last one it has (beforeSeq): pages follow each other
// without gaps or repeats, and the totals stay the same (011 FR-009).
func TestQueryLog_BeforeSeqPaging_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	var all []Entry
	for range 5 {
		all = append(all, f.logAs(t, priya, parentOne(ActionChanged, "")))
		f.logAs(t, daniel, childOne(ActionCreated)) // filtered out below
	}
	slices.Reverse(all)
	q := LogQuery{ActorIDs: []string{priya.ID}, Limit: 2}

	var got []Entry
	for page := range 3 {
		result := f.query(t, q)
		if result.Total != 5 || result.UnfilteredTotal != 10 {
			t.Errorf("page %d: total %d of %d, want 5 of 10", page+1, result.Total, result.UnfilteredTotal)
		}
		for _, item := range result.Items {
			got = append(got, item.Entry)
		}
		if len(result.Items) > 0 {
			q.BeforeSeq = result.Items[len(result.Items)-1].Seq
		}
	}

	assertEntries(t, got, all)
	q.BeforeSeq = all[len(all)-1].Seq // nothing is older
	if last := f.query(t, q); len(last.Items) != 0 || last.Total != 5 {
		t.Errorf("after the oldest entry = %v (total %d), want nothing (total 5)", pageIDs(last), last.Total)
	}
}
