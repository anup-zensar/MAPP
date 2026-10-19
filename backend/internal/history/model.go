// Package history is NorthStar's activity log and recycle bin core (FR-022 – FR-027, research R8,
// ADR-0007). It writes one activity-log entry for every business change, and turns every delete into a
// move to the recycle bin: the record and everything under it are kept there, and Undo or Restore brings
// them back. Feature 011 adds the screens.
//
// Every business package registers a Kind for each record type it owns (in internal/app), logs its changes
// with Service.Log inside the Write that makes the change, and deletes with Service.MoveToBin.
package history

import (
	"encoding/json"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
)

// RecordRef points at a record: its kind, its ID and its name at the time. It is the errs type, so that the
// restore_parent_first error can carry one.
type RecordRef = errs.RecordRef

// Resolution says how a recycle-bin entry left the bin ("undone" or "restored"), by whom and when. It is the
// errs type, so that the already_resolved error can carry one.
type Resolution = errs.Resolution

// Action is the kind of change an activity-log entry records. Feature 011 filters and colours entries by it.
type Action string

// The actions of feature 001. Feature 011 adds "deletedPermanently" and "emptiedRecycleBin".
const (
	ActionCreated       Action = "created"
	ActionChanged       Action = "changed"
	ActionStatusChanged Action = "statusChanged"
	ActionMoved         Action = "moved"
	ActionCopied        Action = "copied"
	ActionDeleted       Action = "deleted"
	ActionRestored      Action = "restored"
)

// Entry is one activity-log entry (data model §5), stored in the append-only table "activity_log". Entries
// are only ever added, never changed or removed (FR-023). The API returns them exactly as they are stored
// (ActivityLogEntry), so there is no separate API type.
type Entry struct {
	ID              string      `json:"id"`              // "LOG-1001", …
	Seq             int         `json:"seq"`             // the number of the id: orders entries, also within one millisecond
	At              time.Time   `json:"at"`              // when the change was saved
	Actor           actor.Actor `json:"actor"`           // who made the change
	Action          Action      `json:"action"`          // for filters and the action chip
	Verb            string      `json:"verb"`            // as in "<person> <verb> <record type> '<name>'": "created", "published", …
	RecordType      string      `json:"recordType"`      // the kind's name, such as "sampleFolder"
	RecordTypeLabel string      `json:"recordTypeLabel"` // the kind's mid-sentence name at the time, such as "sample folder"
	RecordID        string      `json:"recordId"`        // the record's readable ID, such as "SF-1001"
	RecordName      string      `json:"recordName"`      // the record's name at the time
	Path            []RecordRef `json:"path"`            // the record's ancestors at the time, top level first; empty for top-level records
	Changes         []Change    `json:"changes"`         // every changed field; empty when the action changes no field
	Comment         string      `json:"comment"`         // a reason the person gave; empty when there is none
	Details         string      `json:"details"`         // plain text such as "with 2 sample notes"; empty when there is nothing to add
}

// Change is one changed field of a record (data model §5). Old and New are the stored JSON values, kept
// exactly; Label, OldText and NewText are what people read, fixed at the time of the change.
type Change struct {
	Field   string          `json:"field"`   // the JSON field name, such as "plannedCost"
	Label   string          `json:"label"`   // the field's name for people, such as "Planned cost"
	Old     json.RawMessage `json:"old"`     // the stored value before the change; null when there was none
	New     json.RawMessage `json:"new"`     // the stored value after the change; null when there is none
	OldText string          `json:"oldText"` // the value before, as people read it; "Not filled in" when empty
	NewText string          `json:"newText"` // the value after, as people read it; "Not filled in" when empty
}

// LogInput is what a service tells Log about one business change. History adds the rest: the id, the time,
// the actor, the kind's label, the record's path and the readable texts of the changes.
type LogInput struct {
	Kind       string   // the kind's name, such as "sampleFolder"
	RecordID   string   // the record's readable ID
	RecordName string   // the record's name now
	ParentID   string   // optional: the parent's id, where the path starts when the record itself is gone
	Action     Action   // what happened
	Verb       string   // optional; required for ActionStatusChanged, such as "published"
	Changes    []Change // usually from Diff; history fills in Label, OldText and NewText
	Comment    string   // optional: a reason the person gave, such as a decline reason
	Details    string   // optional: plain text, such as "with 2 sample notes"
}

// LogFilter chooses the activity-log entries ListLog returns.
type LogFilter struct {
	RecordType string // only entries about this kind of record; "" for every kind
	RecordID   string // only entries about this record; "" for every record
	Limit      int    // at most this many entries: 1 to 200
}

// Snapshot is one deleted record as the recycle bin keeps it (data model §6).
type Snapshot struct {
	Kind     string          `json:"kind"`     // the kind's name; history fills it in
	ID       string          `json:"id"`       // the record's id
	Name     string          `json:"name"`     // the record's name, as the log and the bin show it
	ParentID string          `json:"parentId"` // the parent's id; "" for top-level records
	Data     json.RawMessage `json:"data"`     // the record exactly as it was stored
}

// ContentCount is how many records of one kind were under a deleted record.
type ContentCount struct {
	RecordType string `json:"recordType"` // the kind's name, such as "sampleNote"
	Label      string `json:"label"`      // "sample note" when Count is 1, otherwise "sample notes"
	Count      int    `json:"count"`      // at least 1: kinds with nothing under the record are left out
}

// BinEntry is a recycle-bin entry (data model §6), stored in the table "recycle_bin": a deleted record with
// everything that was under it. An entry stays in the table after it is undone or restored, with its
// Resolution and without its snapshots.
type BinEntry struct {
	ID              string         `json:"id"`              // "BIN-1001", …
	RecordType      string         `json:"recordType"`      // the deleted record's kind
	RecordTypeLabel string         `json:"recordTypeLabel"` // the kind's mid-sentence name at the time
	RecordID        string         `json:"recordId"`        // the deleted record's readable ID
	RecordName      string         `json:"recordName"`      // its name at the time of the delete
	Path            []RecordRef    `json:"path"`            // its ancestors at the time of the delete, top level first
	Parent          *RecordRef     `json:"parent"`          // its parent at the time of the delete; null for top-level records
	DeletedAt       time.Time      `json:"deletedAt"`       // when it was deleted
	DeletedBy       actor.Actor    `json:"deletedBy"`       // who deleted it
	Contents        []ContentCount `json:"contents"`        // what was under it, counted by kind
	Snapshots       []Snapshot     `json:"snapshots"`       // the record, then everything under it, parents before children; empty once resolved
	Resolution      *Resolution    `json:"resolution"`      // how it left the bin; null while it is in the bin
}

// The API types below are what the API returns. They are kept apart from the stored records above, so the
// stored data can change without changing the API (research R10).

// BinEntryView is a recycle-bin entry as the API returns it (RecycleBinEntry): the stored entry without its
// snapshots and resolution.
type BinEntryView struct {
	ID              string         `json:"id"`
	RecordType      string         `json:"recordType"`
	RecordTypeLabel string         `json:"recordTypeLabel"`
	RecordID        string         `json:"recordId"`
	RecordName      string         `json:"recordName"`
	Path            []RecordRef    `json:"path"`
	Parent          *RecordRef     `json:"parent"` // no omitempty: the contract wants null for top-level records
	DeletedAt       time.Time      `json:"deletedAt"`
	DeletedBy       actor.Actor    `json:"deletedBy"`
	Contents        []ContentCount `json:"contents"`
}

// DeleteResult is the answer of every delete of a business record (research R10): the recycle-bin entry,
// so the web app can offer Undo.
type DeleteResult struct {
	RecycleBinEntry BinEntryView `json:"recycleBinEntry"`
}

// RestoreResult is the answer of an Undo or a Restore: the record that came back and what the restore
// changed.
type RestoreResult struct {
	RecordType      string `json:"recordType"`
	RecordTypeLabel string `json:"recordTypeLabel"`
	RecordID        string `json:"recordId"`
	RecordName      string `json:"recordName"` // the name after the restore, which may have a suffix such as " (restored)"
	Details         string `json:"details"`    // the restore's notes, such as "Renamed to … because the name was taken."; "" when none
}

// LogPage is the answer of GET /api/activity-log (ActivityLogPage): the newest matching entries, and how
// many entries match in all.
type LogPage struct {
	Items []Entry `json:"items"`
	Total int     `json:"total"`
}

// BinPage is the answer of GET /api/recycle-bin (RecycleBinPage): everything in the recycle bin, newest
// first.
type BinPage struct {
	Items []BinEntryView `json:"items"`
	Total int            `json:"total"`
}

// View returns the entry as the API returns it. Every delete answers it, in a DeleteResult, so the web app
// can offer Undo.
func (e BinEntry) View() BinEntryView {
	return BinEntryView{
		ID:              e.ID,
		RecordType:      e.RecordType,
		RecordTypeLabel: e.RecordTypeLabel,
		RecordID:        e.RecordID,
		RecordName:      e.RecordName,
		Path:            e.Path,
		Parent:          e.Parent,
		DeletedAt:       e.DeletedAt,
		DeletedBy:       e.DeletedBy,
		Contents:        e.Contents,
	}
}
