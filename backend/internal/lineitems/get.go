package lineitems

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// Get returns the line item id with its activity and the activity's unassigned planned cost now (FR-026),
// its amounts in USD answered in a's display currency (FR-005, D-213).
// A line item that is gone answers 404 with why (D-154): its activity was deleted (it went to the recycle bin
// with it), or the line item itself no longer exists.
func (s *Service) Get(ctx context.Context, a actor.Actor, id string) (Details, error) {
	var details Details
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		li, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		f, err := s.facts(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		items, err := s.repo.ListByActivity(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		views, err := s.views(ctx, f, items)
		if err != nil {
			return err
		}
		d, err := s.displayOf(ctx, a)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(views, func(v View) bool { return v.ID == id })
		details = Details{LineItem: d.view(views[i]), Activity: activityView(f), Balance: d.balance(balanceOf(f, views)),
			Access: authz.LineItemAccessOf(ctx, f.Access)} // 014 FR-011
		return nil
	})
	if err != nil {
		return Details{}, err
	}
	return details, nil
}
