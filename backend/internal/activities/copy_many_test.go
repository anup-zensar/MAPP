package activities_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/activities"
)

// A campaign copy copies all its initiatives' activities at once, yet the first initiative's A-1007, the highest
// id, gets the first new ID, as one copy per initiative gave it, and line items follow that order (028 FR-004).
func TestCopyMany_IDsInitiativeByInitiative_028_FR004(t *testing.T) {
	f := newFixture(t)
	const events = "I-1003"
	f.lineItems.facts["A-1007"] = activities.LineItemFacts{Count: 2}
	pairs := []activities.CopyPair{{FromID: events, ToID: retailEvents}, {FromID: digitalAndSocial, ToID: printAndOutdoor}}

	var counts activities.CopyCounts
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		counts, err = f.service.CopyMany(ctx, daniel, pairs, 0)
		return err
	})

	if err != nil || counts != (activities.CopyCounts{Activities: 3, LineItems: 2}) {
		t.Fatalf("CopyMany = %+v, %v; want 3 activities and 2 line items", counts, err)
	}
	partner := f.byName(t, retailEvents)["Partner co-marketing fund"]
	inPrint := f.byName(t, printAndOutdoor)
	launch, _ := f.stored(t, "A-1001")
	second, _ := f.stored(t, "A-1002")
	launchCopy, secondCopy := inPrint[launch.Name], inPrint[second.Name]
	if partner.ID == "" || launchCopy.ID == "" || secondCopy.ID == "" {
		t.Fatalf("copies %q, %q, %q: want all three", partner.ID, launchCopy.ID, secondCopy.ID)
	}
	want := []string{"A-1007 → " + partner.ID + " +0", "A-1001 → " + launchCopy.ID + " +0", "A-1002 → " + secondCopy.ID + " +0"}
	if diff := cmp.Diff(want, f.lineItems.copies); diff != "" {
		t.Errorf("line-item copies (-want +got):\n%s", diff)
	}
	inOrder := partner.ID < launchCopy.ID && launchCopy.ID < secondCopy.ID
	if !inOrder {
		t.Errorf("copy IDs %s, %s, %s; want them in the order of the initiatives", partner.ID, launchCopy.ID, secondCopy.ID)
	}
}
