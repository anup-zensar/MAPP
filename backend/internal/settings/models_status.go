package settings

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// SetModelActive sets the model id inactive or reactivates it (FR-004): plans using an inactive model keep
// it and still calculate goals; new plans cannot choose it. The same status again changes nothing.
func (s *Service) SetModelActive(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (ModelView, error) {
	active, version, err := checkStatus(update)
	if err != nil {
		return ModelView{}, err
	}
	var view ModelView
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		if err := m.CheckVersion(version, modelNoun); err != nil {
			return err
		}
		if m.Active != active {
			m.Active = active
			m.Touch(a, s.now())
			if err := s.repo.PutModel(ctx, m); err != nil {
				return err
			}
			if err := s.logStatus(ctx, a, modelKind, m.ID, m.Name, active); err != nil {
				return err
			}
		}
		view, err = s.modelView(ctx, m)
		return err
	})
	return view, err
}

// DeleteModel moves the model id to the recycle bin, so that it can be undone (FR-004, US1-7). A model in
// use — by any plan, also one in the recycle bin — is refused with in_use: set it inactive instead.
func (s *Service) DeleteModel(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		if usage := u.model(m.ID); usage.InUse {
			return inUseError(m.Name, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, modelKind, m.ID)
		return err
	})
	return entry, err
}
