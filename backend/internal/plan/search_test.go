package plan

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// fakeSearcher holds the records below the plans and matches them as the app's searcher does.
type fakeSearcher struct {
	records []SearchMatch
}

func (f fakeSearcher) Matching(_ context.Context, planIDs []string, lowerText string) ([]SearchMatch, error) {
	found := []SearchMatch{}
	for _, r := range f.records {
		holds := strings.Contains(strings.ToLower(r.Name), lowerText) || strings.Contains(strings.ToLower(r.ID), lowerText)
		if holds && slices.Contains(planIDs, r.PlanID) {
			found = append(found, r)
		}
	}
	return found, nil
}

func searchFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	f.putDemoPlans(t)
	f.service.UseSearcher(fakeSearcher{records: []SearchMatch{
		{Level: LevelLineItem, ID: "L-1001", Name: "Venue hire", ParentID: "A-1001", PlanID: "P-1002"},
		{Level: LevelActivity, ID: "A-1001", Name: "Venue tour", ParentID: "I-1001", PlanID: "P-1002"},
		{Level: LevelCampaign, ID: "C-1001", Name: "Venues", ParentID: "P-1002", PlanID: "P-1002"},
		{Level: LevelActivity, ID: "A-1002", Name: "Social always-on", ParentID: "I-1001", PlanID: "P-1002"},
		{Level: LevelLineItem, ID: "L-1009", Name: "Venue deposit", ParentID: "A-1009", PlanID: "P-1001"},
	}})
	return f
}

func TestSearch_TheRecordsBelowThePlanTopDown_039_US1_FR005(t *testing.T) {
	t.Parallel()
	f := searchFixture(t)

	got, err := f.service.Search(t.Context(), priya, "P-1002", "  VENUE ")

	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := []string{"campaign C-1001 Venues", "activity A-1001 Venue tour", "lineItem L-1001 Venue hire"}
	var texts []string
	for _, m := range got.Items {
		texts = append(texts, m.Level+" "+m.ID+" "+m.Name)
	}
	if !slices.Equal(texts, want) {
		t.Errorf("matches = %q, want %q", texts, want)
	}
}

func TestSearch_RefusesNoTextOrTooMuch_039_FR005(t *testing.T) {
	t.Parallel()
	f := searchFixture(t)

	_, empty := f.service.Search(t.Context(), priya, "P-1002", "   ")
	_, long := f.service.Search(t.Context(), priya, "P-1002", strings.Repeat("x", 201))
	_, gone := f.service.Search(t.Context(), priya, "P-9999", "venue")

	assertError(t, empty, problems("q", msgSearchEmpty))
	assertError(t, long, problems("q", msgSearchTooLong))
	if gone == nil || !strings.Contains(gone.Error(), planGoneMessage) {
		t.Errorf("unknown plan: error = %v, want %q", gone, planGoneMessage)
	}
}

func TestList_SearchFindsAPlanByItsContents_039_US2_FR006_FR007(t *testing.T) {
	t.Parallel()
	f := searchFixture(t)

	byContents, err := f.service.List(t.Context(), priya, ListFilter{Q: "venue"})
	if err != nil {
		t.Fatalf("List(venue): %v", err)
	}
	byName, err := f.service.List(t.Context(), priya, ListFilter{Q: "lubricants"})
	if err != nil {
		t.Fatalf("List(lubricants): %v", err)
	}

	if ids := itemIDs(byContents); !slices.Equal(ids, []string{"P-1001", "P-1002"}) {
		t.Fatalf("venue: plans %q, want P-1001 and P-1002", ids)
	}
	want := &FoundIn{Match: SearchMatch{Level: LevelCampaign, ID: "C-1001", Name: "Venues", ParentID: "P-1002",
		PlanID: "P-1002"}, More: 2}
	if got := byContents.Items[1].FoundIn; !reflect.DeepEqual(got, want) {
		t.Errorf("P-1002 found in %+v, want %+v", got, want)
	}
	if ids := itemIDs(byName); !slices.Equal(ids, []string{"P-1001"}) || byName.Items[0].FoundIn != nil {
		t.Errorf("lubricants: plans %q, found in %+v; want P-1001 by its own name", ids, byName.Items[0].FoundIn)
	}
}

func TestList_SearchKeepsTheOtherFilters_039_FR009(t *testing.T) {
	t.Parallel()
	f := searchFixture(t)

	published, err := f.service.List(t.Context(), priya, ListFilter{Q: "venue", Status: "published"})

	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ids := itemIDs(published); !slices.Equal(ids, []string{"P-1001"}) {
		t.Errorf("published venue plans = %q, want only P-1001", ids)
	}
}
