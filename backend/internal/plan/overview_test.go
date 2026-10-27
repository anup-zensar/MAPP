package plan

import (
	"reflect"
	"testing"

	"mapp/internal/people"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Feature 009's Home reads every plan with the figures, completeness and contents of the plan page, in US
// dollars whatever the reader's display currency, so that it can add plans up before converting once
// (FR-012, FR-017, 009 D-010).
func TestOverviews_PlanPageFiguresInUSD_FR012_FR017(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	in := input("FY2026 Lubricants Europe")
	in.Budget = &money.Input{Amount: "1200000.00", Currency: "USD"}
	created := f.create(t, priya, in)
	planned := usd(t, "960000.00")
	retail := AttentionItem{Record: errs.RecordRef{Type: "campaign", ID: "C-1001", Name: "Retail growth"},
		Missing: []settings.MissingField{{Field: "attributes.SA-1003", Label: "Cluster"}}, BlocksPublish: true}
	f.addLevel(t, "campaign", map[string]Facts{created.ID: {
		Campaigns: 2, Initiatives: 4, Activities: 7, ApprovedActivities: 4, LineItems: 15,
		CampaignBudgets: usd(t, "1300000.00"), PlannedCost: &planned, NeedsAttention: []AttentionItem{retail},
	}})
	f.directory.rates[daniel.ID] = people.DisplayRate{Code: "EUR", Rate: "0.92"}
	page, err := f.service.Get(t.Context(), priya, created.ID) // priya reads in US dollars

	list, err2 := f.service.Overviews(t.Context(), daniel)

	if err != nil || err2 != nil || len(list) != 1 {
		t.Fatalf("Overviews = %d plans, %v, %v; want 1", len(list), err, err2)
	}
	got := list[0]
	if got.Plan.ID != created.ID || got.Plan.Name != "FY2026 Lubricants Europe" || got.Plan.OwnerID != priya.ID {
		t.Errorf("plan = %+v", got.Plan)
	}
	if !reflect.DeepEqual(got.Figures, page.Figures) || got.Figures.Currency != "USD" {
		t.Errorf("figures = %+v\nwant the plan page's in USD %+v", got.Figures, page.Figures)
	}
	if !reflect.DeepEqual(got.Completeness, page.Completeness) || !reflect.DeepEqual(got.Contents, page.Contents) ||
		!reflect.DeepEqual(got.Teams, page.Teams) {
		t.Errorf("completeness %+v, contents %+v, teams %+v; want the plan page's", got.Completeness, got.Contents, got.Teams)
	}
}
