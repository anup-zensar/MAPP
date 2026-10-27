package lineitem

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

const (
	activityGoneMessage = "This activity was deleted — restore it from the recycle bin to keep working on its line items."
	lineItemGoneMessage = "This line item no longer exists — someone may have deleted it."
	restartMessage      = "This activity will need approval again. Confirm the change to send it back for approval."
	reloadMessage       = "Reload the page and try again."
)

const (
	addedReason   = "Line item '%s' (%s) added: %s %s"
	deletedReason = "Line item '%s' (%s) deleted: %s %s"
	changedReason = "Planned cost of line item '%s' (%s): %s %s → %s %s"
)

type Service struct {
	tx      Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	deps    Deps
	now     clock.Clock
}

func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, deps: deps, now: tools.Now}
}

// gone says the activity was deleted when the line item is in the bin with it (or a level above), else it is gone.
func (s *Service) gone(ctx context.Context, a actor.Actor, id string) error {
	entries, err := s.history.ListBin(ctx, a)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		withActivity := entry.RecordType != lineItemKind && slices.ContainsFunc(entry.Snapshots, func(snapshot history.Snapshot) bool {
			return snapshot.Kind == lineItemKind && snapshot.ID == id
		})
		if withActivity {
			return errs.NotFound(activityGoneMessage)
		}
	}
	return errs.NotFound(lineItemGoneMessage)
}

func (s *Service) siblings(ctx context.Context, li LineItem) ([]LineItem, error) {
	items, err := s.repo.ListByActivity(ctx, li.ActivityID)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(items, func(other LineItem) bool { return other.ID == li.ID }), nil
}

// changes lists every changed field for the log, each Shell attribute as a field "attributes.<id>" of its own.
func (s *Service) changes(before, after LineItem) ([]history.Change, error) {
	b, x := before, after
	b.Attributes, x.Attributes = nil, nil
	changes, err := history.Diff(b, x)
	if err != nil {
		return nil, err
	}
	attributeIDs := slices.Sorted(maps.Keys(before.Attributes))
	for id := range after.Attributes {
		if !slices.Contains(attributeIDs, id) {
			attributeIDs = append(attributeIDs, id)
		}
	}
	slices.Sort(attributeIDs)
	for _, id := range attributeIDs {
		old, value := before.Attributes[id], after.Attributes[id]
		if !sameValue(old, value) {
			changes = append(changes, history.Change{Field: "attributes." + id, Old: old, New: value})
		}
	}
	return changes, nil
}

// sameValue compares stored attribute values as compact JSON; every kind of empty value is the same.
func sameValue(x, y json.RawMessage) bool {
	compact := func(raw json.RawMessage) string {
		var b bytes.Buffer
		if json.Compact(&b, raw) != nil {
			return string(raw)
		}
		switch b.String() {
		case "", "null", `""`, "[]", "{}":
			return ""
		}
		return b.String()
	}
	return compact(x) == compact(y)
}

// confirmRestart refuses, after the form's problems, a change that restarts approval without confirmation.
func confirmRestart(problems errs.Problems, f ActivityFacts, confirmed bool) error {
	if err := problems.Err(); err != nil {
		return err
	}
	if f.restartsApproval() && !confirmed {
		return errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage})
	}
	return nil
}

// restart sends a Submitted or Approved activity back to Created and returns its facts afterwards.
func (s *Service) restart(ctx context.Context, a actor.Actor, f ActivityFacts, reason string) (ActivityFacts, error) {
	if !f.restartsApproval() {
		return f, nil
	}
	if err := s.deps.Activities.RestartApproval(ctx, a, f.ID, []string{reason}); err != nil {
		return ActivityFacts{}, err
	}
	return s.facts(ctx, f.ID)
}

// facts answers "activity was deleted" (404) for an activity that is not stored.
func (s *Service) facts(ctx context.Context, activityID string) (ActivityFacts, error) {
	f, found, err := s.deps.Activities.Facts(ctx, activityID)
	if err != nil {
		return ActivityFacts{}, err
	}
	if !found {
		return ActivityFacts{}, errs.NotFound(activityGoneMessage)
	}
	return f, nil
}

// uniqueName returns the first base + suffix(n) no line item has, ignoring case; a limit above 0 shortens base.
func uniqueName(base string, suffix func(n int) string, items []LineItem, limit int) string {
	for n := 1; ; n++ {
		end := suffix(n)
		start := base
		if limit > 0 {
			start = shorten(base, limit-utf8.RuneCountInString(end))
		}
		if _, taken := named(items, start+end); !taken {
			return start + end
		}
	}
}

func shorten(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return strings.TrimRight(string([]rune(text)[:max(n, 0)]), " ")
}
