package history

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"mapp/internal/platform/actor"
)

// Log writes one activity-log entry about a change of a record (FR-022) inside the Write that makes the
// change, so both are saved or neither; a mistake of the caller (an unregistered kind) writes nothing.
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

// appendEntry stores e with its id, sequence number, time and actor; Purge and EmptyBin call it too.
func (s *Service) appendEntry(ctx context.Context, a actor.Actor, e Entry) (Entry, error) {
	seq, err := s.ids.NextNumber(ctx, logPrefix)
	if err != nil {
		return Entry{}, err
	}
	e.ID = fmt.Sprintf("%s-%d", logPrefix, seq)
	e.Seq, e.At, e.Actor = seq, s.now(), a
	e.Details = withNote(ctx, e.Details)
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

// logCollector keeps a copy of every entry a dry run of a restore appends (D-177).
type logCollector struct {
	entries []Entry
}

type collectorKey struct{}

func withLogCollector(ctx context.Context) (context.Context, *logCollector) {
	c := &logCollector{}
	return context.WithValue(ctx, collectorKey{}, c), c
}

// verbOf returns the verb of "<person> <verb> <record type>"; a status change must name its own.
func verbOf(action Action, verb string) (string, error) {
	switch action {
	case ActionCreated, ActionChanged, ActionMoved, ActionCopied, ActionDeleted, ActionRestored:
		return cmp.Or(verb, string(action)), nil
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

// pathOf starts at the parent the stored snapshot names, or at parentID for a record no longer stored.
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

func (s *Service) ancestors(ctx context.Context, kindName, id string) ([]RecordRef, error) {
	return s.newReadSnapshots().ancestors(ctx, kindName, id)
}

func describeChanges(ctx context.Context, k Kind, changes []Change) ([]Change, error) {
	described := make([]Change, 0, len(changes))
	for _, c := range changes {
		c.Old, c.New = orNull(c.Old), orNull(c.New)
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

// ListLog is QueryLog with the filters of feature 001.
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

// Entries returns every entry, oldest first, without access filter, for features that select their own.
func (s *Service) Entries(ctx context.Context) ([]Entry, error) {
	var entries []Entry
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		entries, err = s.repo.Entries(ctx)
		return err
	})
	return entries, err
}
