package lineitems

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// Section returns the Line items section of the activity activityID (FR-001 – FR-004, FR-023): its line
// items in creation order, each with its USD amount at the currency's current rate, its spread, what is
// missing and its SWBS request status; the unassigned planned cost; and the spread grid. Nothing in USD is
// stored, so a rate change in Settings shows at once and changes no status (FR-013, D-141). The amounts the
// section shows in USD are answered in a's display currency (FR-005, D-213); every signed-in person may see
// it (D-004).
func (s *Service) Section(ctx context.Context, a actor.Actor, activityID string) (Section, error) {
	var section Section
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, activityID)
		if err != nil {
			return err
		}
		items, err := s.repo.ListByActivity(ctx, activityID)
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
		section = Section{
			Activity: activityView(f), Items: d.views(views), Balance: d.balance(balanceOf(f, views)), Grid: d.grid(Grid(f, views)),
			Access: authz.LineItemAccessOf(ctx, f.Access), // 014 FR-011
		}
		return nil
	})
	if err != nil {
		return Section{}, err
	}
	return section, nil
}
