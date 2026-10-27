package people

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// SetPersonStatus deactivates (active false) or reactivates (active true) the person id: people are never
// deleted (FR-005 – FR-008). A deactivated person can no longer sign in — feature 001's sign-in check ends
// their open sessions at their next request — and is no longer offered in pickers; every record that names
// them keeps them, and so do their teams. Nobody may deactivate themselves (FR-006). A person already in
// that status is answered unchanged: no new version, no log entry.
func (s *Service) SetPersonStatus(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (PersonDetails, error) {
	if err := checkStatusUpdate(update); err != nil {
		return PersonDetails{}, err
	}
	var details PersonDetails
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.storedPerson(ctx, id)
		if err != nil {
			return err
		}
		if err = s.checkDeactivation(ctx, a, p, *update.Active); err != nil {
			return err
		}
		if err = p.CheckVersion(*update.Version, personSingular); err != nil {
			return err
		}
		if p.Active != *update.Active {
			if p, err = s.changePersonStatus(ctx, a, p, *update.Active); err != nil {
				return err
			}
		}
		details, err = s.details(ctx, p)
		return err
	})
	if err != nil {
		return PersonDetails{}, err
	}
	return details, nil
}

// checkDeactivation refuses to deactivate the person acting themselves, or the last administrator through
// the access hook.
func (s *Service) checkDeactivation(ctx context.Context, a actor.Actor, p Person, active bool) error {
	if p.ID == a.ID && !active {
		return errs.NotAllowed(deactivateYourselfMessage)
	}
	if p.Active && !active && s.access.CanDeactivate != nil { // the last administrator (014 FR-021)
		return s.access.CanDeactivate(ctx, p.ID)
	}
	return nil
}

// changePersonStatus saves p with the status active, a new version and a "deactivated" or "reactivated"
// log entry, inside the caller's Write, and returns the person as they are stored now. Memberships are
// kept (FR-005, FR-007).
func (s *Service) changePersonStatus(ctx context.Context, a actor.Actor, p Person, active bool) (Person, error) {
	changed := p
	changed.Active = active
	changes, err := history.Diff(p.logView(nil), changed.logView(nil))
	if err != nil {
		return Person{}, err
	}
	changed.Touch(a, s.now())
	if err = s.repo.Put(ctx, changed); err != nil {
		return Person{}, err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: personKind, RecordID: changed.ID, RecordName: changed.Name,
		Action: history.ActionStatusChanged, Verb: statusVerb(active), Changes: changes,
	})
	return changed, err
}
