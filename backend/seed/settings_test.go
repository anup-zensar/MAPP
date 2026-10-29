package seed_test

import (
	"fmt"
	"slices"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// The Settings seed: 5 currencies with rates, 6 activity types with default planned costs, 2 models with rates and
// allowed types (in type-list order), 2026's 2 finance budgets of 3 items each and 2027's of 2 (038 FR-061, L-30),
// 35 Shell attributes with 162 options.
func TestLoad_Settings_FR032_038_FR061_L30(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)

	// The current rate of each currency is its latest dated rate on or before the seed day.
	current := map[string]settings.ExchangeRate{}
	for _, r := range d.rates {
		if latest, found := current[r.Currency]; r.Date.Compare(date.Today(loadedAt)) <= 0 && (!found || r.Date.After(latest.Date)) {
			current[r.Currency] = r
		}
	}
	var currencies []string
	for _, c := range d.currencies {
		rate := "1"
		if r, found := current[c.Code]; found {
			rate = r.Rate.String()
		}
		currencies = append(currencies, c.Code+" "+rate+" "+c.Symbol)
		if !c.Active || c.CreatedBy != actor.System || c.Rate != (money.Rate{}) {
			t.Errorf("currency %s: active %v, created by %v, legacy rate %v", c.Code, c.Active, c.CreatedBy, c.Rate)
		}
	}
	if want := []string{"CNY 7.1 ¥", "EUR 0.92 €", "GBP 0.79 £", "INR 83.5 ₹", "USD 1 $"}; !slices.Equal(currencies, want) {
		t.Errorf("currencies = %q, want %q", currencies, want)
	}

	var types []string
	for _, at := range d.activityTypes {
		text := at.ID + " " + at.Name
		if at.DefaultPlannedCost != nil {
			text += " " + at.DefaultPlannedCost.Format()
		}
		types = append(types, text)
	}
	wantTypes := []string{
		"AT-1001 Agency fees – Creative", "AT-1002 Event 25,000.00 USD", "AT-1003 Digital media 10,000.00 USD",
		"AT-1004 Content production", "AT-1005 Sponsorship 50,000.00 USD", "AT-1006 MOBILITY – Long-term contract",
	}
	if !slices.Equal(types, wantTypes) {
		t.Errorf("activity types = %q, want %q", types, wantTypes)
	}

	if len(d.models) != 2 {
		t.Fatalf("%d models, want 2", len(d.models))
	}
	lubricants, mobility := d.models[0], d.models[1]
	if lubricants.ID != "MOD-1001" || lubricants.Name != "Lubricants 1.0" || lubricants.InqToMqlRate.String() != "2.00" ||
		lubricants.MqlToDealRate.String() != "5.00" || lubricants.AverageDealSize.Format() != "10,000.00 USD" ||
		!slices.Equal(lubricants.ActivityTypeIDs, []string{"AT-1001", "AT-1004", "AT-1003", "AT-1002", "AT-1005"}) {
		t.Errorf("Lubricants 1.0 = %+v", lubricants)
	}
	if mobility.ID != "MOD-1002" || mobility.Name != "Mobility 1.0" || mobility.InqToMqlRate.String() != "3.00" ||
		mobility.MqlToDealRate.String() != "4.00" || mobility.AverageDealSize.Format() != "5,000.00 USD" ||
		!slices.Equal(mobility.ActivityTypeIDs, []string{"AT-1001", "AT-1004", "AT-1003", "AT-1002", "AT-1006", "AT-1005"}) {
		t.Errorf("Mobility 1.0 = %+v", mobility)
	}

	var budgets []string
	for _, b := range d.budgets {
		budgets = append(budgets, fmt.Sprintf("%s %s %d", b.ID, b.Name, b.Year))
	}
	want := []string{"FB-1001 2026 Lubricants Budget 2026", "FB-1002 2026 Mobility Budget 2026",
		"FB-1003 2027 Lubricants Budget 2027"} // 038 FR-061 (L-30): the FY2027 line items' budget
	if !slices.Equal(budgets, want) {
		t.Errorf("finance budgets = %q, want %q", budgets, want)
	}
	var items []string
	for _, i := range d.items {
		items = append(items, i.ID+" "+i.BudgetID+" "+i.Name+" "+i.Code)
	}
	wantItems := []string{
		"FBI-1001 FB-1001 Europe – Digital LUB-EU-DIG", "FBI-1002 FB-1001 Europe – Events LUB-EU-EVT",
		"FBI-1003 FB-1001 Europe – Brand LUB-EU-BRD", "FBI-1004 FB-1002 China – Retail MOB-CN-RET",
		"FBI-1005 FB-1002 China – Fleet MOB-CN-FLT", "FBI-1006 FB-1002 Delhi MOB-IN-DEL",
		"FBI-1007 FB-1003 UK & Ireland – Events LUB-UKI-EVT", "FBI-1008 FB-1003 UK & Ireland – Agency LUB-UKI-AGY",
	}
	if !slices.Equal(items, wantItems) {
		t.Errorf("finance budget items = %q, want %q", items, wantItems)
	}

	if len(d.attributes) != 56 || len(d.options) != 185 {
		t.Errorf("%d Shell attributes and %d options, want 56 and 185 (032 FR-006)", len(d.attributes), len(d.options))
	}
}

// Business unit (SA-1001) is the plan's business-unit attribute; markings fit their level; splits only at activity and
// line-item level; each level's positions run 1…n; sections and conditions are those of the seed table.
func TestLoad_ShellAttributes_FR017_FR023(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	businessUnit := d.attributes[0]
	purchaseRoles := []string{settings.RoleCompanyCode, settings.RoleCostCentre, settings.RoleGLCode, settings.RolePRType,
		settings.RoleShortText, settings.RoleVendor, settings.RoleVendorCode, settings.RoleVendorEmail, settings.RoleEmailGroup,
		settings.RolePRRecipients} // 032 FR-006
	if businessUnit.ID != "SA-1001" || businessUnit.Label != "Business unit" || businessUnit.Role != settings.RoleBusinessUnit ||
		businessUnit.Level != settings.LevelPlan || businessUnit.Marking != settings.MarkingNeededToPublish {
		t.Errorf("Business unit = %+v", businessUnit)
	}
	var buOptions []string
	for _, o := range d.options {
		if o.AttributeID == businessUnit.ID {
			buOptions = append(buOptions, o.Label)
		}
	}
	if !slices.Equal(buOptions, []string{"Mobility", "Lubricants"}) {
		t.Errorf("business units = %q", buOptions)
	}

	positions := map[settings.Level]int{}
	counts := map[settings.Level]int{}
	conditional := map[string][]string{}
	for _, a := range d.attributes {
		positions[a.Level]++
		counts[a.Level]++
		if a.Position != positions[a.Level] {
			t.Errorf("%s/%s is at position %d, want %d", a.Level, a.Label, a.Position, positions[a.Level])
		}
		publishLevel := a.Level == settings.LevelPlan || a.Level == settings.LevelCampaign || a.Level == settings.LevelInitiative
		switch {
		case a.Marking == settings.MarkingNeededToPublish && !publishLevel,
			a.Marking == settings.MarkingNeededToSubmit && publishLevel,
			a.Type == settings.TypeYesNo && a.Marking != settings.MarkingOptional,
			a.Type == settings.TypeMultipleChoiceSplit && publishLevel:
			t.Errorf("%s/%s: type %s and marking %s do not fit its level", a.Level, a.Label, a.Type, a.Marking)
		}
		if a.Role != "" && a.ID != businessUnit.ID && !slices.Contains(purchaseRoles, a.Role) {
			t.Errorf("%s has role %q", a.Label, a.Role)
		}
		for _, optionID := range a.Condition {
			conditional[d.label(optionID)] = append(conditional[d.label(optionID)], a.Section+": "+a.Label)
		}
		if !a.Active || a.Condition == nil {
			t.Errorf("%s: active %v, condition %v", a.Label, a.Active, a.Condition)
		}
	}
	wantCounts := map[settings.Level]int{
		settings.LevelPlan: 2, settings.LevelCampaign: 5, settings.LevelInitiative: 4, settings.LevelActivity: 20, settings.LevelLineItem: 4,
	}
	for level, want := range wantCounts {
		if counts[level] != want {
			t.Errorf("%d %s-level attributes, want %d", counts[level], level, want)
		}
	}
	wantLubricants := []string{
		"Lubricants attributes: Brands", "Lubricants attributes: Global key accounts", "Lubricants attributes: Sector",
		"Lubricants attributes: Sub-sector", "Lubricants attributes: Product launch",
	}
	wantMobility := []string{
		"Shell attributes: Strategic Marketing Plan (SMP)", "Mobility attributes: Function (Retail)",
		"Mobility attributes: Focus area (Retail)", "Mobility attributes: Function (Mobility)",
		"Mobility attributes: Focus area (Mobility)", "Mobility attributes: Innovation",
	}
	if !slices.Equal(conditional["Lubricants"], wantLubricants) || !slices.Equal(conditional["Mobility"], wantMobility) {
		t.Errorf("conditions = %q", conditional)
	}
}

// Every Region is linked to a Cluster (Retail – India has none, D-034). GL code depends on nothing: no source gives
// Shell's spend type → GL code mapping (D-325).
func TestLoad_DependentOptions_FR022(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	links := map[string][]string{} // "<attribute>: <option>" → parent option labels
	for _, o := range d.options {
		attribute := d.attributeByID[o.AttributeID]
		if attribute.ParentAttributeID == "" {
			if len(o.ParentOptionIDs) != 0 {
				t.Errorf("%s '%s' is linked although %s depends on nothing", attribute.Label, o.Label, attribute.Label)
			}
			continue
		}
		var parents []string
		for _, p := range o.ParentOptionIDs {
			if d.optionByID[p].AttributeID != attribute.ParentAttributeID {
				t.Errorf("%s '%s' is linked to %s, not an option of its parent", attribute.Label, o.Label, p)
			}
			parents = append(parents, d.label(p))
		}
		links[attribute.Label+": "+o.Label] = parents
	}

	region := d.attributeByID[d.attributeID(t, settings.LevelCampaign, "Region")]
	if d.attributeByID[region.ParentAttributeID].Label != "Cluster" {
		t.Errorf("Region depends on %s, want Cluster", region.ParentAttributeID)
	}
	if glCode := d.attributeByID[d.attributeID(t, settings.LevelLineItem, "GL code")]; glCode.ParentAttributeID != "" {
		t.Errorf("GL code depends on %s, want nothing", glCode.ParentAttributeID)
	}
	want := map[string][]string{
		"Region: LUBES – DACH":            {"Lubricants – Europe"},
		"Region: FLEET – SOUTH CHINA":     {"Fleet – China"},
		"Region: LUBES – SOUTH EAST ASIA": {"Lubricants – Asia Pacific"},
		"Region: RETAIL – CHINA":          {"Retail – China"},
		"Region: LUBES – UK & IRELAND":    {"Lubricants – Europe"},
		"Region: LUBES – BENELUX":         {"Lubricants – Europe"},
		"Region: FLEET – NORTH CHINA":     {"Fleet – China"},
	}
	if len(links) != len(want) {
		t.Errorf("%d linked options, want %d: %q", len(links), len(want), links)
	}
	for option, parents := range want {
		if !slices.Equal(links[option], parents) {
			t.Errorf("%s is linked to %q, want %q", option, links[option], parents)
		}
	}
	for option, parents := range links {
		for _, p := range parents {
			if p == "Retail – India" {
				t.Errorf("%s is linked to Retail – India, which has no regions (D-034)", option)
			}
		}
	}
}

// The simulated GSAP rate history of 016 data-model §8: one import RI-1001 by MAPP and, for each currency
// but USD, a GSAP-imported rate for each of the 30 days before the seed day, swinging −0.6 % … +0.6 % around the
// seeded rate, and the seeded rate on the seed day — so the current rates stay the seeded ones (D-274).
func TestLoad_RateHistory_016_D274(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	today := date.Today(loadedAt)

	if len(d.rateImports) != 1 {
		t.Fatalf("%d rate imports, want 1", len(d.rateImports))
	}
	imp := d.rateImports[0]
	if imp.ID != "RI-1001" || imp.FileName != "gsap_rates_seed.csv" || imp.Feed != "gsapDailyRatesFile" || imp.Added != 124 ||
		imp.FirstDate == nil || *imp.FirstDate != today.AddDays(-30) || imp.LastDate == nil || *imp.LastDate != today ||
		imp.CreatedBy != actor.System {
		t.Errorf("rate import = %+v", imp)
	}
	if len(d.rates) != 124 {
		t.Fatalf("%d exchange rates, want 124 (31 for each of EUR, GBP, CNY, INR)", len(d.rates))
	}
	first, last := d.rates[0], d.rates[30]
	if first.ID != "XR-1001" || first.Currency != "EUR" || first.Date != today.AddDays(-30) || first.Rate.String() != "0.91448" ||
		first.Source != settings.RateGSAPImport || first.ImportID != "RI-1001" {
		t.Errorf("first rate = %+v, want EUR 0.91448 (0.92 − 0.6 %%) 30 days before the seed day", first)
	}
	if last.Currency != "EUR" || last.Date != today || last.Rate.String() != "0.92" {
		t.Errorf("EUR's rate on the seed day = %+v, want 0.92", last)
	}
	for _, r := range d.rates {
		if r.Date.After(today) || r.Currency == "USD" {
			t.Errorf("rate %s: %s on %s", r.ID, r.Currency, r.Date)
		}
	}
}
