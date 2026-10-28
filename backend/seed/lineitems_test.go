package seed_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/lineitem"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Every line item has its activity's owner, the finance budget year 2026 and finance coding from Shell's lists (D-325).
func TestLoad_LineItems_FR009(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	spendType := d.attributeID(t, settings.LevelLineItem, "Spend type")
	glCode := d.attributeID(t, settings.LevelLineItem, "GL code")
	activityByID := map[string]activity.Activity{}
	for _, a := range d.activities {
		activityByID[a.ID] = a
	}

	var got []string
	for _, l := range d.lineItems {
		var months []string
		for _, m := range l.Spread.Months {
			months = append(months, m.Month+"="+m.Amount.Format())
		}
		coding := []string{l.FinanceBudgetItemID}
		for _, attribute := range []string{spendType, glCode} {
			for _, id := range optionIDs(t, d.attributeByID[attribute], l.Attributes[attribute]) {
				coding = append(coding, d.label(id))
			}
		}
		got = append(got, fmt.Sprintf("%s %s %s %s %s %s[%s] swbs=%v %s", l.ID, activityByID[l.ActivityID].Name, l.Name,
			l.Currency, l.PlannedCost.Format(), l.Spread.Method, strings.Join(months, " "), l.RequestSwbs, strings.Join(coding, " / ")))
		year := 2026
		if strings.HasPrefix(activityByID[l.ActivityID].StartDate.String(), "2027") { // the GBP plan of 016
			year = 2027
		}
		if l.OwnerID != activityByID[l.ActivityID].OwnerID || l.FinanceBudgetYear == nil || *l.FinanceBudgetYear != year {
			t.Errorf("%s: owner %q, finance budget year %v", l.Name, l.OwnerID, l.FinanceBudgetYear)
		}
	}
	want := []string{
		"L-1001 Launch film Agency fee USD 50,000.00 evenly[] swbs=true FBI-1001 / ATL - Digital / 7250080 - Advertising - Digital",
		"L-1002 Launch film Film production EUR 46,000.00 evenly[] swbs=true ",
		"L-1003 Social always-on Venue hire GBP 15,800.00 firstMonth[] swbs=true FBI-1002 / BTL - Event / 7250150 - Sponsorship programmes",
		"L-1004 Social always-on Speaker fees USD 5,000.00 lastMonth[] swbs=false FBI-1002 / BTL - Event / 7250150 - Sponsorship programmes",
		"L-1005 Trade magazine ads Media booking USD 30,000.00 custom[2026-03=6,000.00 2026-04=4,000.00 2026-05=4,000.00 " +
			"2026-06=4,000.00 2026-07=4,000.00 2026-08=4,000.00 2026-09=4,000.00] swbs=true FBI-1001 / ATL - Digital / 7250080 - Advertising - Digital",
		"L-1006 Billboard winter wave Stand build EUR 9,200.00 evenly[] swbs=true FBI-1003 / BTL - Event / 7250150 - Sponsorship programmes",
		"L-1007 Grand opening Shanghai Station signage USD 1,000.00 custom[2026-04=450.00 2026-05=450.00] swbs=true FBI-1004 / BTL - Event",
		"L-1008 Grand opening Shanghai Fleet card promotion CNY 280,450.00 evenly[] swbs=true FBI-1005 / ATL - Digital / 7250080 - Advertising - Digital",
		"L-1009 UK launch event Venue GBP 15,000.00 evenly[] swbs=true  / BTL - Event / 7250150 - Sponsorship programmes",
		"L-1010 UK launch event Agency fee (Ireland) EUR 23,000.00 evenly[] swbs=true  / BTL - Agency Fees / 7250060 - Advertising - Corp/Agency/Production",
	}
	if !slices.Equal(got, want) {
		t.Errorf("line items =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// With the seeded rates, the demo story's unassigned planned cost holds: Launch film keeps 20,000.00 USD, Grand
// opening Shanghai's line items exceed its planned cost by 500.00 USD, and Fleet roadshow has no line items.
func TestLoad_UnassignedPlannedCost_US1(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	rates := d.currentRates()
	want := map[string]string{
		"Launch film": "20,000.00", "Grand opening Shanghai": "-500.00", "Social always-on": "55,000.00",
		"Trade magazine ads": "0.00", "Billboard winter wave": "35,000.00", "Fleet roadshow": "60,000.00",
	}
	for _, a := range d.activities {
		expected, listed := want[a.Name]
		if !listed {
			continue
		}
		var assigned []money.Amount
		for _, l := range d.lineItems {
			if l.ActivityID == a.ID {
				usd, err := rates.ToUSD(money.Money{Amount: l.PlannedCost, Currency: l.Currency})
				if err != nil {
					t.Fatalf("%s: %v", l.Name, err)
				}
				assigned = append(assigned, usd)
			}
		}
		if got := a.PlannedCost.Amount.Sub(money.Sum(assigned...)).Format(); got != expected {
			t.Errorf("%s: unassigned %s USD, want %s", a.Name, got, expected)
		}
	}
	for _, l := range d.lineItems {
		if l.Spread.Method != lineitem.SpreadCustom && len(l.Spread.Months) != 0 {
			t.Errorf("%s keeps months although it is spread %s", l.Name, l.Spread.Method)
		}
	}
}
