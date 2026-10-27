package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// log writes an entry as Priya and stops the test when that fails.
func (f *fixture) log(t *testing.T, in LogInput) Entry {
	t.Helper()
	entry, err := f.service.Log(t.Context(), priya, in)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	return entry
}

// moveToBin deletes the record id of kind as Priya and stops the test when that fails.
func (f *fixture) moveToBin(t *testing.T, kind, id string) BinEntry {
	t.Helper()
	entry, err := f.service.MoveToBin(t.Context(), priya, kind, id)
	if err != nil {
		t.Fatalf("MoveToBin(%s %s): %v", kind, id, err)
	}
	return entry
}

// undo undoes the delete of entryID as a and stops the test when that fails.
func (f *fixture) undo(t *testing.T, a actor.Actor, entryID string) {
	t.Helper()
	if _, err := f.service.Undo(t.Context(), a, entryID); err != nil {
		t.Fatalf("Undo(%s): %v", entryID, err)
	}
}

// restore restores entryID as a and stops the test when that fails.
func (f *fixture) restore(t *testing.T, a actor.Actor, entryID string) {
	t.Helper()
	if _, err := f.service.Restore(t.Context(), a, entryID); err != nil {
		t.Fatalf("Restore(%s): %v", entryID, err)
	}
}

// listBin returns what is in the recycle bin and stops the test when that fails.
func (f *fixture) listBin(t *testing.T) []BinEntry {
	t.Helper()
	bin, err := f.service.ListBin(t.Context(), priya)
	if err != nil {
		t.Fatalf("ListBin: %v", err)
	}
	return bin
}

// parentOne is the log input of a change of the parent P-1.
func parentOne(action Action, verb string) LogInput {
	return LogInput{Kind: "parent", RecordID: "P-1", RecordName: "Parent one", Action: action, Verb: verb}
}

// snapshotOf is the snapshot history keeps of r, a record of kind.
func snapshotOf(kind string, r testRecord) Snapshot {
	data, _ := json.Marshal(r) // cannot fail: a testRecord holds only strings, numbers and times
	return Snapshot{Kind: kind, ID: r.ID, Name: r.Name, ParentID: r.ParentID, Data: data}
}

// raw is a stored JSON value, written as in the data file.
func raw(value string) json.RawMessage {
	return json.RawMessage(value)
}

func entryIDs(entries []Entry) []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		list = append(list, e.ID)
	}
	return list
}

// assertEntries checks that got are exactly want, and shows both as JSON when they differ.
func assertEntries(t *testing.T, got, want []Entry) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("entries =\n%s\nwant\n%s", asJSON(got), asJSON(want))
	}
}

// assertError checks that err is exactly want: the same status, code, message and details.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %v, want %s", err, asJSON(want))
	}
}

// asJSON writes v as indented JSON, for failure messages.
func asJSON(v any) string {
	text, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(text)
}

// listBinItems returns the rows of the recycle bin page and stops the test when that fails.
func (f *fixture) listBinItems(t *testing.T) []BinListItem {
	t.Helper()
	items, err := f.service.ListBinItems(t.Context(), priya)
	if err != nil {
		t.Fatalf("ListBinItems: %v", err)
	}
	return items
}

// preview returns the restore preview of entryID as Priya and stops the test when that fails.
func (f *fixture) preview(t *testing.T, entryID string) RestorePreview {
	t.Helper()
	preview, err := f.service.PreviewRestore(t.Context(), priya, entryID)
	if err != nil {
		t.Fatalf("PreviewRestore(%s): %v", entryID, err)
	}
	return preview
}

// childOne is the log input of a change of the child C-1.
func childOne(action Action) LogInput {
	return LogInput{Kind: "child", RecordID: "C-1", RecordName: "Child one", Action: action}
}

// logAs writes an entry as a at the clock's time and stops the test when that fails. With a kind
// registered only for this entry (withKind), the entry is about a kind the service does not know.
func (f *fixture) logAs(t *testing.T, a actor.Actor, in LogInput, service ...*Service) Entry {
	t.Helper()
	s := f.service
	if len(service) > 0 {
		s = service[0]
	}
	entry, err := s.Log(t.Context(), a, in)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	return entry
}

// query runs QueryLog as Priya and stops the test when it fails.
func (f *fixture) query(t *testing.T, q LogQuery) LogPage {
	t.Helper()
	page, err := f.service.QueryLog(t.Context(), priya, q)
	if err != nil {
		t.Fatalf("QueryLog(%+v): %v", q, err)
	}
	return page
}

// summary runs LogSummary as Priya and stops the test when it fails.
func (f *fixture) summary(t *testing.T, todayStart, weekStart time.Time) LogSummary {
	t.Helper()
	summary, err := f.service.LogSummary(t.Context(), priya, todayStart, weekStart)
	if err != nil {
		t.Fatalf("LogSummary: %v", err)
	}
	return summary
}

// pageIDs returns the ids of the entries of page, in order.
func pageIDs(page LogPage) []string {
	list := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		list = append(list, item.ID)
	}
	return list
}

// assertPage checks that the page holds exactly the entries want, in order, and that total entries match.
func assertPage(t *testing.T, page LogPage, want []Entry, total int) {
	t.Helper()
	got := make([]Entry, 0, len(page.Items))
	for _, item := range page.Items {
		got = append(got, item.Entry)
	}
	assertEntries(t, got, want)
	if page.Total != total {
		t.Errorf("total = %d, want %d", page.Total, total)
	}
}
