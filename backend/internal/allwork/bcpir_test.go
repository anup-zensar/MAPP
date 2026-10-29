package allwork

import (
	"slices"
	"testing"

	"mapp/internal/platform/errs"
)

// seedWithBCPIR is the seed year with FY2026 Lubricants Europe marked as a BCPIR plan.
func seedWithBCPIR() *yearBuilder {
	b := seedYear()
	yes := true
	b.node("P-1001").Fields[bcpirField] = Value{Yes: &yes}
	return b
}

func TestBCPIR_013_FR003_FilterOnlyAndExclude(t *testing.T) {
	t.Parallel()
	b := seedWithBCPIR()
	only := view(b, Query{BCPIR: StrategicOnly})
	exclude := view(b, Query{BCPIR: StrategicExclude})
	if !hasRow(only.Rows, "P-1001") || hasRow(only.Rows, "P-1002") || !hasRow(only.Rows, "A-1001") {
		t.Errorf("only = %q", rowKeys(only.Rows))
	}
	if hasRow(exclude.Rows, "P-1001") || !hasRow(exclude.Rows, "P-1002") {
		t.Errorf("exclude = %q", rowKeys(exclude.Rows))
	}
}

func TestBCPIR_013_FR002_PlanRowsCarryTheMarker(t *testing.T) {
	t.Parallel()
	rows := view(seedWithBCPIR(), Query{}).Rows
	lub, mob := rowOf(rows, "P-1001"), rowOf(rows, "P-1002")
	if lub.BCPIR == nil || !*lub.BCPIR || mob.BCPIR == nil || *mob.BCPIR || rowOf(rows, "C-1001").BCPIR != nil {
		t.Errorf("markers: P-1001 %v, P-1002 %v", lub.BCPIR, mob.BCPIR)
	}
}

func TestBCPIR_013_FR003_ExportNamesTheFilter(t *testing.T) {
	t.Parallel()
	notes, _ := csvLines(t, exportOf(t, seedWithBCPIR(), Query{BCPIR: StrategicOnly}, FormatCSV, true))
	if !slices.ContainsFunc(notes, func(line []string) bool { return slices.Contains(line, "BCPIR plans: Only") }) {
		t.Errorf("notes = %q", notes)
	}
}

func TestBCPIR_013_FR003_BreakdownGroupsByTheFlag(t *testing.T) {
	t.Parallel()
	nodes := twoYears()
	yes := true
	for i := range nodes {
		if nodes[i].ID == "P-1001" {
			nodes[i].Fields = map[string]Value{bcpirField: {Yes: &yes}}
		}
	}
	b := mustBreakdown(t, nodes, Query{By: byBCPIR})
	assertGroups(t, "BCPIR plan", b.Groups, []string{
		"Yes 2/2 16,000.00 USD 66.67% of 100,000.00 USD (16 %)",
		"No 2/1 8,000.00 USD 33.33% of 50,000.00 USD (16 %)",
	})
}

func TestBCPIR_013_FR003_SavedViewsKeepTheFilter(t *testing.T) {
	t.Parallel()
	for value, valid := range map[string]bool{"": true, StrategicOnly: true, StrategicExclude: true, "yes": false} {
		state := standardState()
		state.Filters.BCPIR = value
		var problems errs.Problems
		checkState(&problems, state)
		if (len(problems) == 0) != valid {
			t.Errorf("bcpir %q: problems %v", value, problems)
		}
	}
}
