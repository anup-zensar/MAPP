package plan

import (
	"errors"
	"slices"
	"testing"

	"mapp/internal/platform/errs"
)

func TestList_BCPIRFilterAndMarker_013_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	bcpir := seedPlan(t, planSeed{"P-1001", "FY2026 Lubricants Europe", 2026, teamLubricants, priya.ID, "1200000.00", StatusPublished})
	bcpir.BCPIR = true
	f.put(t, bcpir, seedPlan(t, planSeed{"P-1002", "FY2026 Mobility China", 2026, teamMobility, "U-1004", "800000.00", StatusDraft}))
	for filter, want := range map[string][]string{"only": {"P-1001"}, "exclude": {"P-1002"}, "include": {"P-1001", "P-1002"}, "": {"P-1001", "P-1002"}} {
		list, err := f.service.List(t.Context(), priya, ListFilter{BCPIR: filter})
		if ids := itemIDs(list); err != nil || !slices.Equal(ids, want) {
			t.Errorf("bcpir=%q: %v, %v; want %v", filter, ids, err, want)
		}
		if err == nil && list.Items[0].BCPIR != (list.Items[0].ID == "P-1001") {
			t.Errorf("bcpir=%q: marker of %s = %v", filter, list.Items[0].ID, list.Items[0].BCPIR)
		}
	}
	_, err := f.service.List(t.Context(), priya, ListFilter{BCPIR: "yes"})
	if e, ok := errors.AsType[*errs.Error](err); !ok || len(e.Fields) != 1 || e.Fields[0].Field != "bcpir" {
		t.Errorf("bcpir=yes: %v, want a problem with bcpir", err)
	}
}
