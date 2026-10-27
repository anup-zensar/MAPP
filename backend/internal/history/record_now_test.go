package history

import (
	"fmt"
	"reflect"
	"testing"
)

// Each entry says where its record is now (011 FR-008, data model §4): stored, with its current name and
// place even after a rename; in the recycle bin, also inside another record's entry, with that entry; gone
// for good; or no record at all.
func TestQueryLog_RecordNowStates_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.logAs(t, priya, parentOne(ActionCreated, ""))
	f.put(t, parentsTable, newRecord("P-1", "Parent renamed", ""))
	f.logAs(t, priya, childOne(ActionCreated))
	f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-1", RecordName: "Grandchild one", Action: ActionCreated})
	f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-3", RecordName: "Grandchild three", Action: ActionCreated})
	f.removeForGood(t, grandchildrenTable, "G-3")
	f.appendRaw(t, daniel, Entry{Action: ActionEmptiedRecycleBin, Verb: verbEmptiedRecycleBin,
		RecordType: emptiedBinType, RecordTypeLabel: "recycle bin", RecordName: "Recycle bin", Details: "1 item"})
	f.logAs(t, priya, LogInput{Kind: "attachment", RecordID: "X-1", RecordName: "Unknown kind", Action: ActionCreated, ParentID: "P-1"}, f.withKind(t, "attachment"))
	bin := f.moveToBin(t, "child", "C-1") // holds G-1 too

	page := f.query(t, LogQuery{Limit: 50})

	parentNow := []RecordRef{{Type: "parent", ID: "P-1", Name: "Parent renamed"}}
	want := map[string]RecordNow{ // by the entry's record and action
		"parent P-1 created":            {State: RecordExists, Name: "Parent renamed", Path: []RecordRef{}},
		"child C-1 created":             {State: RecordInRecycleBin, Name: "Child one", Path: []RecordRef{}, BinEntryID: bin.ID},
		"child C-1 deleted":             {State: RecordInRecycleBin, Name: "Child one", Path: []RecordRef{}, BinEntryID: bin.ID},
		"grandchild G-1 created":        {State: RecordInRecycleBin, Name: "Grandchild one", Path: []RecordRef{}, BinEntryID: bin.ID},
		"grandchild G-3 created":        {State: RecordDeletedPermanently, Path: []RecordRef{}},
		"recycleBin  emptiedRecycleBin": {State: RecordNone, Path: []RecordRef{}},
		"attachment X-1 created":        {State: RecordNone, Path: []RecordRef{}},
	}
	got := map[string]RecordNow{}
	for _, item := range page.Items {
		got[fmt.Sprintf("%s %s %s", item.RecordType, item.RecordID, item.Action)] = item.RecordNow
		if item.Items == nil {
			t.Errorf("%s: items = nil, want an empty list", item.ID)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recordNow =\n%s\nwant\n%s", asJSON(got), asJSON(want))
	}

	f.put(t, childrenTable, newRecord("C-2", "Child two", "P-1"))
	f.logAs(t, priya, LogInput{Kind: "child", RecordID: "C-2", RecordName: "Child two", Action: ActionChanged})
	if latest := f.query(t, LogQuery{Limit: 1}).Items[0].RecordNow; !reflect.DeepEqual(latest.Path, parentNow) {
		t.Errorf("path of C-2 now = %+v, want its current place %+v", latest.Path, parentNow)
	}
}
