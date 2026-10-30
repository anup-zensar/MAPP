package app

import (
	"context"
	"testing"

	"mapp/internal/allwork"
	"mapp/internal/platform/date"
)

// The board reads purchase orders and payments for its own line items, so the year it starts from carries
// no GSAP figures: every stored transaction would otherwise be read on each dashboard (D-356).
func TestDashboardPlanning_YearWithoutFigures_036_FR002(t *testing.T) {
	t.Parallel()
	built, db := newSeededApp(t)
	var all, board allwork.YearData
	today := date.MustParse("2026-03-12")
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		if all, err = built.allWork.Year(ctx, 2026, today, nil); err != nil {
			return err
		}
		board, err = dashboardPlanning{allWorkSource: built.allWork.(allWorkSource)}.Year(ctx, 2026, today, nil)
		return err
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(board.Nodes) != len(all.Nodes) || board.FinanceAsOf == nil {
		t.Fatalf("board year: %d nodes, data as of %v; All work: %d nodes", len(board.Nodes), board.FinanceAsOf,
			len(all.Nodes))
	}
	for _, n := range board.Nodes {
		if n.Finance != nil {
			t.Errorf("%s %s carries GSAP figures: %+v", n.Level, n.Name, n.Finance)
		}
	}
}
