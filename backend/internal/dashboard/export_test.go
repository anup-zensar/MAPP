package dashboard

import (
	"slices"
	"testing"
	"time"

	"mapp/internal/allwork"
	"mapp/internal/people"
	"mapp/internal/platform/date"
	"mapp/internal/platform/deck"
	"mapp/internal/platform/money"
)

func sampleInput(t *testing.T, kind Kind, plan string) *input {
	t.Helper()
	now := time.Date(2026, 10, 28, 9, 30, 0, 0, time.UTC)
	nodes := sampleNodes(t)
	in := &input{request: request{kind: kind, query: Query{Year: 2026, Currency: "USD"}, plan: plan}, now: now,
		today: date.Today(now), display: money.DisplayUSD, data: allwork.YearData{Nodes: nodes}}
	in.board = newBoard(nodes, in.keeps, true, []string{"P-1"})
	return in
}

func titles(d deck.Deck) []string {
	var list []string
	for _, p := range d.Pages {
		list = append(list, p.Title)
	}
	return list
}

// A plan's file: its title page with its figures, the months, the campaigns with their chart, and its activities;
// a plan whose finance the person may not see says so (FR-023, FR-026).
func TestPlanDeck_FR023(t *testing.T) {
	t.Parallel()
	beta := sampleInput(t, KindExecutive, "P-1").planDeck()

	want := []string{"Beta plan", "Planned and actual by month", "Campaigns", "Activities"}
	if got := titles(beta); !slices.Equal(got, want) {
		t.Fatalf("pages = %v, want %v", got, want)
	}
	activities := beta.Pages[3].Table.Rows
	alpha := sampleInput(t, KindExecutive, "P-2").planDeck()
	same(t, "plan file", texts("1,000.00 USD", true, 2, 3, "Over", "Approved", "200.00 USD", true, financeNote),
		texts(beta.Pages[0].Cards[0].Value, beta.Pages[2].Chart != nil, len(beta.Pages[2].Table.Rows), len(activities),
			activities[0].Cells[1], activities[0].Cells[2], activities[0].Cells[6], slices.Contains(alpha.Pages[0].Notes, financeNote),
			alpha.Pages[0].Cards[4].Note))
}

// A dashboard's file holds the screen's panels; the Regional one has no market page, and a plan's campaigns
// page appears when a plan is chosen (FR-024).
func TestDashboardDeck_FR024(t *testing.T) {
	t.Parallel()
	exec := sampleInput(t, KindExecutive, "")
	file := exec.dashboardDeck(exec.view(), exec.dashboardTitle())
	want := []string{"Executive dashboard", "By market", "By plan", "Planned and actual by month", "Campaigns most over budget",
		"Campaigns with the most left to plan", "Approvals", "Purchase requests", "Plan details"}
	if got := titles(file); !slices.Equal(got, want) {
		t.Errorf("executive pages = %v", got)
	}
	region := sampleInput(t, KindRegional, "")
	region.query.TeamID, region.query.PlanID = "T-1", "P-1"
	region.team = &people.TeamRef{ID: "T-1", Name: "Lubricants Europe"}
	region.board = newBoard(region.data.Nodes, region.keeps, true, nil)
	pages := titles(region.dashboardDeck(region.view(), region.dashboardTitle()))
	if slices.Contains(pages, "By market") || !slices.Contains(pages, "Campaigns") || pages[0] != "Regional dashboard – Lubricants Europe" {
		t.Errorf("regional pages = %v", pages)
	}
	if name := fileName("Plan: A/B", "28 Oct 2026"); name != "SP&A One – Plan- A-B – 28 Oct 2026" {
		t.Errorf("fileName = %q", name)
	}
}
