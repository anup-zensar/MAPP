package seed_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"mapp/internal/budgets"
)

// The demo global budget of 014 (FR-040, plan "Demo data"): GB-1001 "FY2026 Global marketing budget", 2026,
// 2,500,000.00 USD, owned by Carlos Mendes, allocated to FY2026 Lubricants Europe (BA-1001, 1,200,000.00) and
// FY2026 Mobility China (BA-1002, 800,000.00) — exactly their seeded budgets, so the plans are not changed.
func TestLoad_GlobalBudget_FR040(t *testing.T) {
	t.Parallel()
	db := loadedStore(t)
	d := loadDemo(t)
	var stored []budgets.GlobalBudget
	var allocations []budgets.Allocation
	read(t, db, func(ctx context.Context) error {
		repo := budgets.NewJSONRepository()
		var errB, errA error
		stored, errB = repo.ListBudgets(ctx)
		allocations, errA = repo.ListAllocations(ctx)
		return errors.Join(errB, errA)
	})
	names := map[string]string{}
	for _, p := range d.people {
		names[p.ID] = p.Name
	}
	budgetOf := map[string]string{}
	for _, p := range d.plans {
		names[p.ID] = p.Name
		if p.Budget != nil {
			budgetOf[p.ID] = p.Budget.String()
		}
	}

	if len(stored) != 1 {
		t.Fatalf("global budgets = %+v, want one", stored)
	}
	b := stored[0]
	got := fmt.Sprintf("%s %s %d %s %s %q v%d", b.ID, b.Name, b.Year, b.Amount.String(), names[b.OwnerID], b.Description, b.Version)
	want := `GB-1001 FY2026 Global marketing budget 2026 2500000.00 Carlos Mendes "Shared out to market plans by Global Marketing Ops." v1`
	if got != want {
		t.Errorf("global budget = %s\nwant %s", got, want)
	}
	var rows []string
	for _, a := range allocations {
		rows = append(rows, fmt.Sprintf("%s %s %s %s v%d", a.ID, a.GlobalBudgetID, names[a.PlanID], a.Amount.Format(), a.Version))
		if budgetOf[a.PlanID] != a.Amount.Amount.String() || a.Amount.Currency != "USD" {
			t.Errorf("allocation %s = %s, want the plan's budget %s USD", a.ID, a.Amount.Format(), budgetOf[a.PlanID])
		}
	}
	wantRows := []string{
		"BA-1001 GB-1001 FY2026 Lubricants Europe 1,200,000.00 USD v1",
		"BA-1002 GB-1001 FY2026 Mobility China 800,000.00 USD v1",
	}
	if !slices.Equal(rows, wantRows) {
		t.Errorf("allocations =\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(wantRows, "\n"))
	}
}
