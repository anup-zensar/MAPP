package lineitem

import (
	"context"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// Update saves the whole form of the line item id (FR-027, D-153): the version it started from is needed
// (missing: reported with every other problem; out of date: version_conflict naming who changed it); an
// inactive owner, currency or finance budget item already stored may stay (FR-011, FR-016); the custom months
// are kept only for a custom spread (FR-018) and stay when the cost or currency changes (FR-022). A save that
// changes nothing creates no version and no log entry. Changing the planned cost or the currency of a line
// item of a Submitted or Approved activity needs u.RestartApproval and sends the activity back to Created
// (FR-031); other changes never change its status (FR-032). One "changed" entry lists every change, old and
// new.
func (s *Service) Update(ctx context.Context, a actor.Actor, id string, u Update) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		before, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		f, after, err := s.checkUpdate(ctx, before, u)
		if err != nil {
			return err
		}
		changes, err := s.changes(before, after)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			view, err = s.shownView(ctx, a, f, before)
			return err
		}
		view, err = s.saveEdit(ctx, a, f, edit{before: before, after: after, changes: changes, restartConfirmed: u.RestartApproval})
		return err
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}

// checkUpdate returns the facts of before's activity and before as u would save it, or every problem at once.
func (s *Service) checkUpdate(ctx context.Context, before LineItem, u Update) (ActivityFacts, LineItem, error) {
	f, err := s.facts(ctx, before.ActivityID)
	if err != nil {
		return ActivityFacts{}, LineItem{}, err
	}
	siblings, err := s.siblings(ctx, before)
	if err != nil {
		return ActivityFacts{}, LineItem{}, err
	}
	after, problems, err := s.check(ctx, f, siblings, u.input(), &before)
	if err != nil {
		return ActivityFacts{}, LineItem{}, err
	}
	if u.Version == nil || *u.Version < 1 {
		problems.Add("version", reloadMessage)
	}
	if err = problems.Err(); err != nil {
		return ActivityFacts{}, LineItem{}, err
	}
	if err = before.CheckVersion(*u.Version, lineItemSingular); err != nil {
		return ActivityFacts{}, LineItem{}, err
	}
	after.ID, after.ActivityID, after.Meta = before.ID, before.ActivityID, before.Meta
	return f, after, nil
}

// edit is a save that changes a line item, and whether the person confirmed that it restarts the approval.
type edit struct {
	before, after    LineItem
	changes          []history.Change
	restartConfirmed bool
}

// saveEdit saves and logs e; a new planned cost or currency restarts the activity's approval.
func (s *Service) saveEdit(ctx context.Context, a actor.Actor, f ActivityFacts, e edit) (View, error) {
	before, after := e.before, e.after
	costChanged := after.PlannedCost != before.PlannedCost || after.Currency != before.Currency
	if costChanged {
		if err := confirmRestart(nil, f, e.restartConfirmed); err != nil {
			return View{}, err
		}
	}
	after.Touch(a, s.now())
	if err := s.repo.Put(ctx, after); err != nil {
		return View{}, err
	}
	if _, err := s.history.Log(ctx, a, history.LogInput{
		Kind: lineItemKind, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Changes: e.changes,
	}); err != nil {
		return View{}, err
	}
	if !costChanged {
		return s.shownView(ctx, a, f, after)
	}
	reason := fmt.Sprintf(changedReason, after.Name, after.ID, before.PlannedCost.Format(), before.Currency,
		after.PlannedCost.Format(), after.Currency)
	restarted, err := s.restart(ctx, a, f, reason)
	if err != nil {
		return View{}, err
	}
	return s.shownView(ctx, a, restarted, after)
}
