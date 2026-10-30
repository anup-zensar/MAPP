package allwork

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// matched returns the keys of the rows that are matches of level, in order.
func matched(r Result, level Level) []string {
	keys := []string{}
	for _, row := range r.Rows {
		if row.Level == level && row.Role == RoleMatch {
			keys = append(keys, row.Key)
		}
	}
	return keys
}

func rolesOf(r Result) []string {
	rows := []string{}
	for _, row := range r.Rows {
		rows = append(rows, row.Key+" "+row.Role)
	}
	return rows
}

// Each filter on its own level, on the demo year with FY2026 Mobility China made a Strategic / AOP plan.
func TestFilter_EachFilterOnItsLevel_FR004(t *testing.T) {
	t.Parallel()
	b := seedYear()
	b.node("P-1002").Strategic = true
	lubricants := []string{"A-1001", "A-1002", "A-1003", "A-1004", "A-1005", "A-1006", "A-1007"}
	mobility := []string{"A-1008", "A-1009", "A-1010"}
	tests := []struct {
		name string
		q    Query
		want []string
	}{
		{"plan", Query{Plans: []string{"P-1002"}}, mobility},
		{"plan status", Query{PlanStatuses: []string{"published"}}, lubricants},
		{"campaign / initiative status", Query{ContainerStatuses: []string{"allApproved"}}, []string{"A-1005"}},
		{"activity status", Query{ActivityStatuses: []string{"declined", "complete"}}, []string{"A-1002", "A-1003", "A-1006"}},
		{"owner", Query{Owners: []string{daniel.ID}}, []string{"A-1007"}},
		{"team", Query{Teams: []string{"T-1002"}}, mobility},
		{"business unit", Query{BusinessUnits: []string{"SO-1002"}}, lubricants},
		{"activity type", Query{ActivityTypes: []string{"AT-1005"}}, []string{"A-1006", "A-1007", "A-1008"}},
		{"strategic only", Query{Strategic: StrategicOnly}, mobility},
		{"strategic excluded", Query{Strategic: StrategicExclude}, lubricants},
		{"strategic included", Query{Strategic: StrategicInclude}, slices.Concat(lubricants, mobility)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := view(b, tt.q)

			if got := matched(r, LevelActivity); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("matching activities = %q, want %q", got, tt.want)
			}
		})
	}
}

// "Activity status: Declined" keeps only branches leading to a declined activity; cards count only matches.
func TestFilter_ActivityStatusShowsOnlyBranchesToMatches_FR005(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{ActivityStatuses: []string{"declined"}})

	want := []string{"P-1001 match", "C-1002 match", "I-1004 match", "A-1006 match", "A-1006:unassigned match"}
	if got := rolesOf(r); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %q\nwant   %q", got, want)
	}
	if s := r.Summary; s.Activities != 1 || s.Campaigns != 1 || s.Initiatives != 1 || moneyText(s.PlannedCost) != "60000.00 USD" ||
		r.Count != (Count{Shown: 1, Total: 10, Noun: "activities"}) {
		t.Errorf("summary = %+v, count = %+v", s, r.Count)
	}
	if !rowOf(r.Rows, "C-1002").Filtered || cell(r, "C-1002", "plannedCost") != "60000.00 USD [all: 185000.00 USD]" {
		t.Errorf("Rimula fleet planned cost = %s, filtered %v", cell(r, "C-1002", "plannedCost"), rowOf(r.Rows, "C-1002").Filtered)
	}
}

// Filtering by owner shows that person's records; parents are context rows the cards do not count.
func TestFilter_OwnerParentsAreContext_FR005(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{Owners: []string{daniel.ID}})

	want := []string{"P-1001 context", "C-1002 match", "I-1003 match", "I-1004 match", "A-1007 match",
		"A-1007:unassigned match"}
	if got := rolesOf(r); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %q\nwant   %q", got, want)
	}
	if s := r.Summary; s.Campaigns != 1 || s.Initiatives != 2 || s.Activities != 1 || s.PlanBudget != nil ||
		moneyText(s.PlannedCost) != "25000.00 USD" {
		t.Errorf("summary = %+v (the context plan's budget is not counted)", s)
	}
	assertTotalsEqualCards(t, r)
}

// A filter on higher levels holds through the nearest tested ancestor: an activity follows its initiative.
func TestFilter_HigherLevelFilterInheritedFromNearestAncestor_FR005(t *testing.T) {
	t.Parallel()
	b := seedYear()

	approved := view(b, Query{ContainerStatuses: []string{"allApproved"}})
	if got := strings.Join(rolesOf(approved), ", "); got != "P-1001 match, C-1002 context, I-1003 match, A-1005 match, A-1005:unassigned match" {
		t.Errorf("All approved: rows = %s", got)
	}
	draft := view(b, Query{PlanStatuses: []string{"draft"}})
	if got := matched(draft, LevelActivity); !reflect.DeepEqual(got, []string{"A-1008", "A-1009", "A-1010"}) {
		t.Errorf("Draft plans: matching activities = %q", got)
	}
}

// Status values combine with OR inside Approval progress or Timing, and the two groups with AND.
func TestFilter_ContainerStatusOrWithinGroupAndAcross_FR005(t *testing.T) {
	t.Parallel()
	tests := []struct {
		values                 []string
		campaigns, initiatives []string
	}{
		{[]string{"allApproved", "noActivities"}, []string{}, []string{"I-1003", "I-1007"}},
		{[]string{"finished"}, []string{}, []string{"I-1001", "I-1003", "I-1005"}},
		{[]string{"inProgress", "finished"}, []string{}, []string{"I-1001", "I-1005"}},
		{[]string{"running"}, []string{"C-1001", "C-1002", "C-1003"}, []string{"I-1002", "I-1004", "I-1007"}},
	}
	for _, tt := range tests {
		r := view(seedYear(), Query{ContainerStatuses: tt.values})

		if got := matched(r, LevelCampaign); !reflect.DeepEqual(got, tt.campaigns) {
			t.Errorf("%q: matching campaigns = %q, want %q", tt.values, got, tt.campaigns)
		}
		if got := matched(r, LevelInitiative); !reflect.DeepEqual(got, tt.initiatives) {
			t.Errorf("%q: matching initiatives = %q, want %q", tt.values, got, tt.initiatives)
		}
	}
}

// The count reads "Showing N of M activities", or counts campaigns or initiatives for the higher Show levels.
func TestCount_ShowingNOfMWithNoun_FR008(t *testing.T) {
	t.Parallel()
	tests := []struct {
		q    Query
		want Count
	}{
		{Query{}, Count{Shown: 10, Total: 10, Noun: "activities"}},
		{Query{ActivityStatuses: []string{"declined"}}, Count{Shown: 1, Total: 10, Noun: "activities"}},
		{Query{Search: "nothing like this"}, Count{Shown: 0, Total: 10, Noun: "activities"}},
		{Query{ShowLevels: ShowCompleteBranches, Owners: []string{mei.ID}}, Count{Shown: 2, Total: 10, Noun: "activities"}},
		{Query{ShowLevels: ShowInitiativesWithoutActivities, Owners: []string{mei.ID}}, Count{Shown: 0, Total: 7, Noun: "initiatives"}},
		{Query{ShowLevels: ShowCampaignsWithoutInitiatives}, Count{Shown: 0, Total: 3, Noun: "campaigns"}},
	}
	for _, tt := range tests {
		if got := view(seedYear(), tt.q).Count; got != tt.want {
			t.Errorf("%+v: count = %+v, want %+v", tt.q, got, tt.want)
		}
	}
}

// For every filter, search and Show levels combination, in every view, the totals row equals the cards.
func TestRollup_TotalsEqualSummaryCards_EveryFilterCombination_SC003(t *testing.T) {
	t.Parallel()
	b := seedYear()
	data := b.data()
	for _, statuses := range [][]string{nil, {"declined"}, {"approved", "complete", "inProgress"}} {
		for _, owners := range [][]string{nil, {daniel.ID}, {priya.ID, mei.ID}} {
			for _, containers := range [][]string{nil, {"inProgress"}, {"finished", "running"}} {
				for _, show := range showLevelsValues() {
					for _, search := range []string{"", "fleet"} {
						for _, v := range viewValues() {
							q := Query{View: v, ActivityStatuses: statuses, Owners: owners, ContainerStatuses: containers,
								ShowLevels: show, Search: search, Columns: allColumns()}
							assertTotalsEqualCards(t, viewOf(data, q).result)
						}
					}
				}
			}
		}
	}
}

// The Grid is the plan tree in tree order (not read order), with the unassigned row last under its activity.
func TestTree_PlansByNameChildrenInPlanTreeOrder_FR011(t *testing.T) {
	t.Parallel()
	b := newYear()
	zeta := b.plan("P-1001", "zeta plan")
	alpha := b.plan("P-1002", "Alpha plan")
	second := b.campaign(alpha, "C-1001", "Second") // order 0
	first := b.campaign(alpha, "C-1002", "First")   // order 1
	b.node(second).Order, b.node(first).Order = 1, 0
	i := b.initiative(first, "I-1001", "Only")
	a := b.activity(i, "A-1001", "Work", planned("100.00"))
	b.lineItem(a, "L-1002", "Later", "10.00")
	b.lineItem(a, "L-1001", "Earlier", "20.00")
	b.node("L-1001").Order, b.node("L-1002").Order = 0, 1
	data := b.data()
	slices.Reverse(data.Nodes)

	r := viewOf(data, Query{}).result

	want := []string{"P-1002", "C-1002", "I-1001", "A-1001", "L-1001", "L-1002", "A-1001:unassigned", "C-1001", zeta}
	if got := rowKeys(r.Rows); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %q\nwant   %q", got, want)
	}
	line := rowOf(r.Rows, "L-1001")
	if line.ParentKey == nil || *line.ParentKey != "A-1001" || line.Depth != 4 || line.Level != LevelLineItem ||
		line.ActivityID != "A-1001" || line.PlanID != "P-1002" || line.RecordID != "L-1001" {
		t.Errorf("line item row = %+v", line)
	}
	plan := rowOf(r.Rows, "P-1002")
	if plan.ParentKey != nil || plan.Depth != 0 || !plan.HasChildren || plan.Strategic == nil || *plan.Strategic {
		t.Errorf("plan row = %+v", plan)
	}
	if rowOf(r.Rows, zeta).HasChildren || !rowOf(r.Rows, "A-1001").HasChildren {
		t.Errorf("hasChildren: an empty plan has none, an activity with line items has some")
	}
}

// A non-zero balance adds an "Unassigned planned cost (Sys_Gen_Balance)" row, negative when line items exceed it.
func TestUnassignedRows_OnlyNonZeroBalances_FR012(t *testing.T) {
	t.Parallel()
	r := view(seedYear(), Query{})

	var unassigned []string
	for _, row := range r.Rows {
		if row.Level == LevelUnassigned {
			unassigned = append(unassigned, row.Key)
		}
	}
	want := []string{"A-1001:unassigned", "A-1002:unassigned", "A-1003:unassigned", "A-1005:unassigned",
		"A-1006:unassigned", "A-1007:unassigned", "A-1008:unassigned", "A-1010:unassigned"}
	if !reflect.DeepEqual(unassigned, want) {
		t.Errorf("unassigned rows = %q\nwant              %q", unassigned, want)
	}
	row := rowOf(r.Rows, "A-1008:unassigned")
	if row.Name != "Unassigned planned cost (Sys_Gen_Balance)" || row.RecordID != "A-1008" || row.ActivityID != "A-1008" ||
		row.Status != nil || row.Depth != 4 || moneyText(row.PlannedCost) != "-500.00 USD" {
		t.Errorf("unassigned row = %+v", row)
	}
	if got := cell(r, "A-1008:unassigned", "unassigned"); got != "-500.00 USD" {
		t.Errorf("unassigned cell = %s, want -500.00 USD", got)
	}
	if c := rowOf(r.Rows, "A-1008:unassigned").Cells[columnIndex(r, "unassigned")]; c.Tone != ToneProblem {
		t.Errorf("a negative balance is red: tone = %q", c.Tone)
	}
	keys := rowKeys(r.Rows)
	if i := slices.Index(keys, "A-1001:unassigned"); i < 1 || keys[i-1] != "L-1002" {
		t.Errorf("the unassigned row comes after the activity's line items: %q", keys)
	}
}
