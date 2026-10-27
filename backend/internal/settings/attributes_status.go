package settings

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// SetAttributeActive sets the Shell attribute id inactive or reactivates it (FR-004, FR-023, US2-7): forms
// stop offering an inactive attribute, and values already on records stay. The Business unit attribute,
// an attribute that active attributes depend on, and a dependent of an inactive attribute (reactivating)
// are refused with the lock's reason. The same status again changes nothing.
func (s *Service) SetAttributeActive(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (ShellAttributeView, error) {
	active, version, statusErr := checkStatus(update)
	if statusErr != nil {
		return ShellAttributeView{}, statusErr
	}
	var view ShellAttributeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, stored, u, err := s.attributeToChange(ctx, id)
		if err != nil {
			return err
		}
		if err = stored.CheckVersion(version, attributeNoun); err != nil {
			return err
		}
		if stored.Active != active {
			if reason, locked := lockOf(attributeLocks(stored, u), fieldActive); locked {
				return errs.Validation(errs.FieldError{Field: fieldActive, Message: reason})
			}
			stored.Active = active
			stored.Touch(a, s.now())
			if err = s.repo.PutAttribute(ctx, stored); err != nil {
				return err
			}
			if err = s.logStatus(ctx, a, attributeKind, stored.ID, stored.Label, active); err != nil {
				return err
			}
		}
		view, err = s.attributeViewOf(ctx, id)
		return err
	})
	return view, err
}

// DeleteAttribute moves the Shell attribute id to the recycle bin with its options, so that it can be
// undone (FR-004, US2-7). The Business unit attribute is refused with its lock's reason, and an attribute
// in use — values on records (also in the recycle bin), dependents, conditions — with in_use: set it
// inactive instead.
func (s *Service) DeleteAttribute(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, stored, u, err := s.attributeToChange(ctx, id)
		if err != nil {
			return err
		}
		if reason, locked := lockOf(attributeLocks(stored, u), actionDelete); locked {
			return errs.InUse(reason)
		}
		if usage := u.attribute(stored); usage.InUse {
			return inUseError(stored.Label, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, attributeKind, id)
		return err
	})
	return entry, err
}
