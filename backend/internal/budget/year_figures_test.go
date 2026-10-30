package budget

import (
	"fmt"
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// The dashboards' global budgets of a year are those of the Global budgets page, by name, with the same figures;
// another year's budget is not among them (D-356).
func TestYearFigures_AsTheGlobalBudgetsPage_036_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.demo(t)
	f.create(t, GlobalBudgetInput{Name: "Asia reserve", Year: ptr(2026), Amount: &money.Input{Amount: "100.00", Currency: "USD"}, OwnerID: carlos.ID})
	f.create(t, GlobalBudgetInput{Name: "FY2027 Global", Year: ptr(2027), Amount: &money.Input{Amount: "1.00", Currency: "USD"}, OwnerID: carlos.ID})

	got, err := f.service.YearFigures(t.Context(), 2026)
	page, errPage := f.service.List(t.Context(), carlos, ptr(2026))

	if err != nil || errPage != nil {
		t.Fatalf("YearFigures: %v, List: %v", err, errPage)
	}
	if !slices.Equal(texts(got), texts(page.Items)) || len(got) != 2 {
		t.Errorf("YearFigures = %v, want the page's %v", texts(got), texts(page.Items))
	}
}

func texts(list []Summary) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, fmt.Sprintf("%s %s %d: %s", s.ID, s.Name, s.AllocationCount, figuresText(s.Figures)))
	}
	return out
}
