package seed_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"mapp/internal/campaigns"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// The campaigns and initiatives of 005 data model §9, C-1001 … C-1004 and I-1001 … I-1008, with their
// plans, owners, dates and budgets.
func TestLoad_CampaignsAndInitiatives_FR002(t *testing.T) {
	d := loadDemo(t)
	var got []string
	for _, c := range d.campaigns {
		got = append(got, fmt.Sprintf("%s %s %s %s %s %s", c.ID, c.Name, c.PlanID, *c.OwnerID, dates(c.StartDate, c.EndDate), c.Budget.Format()))
		for _, i := range d.initiatives {
			if i.CampaignID == c.ID {
				got = append(got, fmt.Sprintf("  %s %s %s %s %s", i.ID, i.Name, *i.OwnerID, dates(i.StartDate, i.EndDate), i.Budget.Format()))
			}
		}
	}
	want := []string{
		"C-1001 Helix launch P-1001 U-1002 01 Jan – 31 Dec 2026 700,000.00 USD",
		"  I-1001 Digital & social U-1002 01 Jan – 30 Jun 2026 300,000.00 USD",
		"  I-1002 Print & outdoor U-1002 01 Jan – 31 Dec 2026 250,000.00 USD",
		"C-1002 Rimula fleet P-1001 U-1003 01 Feb – 31 Dec 2026 450,000.00 USD",
		"  I-1003 Events U-1003 01 Mar – 30 Sep 2026 200,000.00 USD",
		"  I-1004 Retail events U-1003 01 Jun – 31 Dec 2026 150,000.00 USD",
		"C-1003 Retail growth P-1002 U-1004 01 Jan – 31 Dec 2026 600,000.00 USD",
		"  I-1005 Station promotions U-1004 01 Mar – 30 Sep 2026 350,000.00 USD",
		"  I-1006 Fleet cards U-1004 no dates 200,000.00 USD",
		"C-1004 Brand platform P-1003 U-1001 01 Jan 2027 – 31 Mar 2028 300,000.00 USD",
		"  I-1007 Brand film U-1001 01 Jan – 30 Jun 2027 250,000.00 USD",
		"  I-1008 Sponsorships U-1001 01 Apr – 31 Dec 2027 100,000.00 USD",
		"C-1005 Helix UK & Ireland P-1004 U-1002 01 Jan – 31 Dec 2027 400,000.00 GBP",
		"  I-1009 UK U-1002 01 Jan – 31 Dec 2027 250,000.00 GBP",
		"  I-1010 Ireland U-1002 01 Jan – 31 Dec 2027 150,000.00 GBP",
	}
	if !slices.Equal(got, want) {
		t.Errorf("campaigns =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The figures the demo story shows follow from the seeded budgets: Helix launch keeps 150,000.00 for
// initiatives, FY2026 Lubricants Europe 50,000.00 for campaigns, Brand platform is over-allocated by
// 50,000.00 (005 data model §4.1, §9; D-015).
func TestLoad_BudgetsFlowDown_FR010(t *testing.T) {
	d := loadDemo(t)
	remaining := func(budget money.Money, children []money.Money) string {
		return budget.Amount.Sub(sumOf(children)).Format()
	}
	for _, c := range d.campaigns {
		var children []money.Money
		for _, i := range d.initiatives {
			if i.CampaignID == c.ID {
				children = append(children, *i.Budget)
			}
		}
		want := map[string]string{"C-1001": "150,000.00", "C-1002": "100,000.00", "C-1003": "50,000.00", "C-1004": "-50,000.00",
			"C-1005": "0.00"}[c.ID]
		if got := remaining(*c.Budget, children); got != want {
			t.Errorf("%s keeps %s for its initiatives, want %s", c.Name, got, want)
		}
	}
	for _, p := range d.plans {
		var children []money.Money
		for _, c := range d.campaigns {
			if c.PlanID == p.ID {
				children = append(children, *c.Budget)
			}
		}
		want := map[string]string{"P-1001": "50,000.00", "P-1002": "200,000.00", "P-1003": "200,000.00", "P-1004": "200,000.00"}[p.ID]
		if got := remaining(money.USD(*p.Budget), children); got != want {
			t.Errorf("%s keeps %s for its campaigns, want %s", p.Name, got, want)
		}
	}
}

// Initiatives lie inside their campaign's dates (R3); Brand platform ends after its plan's year (the R4
// warning); Fleet cards has no dates; Retail growth has no cluster and no region ("2 fields needed"), the
// other campaigns have both (005 data model §1–§2, §9).
func TestLoad_CampaignRules_R3_R4(t *testing.T) {
	d := loadDemo(t)
	byID := map[string]campaigns.Campaign{}
	for _, c := range d.campaigns {
		byID[c.ID] = c
	}
	for _, i := range d.initiatives {
		c := byID[i.CampaignID]
		if i.StartDate == nil {
			if i.Name != "Fleet cards" || i.EndDate != nil {
				t.Errorf("%s has no start date", i.Name)
			}
			continue
		}
		if i.StartDate.Before(*c.StartDate) || i.EndDate.After(*c.EndDate) || i.EndDate.Before(*i.StartDate) {
			t.Errorf("%s (%s) is not inside %s (%s)", i.Name, dates(i.StartDate, i.EndDate), c.Name, dates(c.StartDate, c.EndDate))
		}
	}
	if brand := byID["C-1004"]; !brand.EndDate.After(date.YearEnd(2027)) {
		t.Errorf("Brand platform ends %s, want after its plan's year", brand.EndDate)
	}

	cluster := d.attributeID(t, settings.LevelCampaign, "Cluster")
	region := d.attributeID(t, settings.LevelCampaign, "Region")
	for _, c := range d.campaigns {
		_, hasCluster := c.Attributes[cluster]
		_, hasRegion := c.Attributes[region]
		if hasCluster != (c.Name != "Retail growth") || hasRegion != hasCluster {
			t.Errorf("%s: cluster %v, region %v", c.Name, hasCluster, hasRegion)
		}
	}
}

// dates writes a date range as people read it, or "no dates".
func dates(start, end *date.Date) string {
	if start == nil || end == nil {
		return "no dates"
	}
	return date.FormatRange(*start, *end)
}

// sumOf adds the amounts of the money values.
func sumOf(values []money.Money) money.Amount {
	var total money.Amount
	for _, v := range values {
		total = total.Add(v.Amount)
	}
	return total
}

// The GBP campaign's budget is phased in four months of 2027 (016 data-model §8); every other record is not
// phased.
func TestLoad_CampaignPhasing_016(t *testing.T) {
	d := loadDemo(t)
	for _, c := range d.campaigns {
		var months []string
		for _, m := range c.BudgetPhasing.Months {
			months = append(months, m.Month+" "+m.Amount.Format())
		}
		want := []string(nil)
		method := "none"
		if c.ID == "C-1005" {
			want, method = []string{"2027-03 100,000.00", "2027-06 100,000.00", "2027-09 100,000.00", "2027-11 100,000.00"}, "custom"
		}
		if string(c.BudgetPhasing.Method) != method || !slices.Equal(months, want) {
			t.Errorf("%s phasing = %s %q", c.ID, c.BudgetPhasing.Method, months)
		}
	}
	for _, i := range d.initiatives {
		if i.BudgetPhasing.Method != "none" {
			t.Errorf("%s phasing = %+v", i.ID, i.BudgetPhasing)
		}
	}
}
