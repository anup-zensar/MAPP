// Package history is NorthStar's activity log and recycle bin core (FR-022 – FR-027, research R8,
// ADR-0007). It writes one activity-log entry for every business change, and turns every delete into a
// move to the recycle bin: the record and everything under it are kept there, and Undo or Restore brings
// them back. Feature 011 adds what the screens need: the activity-log query and summary (query.go), and the
// recycle-bin list with restore states, the restore preview, Restore with its parents, Delete permanently
// and Empty recycle bin (bin.go).
//
// Every business package registers a Kind for each record type it owns (in internal/app), logs its changes
// with Service.Log inside the Write that makes the change, and deletes with Service.MoveToBin.
package history

import (
	"encoding/json"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
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

// The actions of feature 001, and the two of feature 011.
const (
	ActionCreated       Action = "created"
	ActionChanged       Action = "changed"
	ActionStatusChanged Action = "statusChanged"
	ActionMoved         Action = "moved"
	ActionCopied        Action = "copied"
	ActionDeleted       Action = "deleted"
	ActionRestored      Action = "restored"

	ActionDeletedPermanently Action = "deletedPermanently" // one entry per Delete permanently (011 FR-030)
	ActionEmptiedRecycleBin  Action = "emptiedRecycleBin"  // one entry per Empty recycle bin (011 FR-030)
)

// Entry is one activity-log entry (data model §5), stored in the append-only table "activity_log". Entries
// are only ever added, never changed or removed (FR-023). The API returns them as they are stored, with
// where the record is now (EntryView, ActivityLogEntry).
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
	// Items lists the entries an Empty recycle bin deleted permanently (011 data model §1). Every other
	// entry has none and stores none; the API answers them with an empty list.
	Items []LogItem `json:"items,omitempty"`
}

// LogItem is one recycle-bin entry deleted permanently by an Empty recycle bin, as its log entry lists it.
type LogItem struct {
	RecordType      string `json:"recordType"`
	RecordTypeLabel string `json:"recordTypeLabel"`
	RecordID        string `json:"recordId"`
	RecordName      string `json:"recordName"`
	Details         string `json:"details"` // what was inside it: "with 1 activity and 1 line item"; "" when nothing
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

// LogQuery chooses the activity-log entries QueryLog returns (011 FR-003, FR-009, FR-012). Every filter
// applies to the whole log, and all of them together: an entry must match each one that is set.
type LogQuery struct {
	ActorIDs      []string   // only entries by these people; empty = anyone
	RecordTypes   []string   // only entries about these kinds; empty = every kind
	Actions       []string   // the Action filter values (ActionFilters); empty = every action
	PlanIDs       []string   // only entries about these plans and anything that is or was inside them; empty = any plan
	RecordID      string     // only entries about this record; "" = any record
	IncludeInside bool       // with RecordID: also the entries of every record that is or was below it
	From          *time.Time // only entries at or after this moment; nil = no start
	To            *time.Time // only entries before this moment (exclusive); nil = no end
	Search        string     // part of the record's name, the record's ID or the person's name, ignoring letter case
	BeforeSeq     int        // only entries with a smaller seq (Load more); 0 = from the newest
	Limit         int        // at most this many entries: 1 to 200
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

// LogPage is the answer of GET /api/activity-log (ActivityLogPage): the newest matching entries, how many
// entries match in all, how many the whole log holds, and the record the query names.
type LogPage struct {
	Items           []EntryView `json:"items"`
	Total           int         `json:"total"`           // the matching entries, ignoring BeforeSeq and Limit
	UnfilteredTotal int         `json:"unfilteredTotal"` // every entry of the log ("Showing N of M entries")
	Record          *RecordRef  `json:"record"`          // the record of LogQuery.RecordID; null without one or when unknown
}

// EntryView is an activity-log entry as the API returns it (ActivityLogEntry): the stored entry, always with
// its list of items, and where its record is now (011 FR-008).
type EntryView struct {
	Entry
	Items     []LogItem `json:"items"` // the entry's items; [] instead of nothing (it hides Entry.Items in the JSON)
	RecordNow RecordNow `json:"recordNow"`
}

// RecordState says where the record of an activity-log entry is now (011 data model §4).
type RecordState string

// The record states.
const (
	RecordExists             RecordState = "exists"             // stored: the name links to the record
	RecordInRecycleBin       RecordState = "inRecycleBin"       // in an entry of the recycle bin
	RecordDeletedPermanently RecordState = "deletedPermanently" // gone for good
	RecordNone               RecordState = "none"               // no record: an emptied recycle bin, or a kind this API does not know
)

// RecordNow is where the record of an activity-log entry is now (011 FR-008), worked out when the entry is
// read.
type RecordNow struct {
	State      RecordState `json:"state"`
	Name       string      `json:"name"`       // the current name (exists) or the name in the recycle bin; "" otherwise
	Path       []RecordRef `json:"path"`       // the current ancestors, top level first, when it exists; [] otherwise
	BinEntryID string      `json:"binEntryId"` // the recycle-bin entry holding it; "" otherwise
}

// LogSummary is the answer of GET /api/activity-log/summary (ActivityLogSummary): the summary cards of the
// activity log, counted over the whole log (011 FR-002), and the choices of its filters (FR-003).
type LogSummary struct {
	ChangesToday         int               `json:"changesToday"`
	ChangesThisWeek      int               `json:"changesThisWeek"`
	PeopleActiveThisWeek int               `json:"peopleActiveThisWeek"`
	DeletionsThisWeek    int               `json:"deletionsThisWeek"` // deletes, permanent deletes and emptied recycle bins
	Total                int               `json:"total"`
	People               []actor.Actor     `json:"people"`      // everyone with an entry, sorted by name
	RecordTypes          []RecordTypeCount `json:"recordTypes"` // the kinds in the log, sorted by label
	Plans                []RecordRef       `json:"plans"`       // the plans named in the log with their latest name, sorted by name
}

// RecordTypeCount is one record type of the log, with how many entries are about it.
type RecordTypeCount struct {
	Type  string `json:"type"`
	Label string `json:"label"` // the kind's latest mid-sentence name in the log
	Count int    `json:"count"`
}

// BinPage is the answer of GET /api/recycle-bin (RecycleBinPage): everything in the recycle bin, newest
// first.
type BinPage struct {
	Items []BinListItem `json:"items"`
	Total int           `json:"total"`
}

// The restore states of a recycle-bin entry (011 data model §5).
const (
	Restorable               = "restorable"               // no parent, or the parent is stored
	ParentInRecycleBin       = "parentInRecycleBin"       // the parent is in another entry: Restore both / all N
	ParentDeletedPermanently = "parentDeletedPermanently" // the parent, or a record above it, is gone for good
)

// BinListItem is a recycle-bin entry as the recycle bin page lists it (RecycleBinListItem): the entry, its
// restore state and the records it holds (011 FR-015 – FR-017).
type BinListItem struct {
	BinEntryView
	RecordTypePlural string       `json:"recordTypePlural"` // the kind's plural, for the type pills
	RestoreState     string       `json:"restoreState"`     // Restorable, ParentInRecycleBin or ParentDeletedPermanently
	MissingParent    *LabelledRef `json:"missingParent"`    // the parent in the bin, or the record gone for good; null when restorable
	ParentEntryIDs   []string     `json:"parentEntryIds"`   // the entries Restore both / all N brings back first, top-most first
	InsideEntryIDs   []string     `json:"insideEntryIds"`   // the other entries that can't be restored once this one is purged
	Records          []BinRecord  `json:"records"`          // the record, then everything inside it, parents before children
	// Access is what the person may do with the entry (014 FR-011, D-238); left out for an internal call.
	Access *authz.BinEntryAccess `json:"access,omitempty"`
}

// LabelledRef is a record with the mid-sentence name of its kind.
type LabelledRef struct {
	Type      string `json:"type"`
	TypeLabel string `json:"typeLabel"`
	ID        string `json:"id"`
	Name      string `json:"name"`
}

// BinRecord is one record a recycle-bin entry holds, without its data (RecycleBinRecord).
type BinRecord struct {
	RecordType      string `json:"recordType"`
	RecordTypeLabel string `json:"recordTypeLabel"`
	RecordID        string `json:"recordId"`
	RecordName      string `json:"recordName"`
	ParentID        string `json:"parentId"` // "" for top-level records
}

// RestorePreview is the answer of GET /api/recycle-bin/{entryId}/restore-preview: what a restore would do,
// worked out by doing it and throwing the result away (011 FR-018, D-177).
type RestorePreview struct {
	Entry         BinListItem    `json:"entry"`
	Destination   []RecordRef    `json:"destination"`   // the parent's current path, the parent last; [] for top-level records
	ParentEntries []BinListItem  `json:"parentEntries"` // the entries restored first, top-most first
	Changes       []RecordChange `json:"changes"`       // every changed field of every record, in restore order
	Notes         []string       `json:"notes"`         // the notes of the kinds' restore rules, in order
}

// RecordChange is one changed field of one record in a restore preview (RestorePreviewChange).
type RecordChange struct {
	RecordType      string `json:"recordType"`
	RecordTypeLabel string `json:"recordTypeLabel"`
	RecordID        string `json:"recordId"`
	RecordName      string `json:"recordName"`
	Restored        bool   `json:"restored"` // true: the record comes back; false: a stored record the restore changes
	Field           string `json:"field"`
	Label           string `json:"label"`
	OldText         string `json:"oldText"`
	NewText         string `json:"newText"`
}

// PurgeResult is the answer of DELETE /api/recycle-bin/{entryId}: the record deleted permanently.
type PurgeResult struct {
	RecordType      string `json:"recordType"`
	RecordTypeLabel string `json:"recordTypeLabel"`
	RecordID        string `json:"recordId"`
	RecordName      string `json:"recordName"`
}

// EmptyRequest is the body of POST /api/recycle-bin/empty (EmptyRecycleBinRequest): the entries the person
// saw in the confirmation (D-180).
type EmptyRequest struct {
	EntryIDs []string `json:"entryIds"`
}

// EmptyResult is the answer of POST /api/recycle-bin/empty (EmptyRecycleBinResult): how many entries were
// deleted permanently.
type EmptyResult struct {
	Count int `json:"count"`
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
