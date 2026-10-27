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
	"mapp/internal/platform/store"
)

// Messages of the plan's "User-visible messages" and "Validation messages" tables.
const (
	activityGoneMessage = "This activity was deleted — restore it from the recycle bin to keep working on its line items."
	lineItemGoneMessage = "This line item no longer exists — someone may have deleted it."
	restartMessage      = "This activity will need approval again. Confirm the change to send it back for approval."
	reloadMessage       = "Reload the page and try again."
)

// The restart reasons sent to feature 006 (plan, "Restart reasons").
const (
	addedReason   = "Line item '%s' (%s) added: %s %s"
	deletedReason = "Line item '%s' (%s) deleted: %s %s"
	changedReason = "Planned cost of line item '%s' (%s): %s %s → %s %s"
)

// Service holds the rules of line items (feature 007).
type Service struct {
	tx      store.Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	deps    Deps
	now     clock.Clock
}

// NewService returns the line-items service. It keeps line items in repo, gives them readable IDs, logs
// their changes, deletes them and works in transactions through tools, and reads the other features through
// deps.
func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, deps: deps, now: tools.Now}
}

// gone answers a request about the line item id that is not stored (D-154): "This activity was deleted — …"
// when it is in the recycle bin inside its activity's (or a level above's) entry, otherwise "This line item
// no longer exists — …".
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

// siblings returns the other line items of li's activity.
func (s *Service) siblings(ctx context.Context, li LineItem) ([]LineItem, error) {
	items, err := s.repo.ListByActivity(ctx, li.ActivityID)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(items, func(other LineItem) bool { return other.ID == li.ID }), nil
}

// changes lists what differs between two versions of a line item, for the log (data model §7.3): every
// changed field, and each Shell attribute as a field of its own, "attributes.<id>" (as 002's Diff).
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

// sameValue reports whether two stored attribute values are the same: equal JSON, ignoring spaces, or both
// empty.
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

// confirmRestart returns the problems of a form, or, when there are none, refuses a change that would send a
// Submitted or Approved activity back for approval without the person's confirmation (FR-031, D-146, D-189).
func confirmRestart(problems errs.Problems, f ActivityFacts, confirmed bool) error {
	if err := problems.Err(); err != nil {
		return err
	}
	if f.restartsApproval() && !confirmed {
		return errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage})
	}
	return nil
}

// restart sends the activity back to Created with reason when it is Submitted or Approved (FR-031), and
// returns its facts as they are afterwards.
func (s *Service) restart(ctx context.Context, a actor.Actor, f ActivityFacts, reason string) (ActivityFacts, error) {
	if !f.restartsApproval() {
		return f, nil
	}
	if err := s.deps.Activities.RestartApproval(ctx, a, f.ID, []string{reason}); err != nil {
		return ActivityFacts{}, err
	}
	return s.facts(ctx, f.ID)
}

// facts returns the facts of the activity activityID, or the "activity was deleted" answer (404) when it is
// not stored.
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

// uniqueName returns the first of base + suffix(1), base + suffix(2), … that no line item among items has
// (ignoring letter case). With a limit above 0, base is shortened so that the name fits limit characters.
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

// shorten returns the first n characters of text, without spaces at their end.
func shorten(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return strings.TrimRight(string([]rune(text)[:max(n, 0)]), " ")
}
