package history

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"mapp/internal/platform/actor"
)

// Log writes one activity-log entry about a change of a record (FR-022) and returns it. Call it inside the
// Write that makes the change: Log joins that Write, so the change and its entry are saved together, or
// neither is. History adds the id and sequence number, the time, the actor, the kind's label, the record's
// path and the readable label and texts of every change.
//
// A mistake in the calling code is an error, and nothing is written: a kind that is not registered, an
// unknown action, a status change without its verb, or no ParentID for a record that has a parent but is
// no longer stored.
func (s *Service) Log(ctx context.Context, a actor.Actor, in LogInput) (Entry, error) {
	var entry Entry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		k, err := s.registeredKind(in.Kind)
		if err != nil {
			return err
		}
		verb, err := verbOf(in.Action, in.Verb)
		if err != nil {
			return err
		}
		path, err := s.pathOf(ctx, k, in.RecordID, in.ParentID)
		if err != nil {
			return err
		}
		changes, err := describeChanges(ctx, k, in.Changes)
		if err != nil {
			return err
		}
		entry, err = s.appendEntry(ctx, a, Entry{
			Action:          in.Action,
			Verb:            verb,
			RecordType:      k.Name,
			RecordTypeLabel: k.Singular,
			RecordID:        in.RecordID,
			RecordName:      in.RecordName,
			Path:            path,
			Changes:         changes,
			Comment:         in.Comment,
			Details:         in.Details,
		})
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	return entry, nil
}

// appendEntry adds e at the end of the activity log, with its id and sequence number, the time and the
// actor a, and returns it as stored. Call it inside a Write. Log uses it, and so do Purge and EmptyBin,
// whose records are no longer stored. A dry run (withLogCollector) also gets a copy of the entry.
func (s *Service) appendEntry(ctx context.Context, a actor.Actor, e Entry) (Entry, error) {
	// One number gives both the id and the sequence number, which orders entries saved in the same
	// millisecond (spec edge case).
	seq, err := s.ids.NextNumber(ctx, logPrefix)
	if err != nil {
		return Entry{}, err
	}
	e.ID = fmt.Sprintf("%s-%d", logPrefix, seq) // the form of ids.Generator.Next: LOG-1001
	e.Seq, e.At, e.Actor = seq, s.now(), a
	if e.Path == nil {
		e.Path = []RecordRef{}
	}
	if e.Changes == nil {
		e.Changes = []Change{}
	}
	if err := s.repo.AppendEntry(ctx, e); err != nil {
		return Entry{}, err
	}
	if c, ok := ctx.Value(collectorKey{}).(*logCollector); ok {
		c.entries = append(c.entries, e)
	}
	return e, nil
}

// logCollector keeps a copy of every entry appended while it is in the context: the restore preview reads
// what a dry run of a restore would log, including the entries the kinds' hooks write (D-177).
type logCollector struct {
	entries []Entry
}

// collectorKey is the context key of the log collector.
type collectorKey struct{}

// withLogCollector returns a copy of ctx with a new, empty log collector, and the collector.
func withLogCollector(ctx context.Context) (context.Context, *logCollector) {
	c := &logCollector{}
	return context.WithValue(ctx, collectorKey{}, c), c
}

// verbOf returns the verb of an entry's sentence "<person> <verb> <record type> '<name>'". Each action has a
// verb of the same name ("created", "deleted", …), used when the caller names no other; the two actions of
// feature 011 read "permanently deleted" and "emptied the recycle bin". A status change has none: it must
// name its own, such as "published".
func verbOf(action Action, verb string) (string, error) {
	switch action {
	case ActionCreated, ActionChanged, ActionMoved, ActionCopied, ActionDeleted, ActionRestored:
		return cmp.Or(verb, string(action)), nil // cmp.Or returns the first value that is not empty
	case ActionDeletedPermanently:
		return cmp.Or(verb, verbDeletedPermanently), nil
	case ActionEmptiedRecycleBin:
		return cmp.Or(verb, verbEmptiedRecycleBin), nil
	case ActionStatusChanged:
		if verb == "" {
			return "", errors.New(`history: a status change needs its verb, such as "published"`)
		}
		return verb, nil
	default:
		return "", fmt.Errorf("history: %q is not an action", action)
	}
}

// pathOf returns where the record recordID of kind k is: its ancestors as they are stored now, top level
// first, or an empty list for a top-level record. The path starts at the parent the record's snapshot
// names; for a record that is no longer stored, it starts at parentID.
func (s *Service) pathOf(ctx context.Context, k Kind, recordID, parentID string) ([]RecordRef, error) {
	snapshot, found, err := k.Snapshot(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if found {
		parentID = snapshot.ParentID
	} else if k.Parent != "" && parentID == "" {
		return nil, fmt.Errorf("history: the %s %s is not stored, so the log needs its ParentID to say where it was",
			k.Singular, recordID)
	}
	return s.ancestors(ctx, k.Parent, parentID)
}

// ancestors returns the record id of the kind called kindName followed upwards by its own ancestors, as a
// path: top level first. An empty kindName (the parent of a top-level record) gives an empty path.
func (s *Service) ancestors(ctx context.Context, kindName, id string) ([]RecordRef, error) {
	return s.newReadSnapshots().ancestors(ctx, kindName, id)
}

// describeChanges returns changes with the label and texts people read, fixed at the time of the change
// (data model §5). The list is never nil.
func describeChanges(ctx context.Context, k Kind, changes []Change) ([]Change, error) {
	described := make([]Change, 0, len(changes))
	for _, c := range changes {
		c.Old, c.New = orNull(c.Old), orNull(c.New) // a missing value is stored as null
		label, oldText, err := describe(ctx, k, c.Field, c.Old)
		if err != nil {
			return nil, err
		}
		_, newText, err := describe(ctx, k, c.Field, c.New)
		if err != nil {
			return nil, err
		}
		c.Label, c.OldText, c.NewText = label, oldText, newText
		described = append(described, c)
	}
	return described, nil
}

// describe returns the label of field and the text of value: what the kind's Describe gives, or else the
// defaults, the field name in words and the value as people read it (data model §5).
func describe(ctx context.Context, k Kind, field string, value json.RawMessage) (label, text string, err error) {
	decoded, err := decode(value)
	if err != nil {
		return "", "", err
	}
	if k.Describe != nil {
		label, text, err = k.Describe(ctx, field, decoded)
		if err != nil {
			return "", "", err
		}
	}
	return cmp.Or(label, fieldLabel(field)), cmp.Or(text, defaultText(decoded)), nil
}

// ListLog returns the newest activity-log entries that match filter and that a may see (014 FR-007, FR-038,
// D-238), at most filter.Limit of them, and how many entries match in all. The limit must be from 1 to 200.
// It is QueryLog with fewer filters, kept for the callers of feature 001.
func (s *Service) ListLog(ctx context.Context, a actor.Actor, filter LogFilter) ([]Entry, int, error) {
	q := LogQuery{RecordID: filter.RecordID, Limit: filter.Limit}
	if filter.RecordType != "" {
		q.RecordTypes = []string{filter.RecordType}
	}
	page, err := s.QueryLog(ctx, a, q)
	if err != nil {
		return nil, 0, err
	}
	entries := make([]Entry, 0, len(page.Items))
	for _, item := range page.Items {
		entries = append(entries, item.Entry)
	}
	return entries, page.Total, nil
}

// Entries returns every activity-log entry, oldest first, without any access filter: for other features
// that select entries of their own, such as the recent access changes of feature 014. It joins the caller's
// transaction.
func (s *Service) Entries(ctx context.Context) ([]Entry, error) {
	var entries []Entry
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		entries, err = s.repo.Entries(ctx)
		return err
	})
	return entries, err
}
