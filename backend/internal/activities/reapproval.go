package activities

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// RestartApproval sends the Submitted or Approved activity activityID back to Created (FR-027, D-196), for
// changes made elsewhere, such as feature 007's line-item changes: it runs inside the caller's Write,
// writes one status entry with the reasons and tells feature 008, which keeps each reason, a ready-made
// sentence, as one change of its approval history (008 plan, Reconciliation). An activity with another
// status is left as it is.
func (s *Service) RestartApproval(ctx context.Context, a actor.Actor, activityID string, reasons []string) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		activity, found, err := s.repo.Get(ctx, activityID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(activityGoneMessage)
		}
		if !inApproval(activity) {
			return nil
		}
		before := activity
		activity.Status = StatusCreated
		activity.Touch(a, s.now())
		if err := s.repo.Put(ctx, activity); err != nil {
			return err
		}
		changes := make([]history.Change, 0, len(reasons))
		for _, reason := range reasons {
			changes = append(changes, history.Change{Label: reason}) // no old or new text: the label says it all
		}
		return s.logRestart(ctx, a, before, activity, changes, strings.Join(reasons, " "))
	})
}

// logRestart writes the status entry of a restarted approval (Submitted or Approved → Created) with its
// details, and tells feature 008 through the approval hooks, in the caller's Write. changes are the
// changes that caused it, with their labels and texts.
func (s *Service) logRestart(ctx context.Context, a actor.Actor, before, after Activity, changes []history.Change, details string) error {
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionStatusChanged, Verb: restartVerb,
		Changes: []history.Change{{Field: "status", Old: mustJSON(before.Status), New: mustJSON(after.Status)}},
		Details: details,
	})
	if err != nil {
		return err
	}
	return s.hooks.ApprovalRestarted(ctx, a, before, after, changes)
}

// inApproval reports whether a re-approval change of a would restart its approval: it is Submitted or
// Approved (shown also as In progress or Complete).
func inApproval(a Activity) bool {
	return a.Status == StatusSubmitted || a.Status == StatusApproved
}

// reapprovalWords are the fields whose change restarts approval (domain model §2.5), in the order the
// reason names them.
var reapprovalWords = []struct{ field, words string }{
	{"plannedCost", "the planned cost"},
	{"startDate", "the start date"},
	{"endDate", "the end date"},
	{"ongoing", "ongoing (no end date)"},
}

// reapprovalFields returns, in words, the changed fields that restart the approval of before; none when
// before is not Submitted or Approved.
func reapprovalFields(before Activity, changes []history.Change) []string {
	if !inApproval(before) {
		return nil
	}
	var words []string
	for _, w := range reapprovalWords {
		if slices.ContainsFunc(changes, func(c history.Change) bool { return c.Field == w.field }) {
			words = append(words, w.words)
		}
	}
	return words
}

// joinWords joins words as a sentence does: "a", "a and b", "a, b and c".
func joinWords(words []string) string {
	if len(words) <= 1 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}
