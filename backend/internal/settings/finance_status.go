package settings

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// SetFinanceBudgetItemActive sets the finance budget item id inactive or reactivates it (FR-004): line
// items cannot link to an inactive item; those linked keep it. The same status again changes nothing.
func (s *Service) SetFinanceBudgetItemActive(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (FinanceBudgetItemView, error) {
	active, version, statusErr := checkStatus(update)
	if statusErr != nil {
		return FinanceBudgetItemView{}, statusErr
	}
	var view FinanceBudgetItemView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		item, err := s.storedItem(ctx, id)
		if err != nil {
			return err
		}
		if err = item.CheckVersion(version, itemNoun); err != nil {
			return err
		}
		if item.Active != active {
			item.Active = active
			item.Touch(a, s.now())
			if err = s.repo.PutFinanceBudgetItem(ctx, item); err != nil {
				return err
			}
			if err = s.logStatus(ctx, a, itemKind, item.ID, item.Name, active); err != nil {
				return err
			}
		}
		view, err = s.itemViewOf(ctx, item)
		return err
	})
	return view, err
}

// DeleteFinanceBudgetItem moves the finance budget item id to the recycle bin, under its budget, so that it
// can be undone (FR-004, US5-5). An item linked to line items — also those in the recycle bin — is refused
// with in_use: set it inactive instead. Its budget stays (D-047).
func (s *Service) DeleteFinanceBudgetItem(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		item, err := s.storedItem(ctx, id)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		if usage := u.item(item.ID); usage.InUse {
			return inUseError(item.Name, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, itemKind, item.ID)
		return err
	})
	return entry, err
}
