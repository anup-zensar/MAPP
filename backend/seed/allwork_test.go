package seed_test

import (
	"context"
	"slices"
	"testing"

	"mapp/internal/allwork"
	"mapp/internal/allwork/allworktest"
)

// Priya Raman has two saved views, "Lubricants by month" (Budget & cost of FY2026 Lubricants Europe, by
// month) and "Declined activities" (the 2026 grid, Activity status Declined), neither her default; Daniel
// Okoro has none (FR-029, 010 plan "Demo data").
func TestSeed_SavedViews_FR029(t *testing.T) {
	t.Parallel()
	db := loadedStore(t)
	repo := allworktest.NewStore(db)
	read(t, db, func(ctx context.Context) error {
		priyas, err := repo.ListViews(ctx, "U-1002")
		if err != nil {
			return err
		}
		daniels, err := repo.ListViews(ctx, "U-1003")
		if err != nil {
			return err
		}
		if len(priyas) != 2 || len(daniels) != 0 {
			t.Fatalf("Priya has %d views, Daniel %d; want 2 and 0", len(priyas), len(daniels))
		}
		byMonth, declined := priyas[0], priyas[1]
		if byMonth.ID != "SV-1001" || byMonth.Name != "Lubricants by month" || byMonth.IsDefault ||
			byMonth.State.View != allwork.ViewBudget || byMonth.State.Year != 2026 || byMonth.State.Period != allwork.PeriodMonth ||
			len(byMonth.State.Filters.Plans) != 1 || byMonth.State.Filters.Plans[0].ID != "P-1001" ||
			byMonth.State.Filters.Plans[0].Label != "FY2026 Lubricants Europe" {
			t.Errorf("Lubricants by month = %+v", byMonth)
		}
		if declined.ID != "SV-1002" || declined.Name != "Declined activities" || declined.IsDefault ||
			declined.State.View != allwork.ViewGrid || !slices.Equal(declined.State.Filters.ActivityStatuses, []string{"declined"}) ||
			!slices.Equal(declined.State.Columns, allwork.DefaultColumns) {
			t.Errorf("Declined activities = %+v", declined)
		}
		return nil
	})
}
