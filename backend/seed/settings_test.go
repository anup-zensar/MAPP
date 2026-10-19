package seed_test

import (
	"slices"
	"testing"

	"northstar/internal/platform/actor"
	"northstar/internal/settings"
)

// The Settings seed of 002 data model §13: 5 currencies with their rates, 6 activity types with their
// default planned costs, 2 models with their rates and allowed types (in the order of the type list), 2
// finance budgets of 2026 with 3 items each, 35 Shell attributes with 110 options (FR-032).
func TestLoad_Settings_FR032(t *testing.T) {
	d := loadDemo(t)

	var currencies []string
	for _, c := range d.currencies {
		currencies = append(currencies, c.Code+" "+c.Rate.String()+" "+c.Symbol)
		if !c.Active || c.RateChangedAt != loadedAt || c.RateChangedBy != actor.System {
			t.Errorf("currency %s: active %v, rate changed %v by %v", c.Code, c.Active, c.RateChangedAt, c.RateChangedBy)
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
		budgets = append(budgets, b.ID+" "+b.Name)
		if b.Year != 2026 {
			t.Errorf("budget %s of %d, want 2026", b.ID, b.Year)
		}
	}
	if want := []string{"FB-1001 2026 Lubricants Budget", "FB-1002 2026 Mobility Budget"}; !slices.Equal(budgets, want) {
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
	}
	if !slices.Equal(items, wantItems) {
		t.Errorf("finance budget items = %q, want %q", items, wantItems)
	}

	if len(d.attributes) != 35 || len(d.options) != 110 {
		t.Errorf("%d Shell attributes and %d options, want 35 and 110", len(d.attributes), len(d.options))
	}
}

// The Shell attributes follow the rules of 002 data model §4: Business unit (SA-1001) is the plan's
// business-unit attribute; markings fit their level; splits only at activity and line-item level; each
// level's positions run 1…n; the sections and conditions are those of the seed table (FR-016 – FR-023).
func TestLoad_ShellAttributes_FR017_FR023(t *testing.T) {
	d := loadDemo(t)
	businessUnit := d.attributes[0]
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
		if a.Role != "" && a.ID != businessUnit.ID {
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

// Dependent options are linked to their parent's options: every Region to a Cluster (Retail – India has
// none, D-034), every GL code to one or more spend types, BTL – Event to exactly one GL code (FR-022).
func TestLoad_DependentOptions_FR022(t *testing.T) {
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
	glCode := d.attributeByID[d.attributeID(t, settings.LevelLineItem, "GL code")]
	if d.attributeByID[glCode.ParentAttributeID].Label != "Spend type" {
		t.Errorf("GL code depends on %s, want Spend type", glCode.ParentAttributeID)
	}
	want := map[string][]string{
		"Region: LUBES – DACH":                      {"Lubricants – Europe"},
		"Region: FLEET – SOUTH CHINA":               {"Fleet – China"},
		"GL code: 7250080 – Advertising digital":    {"Digital – Platform development", "Digital – Paid media", "Agency – Creative"},
		"GL code: 7250040 – Events and sponsorship": {"BTL – Event"},
		"GL code: 7250090 – Digital platforms":      {"Digital – Platform development"},
		"Region: LUBES – SOUTH EAST ASIA":           {"Lubricants – Asia Pacific"},
		"GL code: 7250010 – Advertising TV":         {"ATL – TV"},
		"Region: RETAIL – CHINA":                    {"Retail – China"},
		"GL code: 7250060 – Agency fees":            {"Agency – Creative"},
		"Region: LUBES – UK & IRELAND":              {"Lubricants – Europe"},
		"Region: LUBES – BENELUX":                   {"Lubricants – Europe"},
		"Region: FLEET – NORTH CHINA":               {"Fleet – China"},
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
