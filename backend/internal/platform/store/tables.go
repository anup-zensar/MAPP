package store

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// listKey is the memo key of a table decoded as one record type in a Read (List).
type listKey struct {
	table  string
	record reflect.Type
}

// Get returns the record id of table, and false when there is none.
func Get[T any](ctx context.Context, table, id string) (T, bool, error) {
	var record T
	t, err := transaction(ctx, table)
	if err != nil {
		return record, false, err
	}
	encoded, found := t.records(table)[id]
	if !found {
		return record, false, nil
	}
	if err := json.Unmarshal(encoded, &record); err != nil {
		return record, false, fmt.Errorf("store: read %s %s: %w", table, id, err)
	}
	return record, true, nil
}

// List returns every record of table, ordered by id: shorter ids first, then alphabetically, so that
// P-999 comes before P-1001. The list is never nil.
//
// A Read decodes the table once and gives every List of it the same records in a list of its own: the
// caller may reorder or append to the list, but the records' slices, maps and pointed-to values are
// shared with the other Lists of that Read and must not be changed in place. A Write decodes afresh on
// every call, so records read to be changed and saved are always the caller's own.
func List[T any](ctx context.Context, table string) ([]T, error) {
	t, err := transaction(ctx, table)
	if err != nil {
		return nil, err
	}
	if t.writable {
		return decodeTable[T](t, table)
	}
	list, err := memoIn(ctx, t, listKey{table: table, record: reflect.TypeFor[T]()},
		func(context.Context) ([]T, error) { return decodeTable[T](t, table) })
	if err != nil {
		return nil, err
	}
	return slices.Clone(list), nil
}

// decodeTable decodes every record of table as the transaction sees it, ordered by id (List).
func decodeTable[T any](t *tx, table string) ([]T, error) {
	records := t.records(table)
	list := make([]T, 0, len(records))
	for _, id := range slices.SortedFunc(maps.Keys(records), compareIDs) {
		var record T
		if err := json.Unmarshal(records[id], &record); err != nil {
			return nil, fmt.Errorf("store: read %s %s: %w", table, id, err)
		}
		list = append(list, record)
	}
	return list, nil
}

// Put saves record under id in table, replacing the record that had that id.
func Put(ctx context.Context, table, id string, record any) error {
	t, err := writableTransaction(ctx, table)
	if err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("store: put in %s: the id is empty", table)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("store: put %s %s: %w", table, id, err)
	}
	t.forgetMemos()
	t.ownCopy(table)[id] = encoded
	return nil
}

// Delete removes the record id from table. Deleting a record that does not exist changes nothing.
func Delete(ctx context.Context, table, id string) error {
	t, err := writableTransaction(ctx, table)
	if err != nil {
		return err
	}
	if _, found := t.records(table)[id]; found {
		t.forgetMemos()
		delete(t.ownCopy(table), id)
	}
	return nil
}

// Append adds record at the end of the append-only table.
func Append(ctx context.Context, table string, record any) error {
	t, err := writableTransaction(ctx, table)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(record) // one line: JSON escapes the line breaks inside strings
	if err != nil {
		return fmt.Errorf("store: append to %s: %w", table, err)
	}
	t.forgetMemos()
	t.appended[table] = append(t.appended[table], encoded)
	return nil
}

// Scan returns every record of the append-only table in the order they were appended, including the
// records appended earlier in the same Write. The list is never nil.
func Scan[T any](ctx context.Context, table string) ([]T, error) {
	t, err := transaction(ctx, table)
	if err != nil {
		return nil, err
	}
	encoded := slices.Concat(t.db.logs[table], t.appended[table])
	list := make([]T, 0, len(encoded))
	for i, line := range encoded {
		var record T
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("store: read %s record %d: %w", table, i+1, err)
		}
		list = append(list, record)
	}
	return list, nil
}

// compareIDs orders ids as people expect: shorter ids first, then alphabetically.
func compareIDs(a, b string) int {
	return cmp.Or(cmp.Compare(len(a), len(b)), strings.Compare(a, b))
}
