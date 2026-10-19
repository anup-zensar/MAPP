package allwork

import (
	"reflect"
	"testing"
)

// childrenOf returns the keys of the rows whose parent is parentKey ("" for plans), in order.
func childrenOf(r Result, parentKey string) []string {
	keys := []string{}
	for _, row := range r.Rows {
		parent := ""
		if row.ParentKey != nil {
			parent = *row.ParentKey
		}
		if parent == parentKey {
			keys = append(keys, row.Key)
		}
	}
	return keys
}

// A sort orders the children of every parent — and the plans — ascending or descending, and without a sort
// they are in the plan tree's order again; the hierarchy stays intact (FR-015, US3 scenario 2).
func TestSort_ChildrenOfEveryParentAscDescDefault_FR015(t *testing.T) {
	b := seedYear()
	ascending := view(b, Query{Sort: &Sort{Column: "plannedCost", Direction: SortAscending}})
	descending := view(b, Query{Sort: &Sort{Column: "plannedCost", Direction: SortDescending}})
	unsorted := view(b, Query{})

	tests := []struct {
		parent                  string
		asc, desc, defaultOrder []string
	}{
		{"", []string{"P-1002", "P-1001"}, []string{"P-1001", "P-1002"}, []string{"P-1001", "P-1002"}},
		{"P-1001", []string{"C-1002", "C-1001"}, []string{"C-1001", "C-1002"}, []string{"C-1001", "C-1002"}},
		{"I-1001", []string{"A-1002", "A-1001"}, []string{"A-1001", "A-1002"}, []string{"A-1001", "A-1002"}},
		{"I-1004", []string{"A-1007", "A-1006"}, []string{"A-1006", "A-1007"}, []string{"A-1006", "A-1007"}},
		{"A-1002", []string{"L-1004", "L-1003", "A-1002:unassigned"}, []string{"L-1003", "L-1004", "A-1002:unassigned"},
			[]string{"L-1003", "L-1004", "A-1002:unassigned"}},
	}
	for _, tt := range tests {
		for name, r := range map[string]Result{"asc": ascending, "desc": descending, "default": unsorted} {
			want := map[string][]string{"asc": tt.asc, "desc": tt.desc, "default": tt.defaultOrder}[name]
			if got := childrenOf(r, tt.parent); !reflect.DeepEqual(got, want) {
				t.Errorf("%s: children of %q = %q, want %q", name, tt.parent, got, want)
			}
		}
	}
	if rowOf(ascending.Rows, "A-1002").ParentKey == nil || *rowOf(ascending.Rows, "A-1002").ParentKey != "I-1001" ||
		len(ascending.Rows) != len(unsorted.Rows) {
		t.Errorf("sorting keeps every row under its parent")
	}
}

// Records without a value come last in both directions, statuses sort in their lifecycle order, and an
// activity's unassigned row stays after its line items (FR-015).
func TestSort_EmptyLast_StatusLifecycle_UnassignedStays_FR015(t *testing.T) {
	b := seedYear()
	tests := []struct {
		sort   Sort
		parent string
		want   []string
	}{
		{Sort{"plannedCost", SortAscending}, "I-1005", []string{"A-1008", "A-1009"}},
		{Sort{"plannedCost", SortDescending}, "I-1005", []string{"A-1008", "A-1009"}},
		{Sort{"owner", SortDescending}, "I-1005", []string{"A-1008", "A-1009"}},
		{Sort{"status", SortAscending}, "I-1002", []string{"A-1004", "A-1003"}},            // Submitted before Complete
		{Sort{"status", SortDescending}, "I-1002", []string{"A-1003", "A-1004"}},           // Complete before Submitted
		{Sort{"status", SortAscending}, "C-1003", []string{"I-1007", "I-1005", "I-1006"}},  // No activities first
		{Sort{"timing", SortDescending}, "C-1003", []string{"I-1005", "I-1007", "I-1006"}}, // Finished, Running, none
		{Sort{"timing", SortAscending}, "C-1003", []string{"I-1007", "I-1005", "I-1006"}},
		{Sort{"completeness", SortDescending}, "I-1005", []string{"A-1009", "A-1008"}},
		{Sort{"startDate", SortDescending}, "I-1004", []string{"A-1007", "A-1006"}},
		{Sort{"endDate", SortAscending}, "I-1003", []string{"A-1005"}}, // ongoing: no end date
		{Sort{"attr.SA-1003", SortAscending}, "P-1002", []string{"C-1003"}},
		{Sort{"unassigned", SortAscending}, "A-1008", []string{"L-1007", "L-1008", "A-1008:unassigned"}},
		{Sort{"plannedCost", SortDescending}, "A-1008", []string{"L-1008", "L-1007", "A-1008:unassigned"}},
		{Sort{"name", SortAscending}, "I-1001", []string{"A-1001", "A-1002"}}, // not a column: left out
	}
	for _, tt := range tests {
		s := tt.sort
		r := view(b, Query{Sort: &s})

		if got := childrenOf(r, tt.parent); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%+v: children of %s = %q, want %q", tt.sort, tt.parent, got, tt.want)
		}
	}
	// Plans sort by their own status: Draft before Published.
	r := view(b, Query{Sort: &Sort{"status", SortAscending}})
	if got := childrenOf(r, ""); !reflect.DeepEqual(got, []string{"P-1002", "P-1001"}) {
		t.Errorf("plans by status = %q", got)
	}
}
