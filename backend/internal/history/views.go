package history

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// The API types in this file are what the API returns. They are kept apart from the stored records
// (model.go), so the stored data can change without changing the API (research R10).

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

// LogPage is the answer of GET /api/v1/activity-log (ActivityLogPage): the newest matching entries, how many
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

// LogSummary is the answer of GET /api/v1/activity-log/summary (ActivityLogSummary): the summary cards of the
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

// BinPage is the answer of GET /api/v1/recycle-bin (RecycleBinPage): everything in the recycle bin, newest
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

// RestorePreview is the answer of GET /api/v1/recycle-bin/{entryId}/restore-preview: what a restore would do,
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

// PurgeResult is the answer of DELETE /api/v1/recycle-bin/{entryId}: the record deleted permanently.
type PurgeResult struct {
	RecordType      string `json:"recordType"`
	RecordTypeLabel string `json:"recordTypeLabel"`
	RecordID        string `json:"recordId"`
	RecordName      string `json:"recordName"`
}

// EmptyRequest is the body of POST /api/v1/recycle-bin/empty (EmptyRecycleBinRequest): the entries the person
// saw in the confirmation (D-180).
type EmptyRequest struct {
	EntryIDs []string `json:"entryIds"`
}

// EmptyResult is the answer of POST /api/v1/recycle-bin/empty (EmptyRecycleBinResult): how many entries were
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
