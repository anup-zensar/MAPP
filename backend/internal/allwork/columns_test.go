package allwork

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Without a choice of columns the grid shows the default columns, in order: ID, Status, Owner, Start date,
// End date, Plan budget, Planned cost, Line items, Unassigned planned cost, then 015's Committed and
// Invoiced; Name is always shown and is not a column (FR-019, 015 FR-023).
func TestColumns_DefaultColumns_FR019(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{})

	var keys []string
	for _, c := range r.Columns {
		keys = append(keys, c.Key)
	}
	want := []string{"id", "status", "owner", "startDate", "endDate", "planBudget", "plannedCost", "lineItems", "unassigned",
		"committed", "invoiced"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("columns = %q\nwant      %q", keys, want)
	}
	for _, row := range r.Rows {
		if len(row.Cells) != len(want) {
			t.Fatalf("row %s has %d cells, want %d", row.Key, len(row.Cells), len(want))
		}
	}
	if len(r.Totals.Cells) != len(want) {
		t.Errorf("totals have %d cells, want %d", len(r.Totals.Cells), len(want))
	}
	// The other views have no grid columns.
	if calendar := view(seedYear(), Query{View: ViewCalendar}); len(calendar.Columns) != 0 || len(calendar.Rows[0].Cells) != 0 {
		t.Errorf("calendar columns = %+v", calendar.Columns)
	}
}

// The chooser offers a Standard group and one group per level, each with its level's fields and then its
// Shell attributes in Settings' order; a new attribute is offered at once, an inactive one only while a
// person's columns still name it, and no field that Finance or GSAP fill later is offered (FR-018).
func TestColumns_StandardGroupAndLevelGroups_FR018(t *testing.T) {
	t.Parallel()
	columns := catalogue(fixtureAttributes)

	groups := columnGroupsOf(offered(columns))
	var names []string
	for _, g := range groups {
		names = append(names, g.Label)
	}
	if !reflect.DeepEqual(names, []string{"Standard", "Plan", "Campaign", "Initiative", "Activity", "Line item", "Financials"}) {
		t.Fatalf("groups = %q", names)
	}
	keysOf := func(g ColumnGroup) []string {
		var keys []string
		for _, c := range g.Columns {
			keys = append(keys, c.Key)
		}
		return keys
	}
	if got := keysOf(groups[2]); !reflect.DeepEqual(got, []string{"campaign.description", "attr.SA-1003"}) {
		t.Errorf("Campaign group = %q (the inactive Region is not offered)", got)
	}
	if got := keysOf(groups[4]); !reflect.DeepEqual(got, []string{"activity.externalName", "activity.type", "activity.ongoing",
		"activity.approvalWorkflow", "activity.description", "activity.setLaterStages", "attr.SA-1020", "attr.SA-1021"}) {
		t.Errorf("Activity group = %q", got)
	}
	// 016 FR-037 adds Year, Plan type, Plan currency, Plan budget (plan currency) and Planned cost (plan currency).
	if len(groups[0].Columns) != 25 || len(groups[1].Columns) != 13 || len(groups[5].Columns) != 9 {
		t.Errorf("standard %d, plan %d, line item %d columns", len(groups[0].Columns), len(groups[1].Columns), len(groups[5].Columns))
	}
	for _, c := range columns {
		if c.Group == GroupFinancials { // the GSAP figures of feature 015 (FR-023)
			continue
		}
		for _, later := range []string{"actual", "invoice", "purchase", "swbsCode"} {
			if strings.Contains(strings.ToLower(c.Key), strings.ToLower(later)) {
				t.Errorf("column %s is filled by Finance or GSAP later", c.Key)
			}
		}
	}
	named := columnGroupsOf(offered(columns, []string{"attr.SA-1004"}))
	if got := keysOf(named[2]); !reflect.DeepEqual(got, []string{"campaign.description", "attr.SA-1003", "attr.SA-1004"}) ||
		!named[2].Columns[2].Inactive {
		t.Errorf("an inactive attribute still named is offered, marked inactive: %+v", named[2].Columns)
	}
	added := catalogue(append(slices.Clone(fixtureAttributes), Attribute{ID: "SA-1099", Label: "Brand", Level: LevelInitiative, Active: true}))
	if got := keysOf(columnGroupsOf(added)[3]); !reflect.DeepEqual(got, []string{"initiative.description", "attr.SA-1099"}) {
		t.Errorf("a new attribute is offered at once: %q", got)
	}
}

// A level-specific column is blank on other levels and "—" for an empty value on its own level; a choice
// shows its label, a % split its shares, a number stays a number, a yes/no is Yes or No (FR-020, US3
// scenario 1).
func TestCells_BlankOnOtherLevels_DashOnOwnLevel_SplitShares_FR020(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{Columns: []string{"attr.SA-1003", "attr.SA-1020", "attr.SA-1021", "plan.model",
		"activity.ongoing", "lineItem.localPlannedCost", "lineItem.swbsStatus"}})

	tests := []struct{ row, column, want string }{
		{"C-1001", "attr.SA-1003", "Lubricants – Europe"},
		{"C-1003", "attr.SA-1003", "—"},
		{"P-1001", "attr.SA-1003", ""},
		{"I-1001", "attr.SA-1003", ""},
		{"A-1001", "attr.SA-1003", ""},
		{"A-1001", "attr.SA-1020", "Germany 60 %, France 40 %"},
		{"A-1002", "attr.SA-1020", "—"},
		{"A-1001", "attr.SA-1021", "1200"},
		{"P-1001", "plan.model", "Lubricants 1.0"},
		{"P-1002", "plan.model", "—"},
		{"A-1005", "activity.ongoing", "—"},
		{"L-1002", "lineItem.localPlannedCost", "46000.00 EUR"},
		{"L-1005", "lineItem.swbsStatus", "Ready to send to GSAP/positive"},
		{"L-1001", "lineItem.swbsStatus", "Not requested/neutral"},
		{"A-1001:unassigned", "attr.SA-1020", ""},
	}
	for _, tt := range tests {
		if got := cell(r, tt.row, tt.column); got != tt.want {
			t.Errorf("%s %s = %q, want %q", tt.row, tt.column, got, tt.want)
		}
	}
	// A local amount stays in its own currency whatever the display currency (FR-017).
	eur := view(seedYear(), Query{Columns: []string{"lineItem.localPlannedCost"}}, Currency{Code: "GBP", Rate: mustRate("0.79")})
	if got := cell(eur, "L-1002", "lineItem.localPlannedCost"); got != "46000.00 EUR" {
		t.Errorf("local amount in GBP display = %s", got)
	}
	if got := cellText(fieldCell(Value{Yes: ptr(true)}, usd)); got != "Yes" {
		t.Errorf("yes/no = %s", got)
	}
	if got := cellText(fieldCell(Value{USD: ptr(amountOf("2500000.00"))}, Currency{Code: "EUR", Rate: mustRate("0.92")})); got != "2300000.00 EUR" {
		t.Errorf("an amount in USD is shown in the display currency: %s", got)
	}
}

// Inactive options, types and people still appear, labelled "(inactive)" (edge case).
func TestCells_InactiveValuesLabelled_EdgeCase(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{Columns: []string{"attr.SA-1004", "attr.SA-1020", "activity.type", "owner"}})

	tests := []struct{ row, column, want string }{
		{"C-1002", "attr.SA-1004", "LUBES – BENELUX (inactive)"},
		{"C-1001", "attr.SA-1004", "LUBES – DACH"},
		{"A-1008", "attr.SA-1020", "China 70 %, Macau (inactive) 30 %"},
		{"A-1010", "activity.type", "MOBILITY – Long-term contract (inactive)"},
		{"I-1007", "owner", "Sara Novak (inactive)"},
	}
	for _, tt := range tests {
		if got := cell(r, tt.row, tt.column); got != tt.want {
			t.Errorf("%s %s = %q, want %q", tt.row, tt.column, got, tt.want)
		}
	}
	if !r.Columns[0].Inactive || r.Columns[0].Label != "Region" {
		t.Errorf("the inactive Region column = %+v", r.Columns[0])
	}
}

// Every row shows its own values in the Standard columns (FR-011, FR-017): the status chip of its level —
// plan status, Approval progress with its detail, activity display status, a line item's completeness —
// owner, dates (plans: the whole year; "Ongoing"), money in the display currency, and a blank cell on
// levels a column does not cover.
func TestCells_StandardColumnsPerLevel_FR011_FR017(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{Columns: allColumns()})

	tests := []struct{ row, column, want string }{
		{"P-1001", "id", "P-1001"},
		{"P-1001", "level", "Plan"},
		{"P-1001", "status", "Published/positive"},
		{"P-1002", "status", "Draft/neutral"},
		{"P-1001", "startDate", "2026-01-01"},
		{"P-1001", "endDate", "2026-12-31"},
		{"P-1001", "owner", "Priya Raman"},
		{"P-1001", "timing", ""},
		{"P-1002", "completeness", "1 field needed/problem"},
		{"P-1001", "completeness", "All filled in/positive"},
		{"C-1001", "status", "Partly approved/review"},
		{"C-1001", "timing", "Running/active"},
		{"I-1003", "status", "All approved/positive"},
		{"I-1006", "timing", "—"},
		{"I-1006", "startDate", "—"},
		{"I-1007", "owner", "Sara Novak (inactive)"},
		{"A-1001", "status", "Created/neutral"},
		{"A-1002", "status", "Complete/done"},
		{"A-1005", "status", "In progress/active"},
		{"A-1006", "status", "Declined/problem"},
		{"A-1005", "endDate", "Ongoing"},
		{"A-1001", "startDate", "2026-03-01"},
		{"A-1009", "owner", "—"},
		{"A-1009", "plannedCost", "—"},
		{"A-1009", "completeness", "5 fields needed/problem"},
		{"A-1001", "planBudget", ""},
		{"A-1001", "timing", ""},
		{"L-1007", "status", "1 field needed/problem"},
		{"L-1001", "status", "All filled in/positive"},
		{"L-1002", "plannedCost", "50000.00 USD"},
		{"L-1002", "startDate", ""},
		{"L-1002", "unassigned", ""},
		{"L-1002", "level", "Line item"},
		{"A-1001:unassigned", "plannedCost", "20000.00 USD"},
		{"A-1001:unassigned", "id", ""},
		{"A-1001", "createdAt", "2026-10-16T14:05:00Z"},
		{"A-1001", "updatedAt", "2026-10-17T14:05:00Z"},
	}
	for _, tt := range tests {
		if got := cell(r, tt.row, tt.column); got != tt.want {
			t.Errorf("%s %s = %q, want %q", tt.row, tt.column, got, tt.want)
		}
	}
	campaign := rowOf(r.Rows, "C-1001")
	if campaign.Status == nil || campaign.Status.Detail != "2 of 4 activities approved" || campaign.Timing == nil ||
		campaign.Timing.Label != "Running" || campaign.Owner == nil || campaign.Owner.Name != "Priya Raman" {
		t.Errorf("campaign row = %+v", campaign)
	}
	activity := rowOf(r.Rows, "A-1005")
	if !activity.Ongoing || activity.StartDate.String() != "2026-03-01" || !activity.EndDate.IsZero() ||
		moneyText(activity.Balance) != "100000.00 USD" || moneyText(activity.PlannedCost) != "100000.00 USD" {
		t.Errorf("ongoing activity row = %+v", activity)
	}
}
