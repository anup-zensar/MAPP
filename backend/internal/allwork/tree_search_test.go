package allwork

import (
	"reflect"
	"testing"
)

// Search finds the text in the name, ID, external name or description of a record, ignoring letter case;
// the record's path is shown as context, the fields that matched are named, and a hit in a long text comes
// with an excerpt (FR-006, US2 scenario 1).
func TestSearch_MatchesNameIdExternalNameDescription_IgnoringCase_FR006(t *testing.T) { //nolint:gocyclo // 11: each search, then its match, its hit and its context rows
	t.Parallel()
	b := seedYear()
	tests := []struct {
		search, match string
		fields        []string
		excerpt       string
	}{
		{"launch FILM", "A-1001", []string{"name"}, ""},
		{"a-1004", "A-1004", []string{"id"}, ""},
		{"helix ULTRA", "A-1001", []string{"externalName", "description"}, "Helix Ultra film"},
	}
	for _, tt := range tests {
		r := view(b, Query{Search: tt.search})

		if got := matched(r, LevelActivity); !reflect.DeepEqual(got, []string{tt.match}) || r.Count.Shown != 1 || r.Count.Total != 10 {
			t.Errorf("%q: matching activities = %q, count %+v", tt.search, got, r.Count)
			continue
		}
		hit := rowOf(r.Rows, tt.match).SearchHit
		if hit == nil || !reflect.DeepEqual(hit.Fields, tt.fields) || hit.Excerpt != tt.excerpt {
			t.Errorf("%q: hit = %+v, want fields %q and excerpt %q", tt.search, hit, tt.fields, tt.excerpt)
		}
		for _, key := range []string{"P-1001", "C-1001"} {
			if rowOf(r.Rows, key).Role != RoleContext {
				t.Errorf("%q: %s is not a context row", tt.search, key)
			}
		}
	}
	long := "Phase one covers the planning; phase two is the regional rollout of the Helix Ultra campaign across " +
		"every market in the cluster, with a review after each wave."
	if got := excerptAround(long, "helix ultra"); got != "…regional rollout of the Helix Ultra campaign across every ma…" {
		t.Errorf("excerpt = %q", got)
	}
}

// A line item whose name, ID or description holds the text makes its activity a match, and is marked
// itself (FR-006).
func TestSearch_LineItemHitMakesActivityMatch_FR006(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{Search: "venue"})

	if got := matched(r, LevelActivity); !reflect.DeepEqual(got, []string{"A-1002"}) {
		t.Fatalf("matching activities = %q, want A-1002", got)
	}
	if hit := rowOf(r.Rows, "L-1003").SearchHit; hit == nil || !reflect.DeepEqual(hit.Fields, []string{"name"}) {
		t.Errorf("Venue hire hit = %+v", hit)
	}
	if other := rowOf(r.Rows, "L-1004"); other.SearchHit != nil || other.Role != RoleMatch {
		t.Errorf("Speaker fees = %+v: shown with its activity, without a hit", other)
	}
	if rowOf(r.Rows, "A-1002").SearchHit != nil {
		t.Errorf("the activity's own fields did not match")
	}
}

// A matching container shows everything inside it (FR-006): its campaigns, initiatives and activities meet
// the search through it.
func TestSearch_MatchingContainerShowsEverythingInside_FR006(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{Search: "RIMULA"})

	want := []string{"P-1001 context", "C-1002 match", "I-1003 match", "A-1005 match", "A-1005:unassigned match",
		"I-1004 match", "A-1006 match", "A-1006:unassigned match", "A-1007 match", "A-1007:unassigned match"}
	if got := rolesOf(r); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %q\nwant   %q", got, want)
	}
	if r.Count.Shown != 3 || rowOf(r.Rows, "C-1002").SearchHit == nil || rowOf(r.Rows, "I-1003").SearchHit != nil {
		t.Errorf("count %+v; only Rimula fleet itself has a hit", r.Count)
	}
	// Search and filters combine: inside a matching container only records that meet the filters match.
	declined := view(seedYear(), Query{Search: "rimula", ActivityStatuses: []string{"declined"}})
	if got := matched(declined, LevelActivity); !reflect.DeepEqual(got, []string{"A-1006"}) {
		t.Errorf("Rimula and Declined: matching activities = %q", got)
	}
}
