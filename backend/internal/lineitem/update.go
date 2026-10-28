package lineitem

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// Update saves the whole form of the line item id with the version it started from (D-153): a save that changes
// nothing makes no version and no log entry; no change, not even of the cost or currency, changes the activity's
// status (D-322); the SWBS code and its times are kept (030 FR-003, FR-004).
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
		view, err = s.saveEdit(ctx, a, f, edit{after: after, changes: changes})
		return err
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}

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
	checkSwbsKept(&problems, before, u.RequestSwbs)
	if err = s.requestedRule(ctx, &problems, before.ID, after); err != nil {
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
	after.ID, after.ActivityID, after.Swbs, after.Meta = before.ID, before.ActivityID, before.Swbs, before.Meta
	return f, after, nil
}

type edit struct {
	after   LineItem
	changes []history.Change
}

func (s *Service) saveEdit(ctx context.Context, a actor.Actor, f ActivityFacts, e edit) (View, error) {
	after := e.after
	after.Touch(a, s.now())
	if err := s.repo.Put(ctx, after); err != nil {
		return View{}, err
	}
	if _, err := s.history.Log(ctx, a, history.LogInput{
		Kind: lineItemKind, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Changes: e.changes,
	}); err != nil {
		return View{}, err
	}
	return s.shownView(ctx, a, f, after)
}
