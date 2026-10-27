package campaign

import (
	"encoding/json"
	"reflect"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

// A campaign's completeness lists its own empty needed-to-publish fields in form order: Owner, Dates (one
// field), then the attributes Settings reports — "2 fields needed" without cluster and region (FR-014,
// US4 scenario 4, D-092).
func TestCampaignView_CompletenessOwnFields_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	bare := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Bare"})
	retail := f.createCampaign(t, meiLin, "P-1002", CampaignInput{Name: "Retail growth", OwnerID: ptr("U-1004"),
		StartDate: day("2026-01-01"), EndDate: day("2026-12-31")})

	want := []settings.MissingField{
		{Field: "ownerId", Label: "Owner"}, {Field: "startDate", Label: "Dates"},
		{Field: "attributes." + clusterID, Label: "Cluster"}, {Field: "attributes." + regionID, Label: "Region"},
	}
	if !reflect.DeepEqual(bare.Completeness.Missing, want) {
		t.Errorf("missing = %+v\nwant %+v", bare.Completeness.Missing, want)
	}
	if !reflect.DeepEqual(retail.Completeness.Missing, want[2:]) {
		t.Errorf("Retail growth missing = %+v, want Cluster and Region", retail.Completeness.Missing)
	}
}

// A region nobody can choose — the chosen cluster has no regions set up — is not counted as missing (spec
// edge case, D-034).
func TestCampaignView_RegionWithoutOptionsNotMissing_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", OwnerID: ptr("U-1002"),
		StartDate: day("2026-01-01"), EndDate: day("2026-12-31"),
		Attributes: settings.AttributeValues{clusterID: json.RawMessage(`["SO-NONE"]`)}})

	if len(view.Completeness.Missing) != 0 {
		t.Errorf("missing = %+v, want nothing", view.Completeness.Missing)
	}
}

// Items below needing attention count incomplete initiatives, activities and line items below; the plan's
// rollup lists the incomplete campaigns and initiatives (FR-014).
func TestViews_ItemsBelowNeedingAttention_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	complete := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Complete", OwnerID: ptr("U-1002"),
		StartDate: day("2026-01-01"), EndDate: day("2026-06-30")})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Without owner or dates"})
	f.activities.add(
		ActivitySummary{ID: "A-1", InitiativeID: complete.ID, Incomplete: true,
			IncompleteLineItems: []errs.RecordRef{{Type: "lineItem", ID: "L-1", Name: "Film"}, {Type: "lineItem", ID: "L-2", Name: "Edit"}}},
		ActivitySummary{ID: "A-2", InitiativeID: complete.ID},
	)

	view, _ := f.service.GetCampaign(t.Context(), priya, c.ID)

	if got := view.Completeness.ItemsBelowNeedingAttention; got != 4 { // 1 initiative, 1 activity, 2 line items
		t.Errorf("campaign: %d items below need attention, want 4", got)
	}
	if got := view.Initiatives[0].Completeness.ItemsBelowNeedingAttention; got != 3 {
		t.Errorf("initiative: %d items below, want 3", got)
	}
}
