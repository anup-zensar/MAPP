package workbook

import (
	"context"
	"errors"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

const keptReason = "It could not be deleted when the import was undone"

// Get returns an import with its records and what Undo import would do (FR-035).
func (s *Service) Get(ctx context.Context, _ actor.Actor, id string) (Detail, error) {
	var detail Detail
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		imp, err := s.found(ctx, id)
		if err != nil {
			return err
		}
		detail, err = s.detail(ctx, imp, nil)
		return err
	})
	return detail, err
}

// Undo moves every record the import created, with anything added under them since, to the recycle bin in
// one Write; a record its delete refuses is kept and named. Updates are not reverted (FR-036, D-340).
func (s *Service) Undo(ctx context.Context, a actor.Actor, id string) (Detail, error) {
	var detail Detail
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		imp, err := s.found(ctx, id)
		if err != nil {
			return err
		}
		if imp.UndoneAt != nil {
			return errs.AlreadyResolved(imp.FileName, errs.Resolution{How: StatusUndone, By: *imp.UndoneBy, At: *imp.UndoneAt})
		}
		if err = s.mayDelete(ctx, imp.PlanID); err != nil {
			return err
		}
		kept, err := s.deleteCreated(history.WithNote(ctx, fmt.Sprintf(noteUndone, id)), a, imp)
		if err != nil {
			return err
		}
		at := s.now()
		if err = s.repo.MarkUndone(ctx, id, a, at); err != nil {
			return err
		}
		imp.UndoneAt, imp.UndoneBy = &at, &a
		detail, err = s.detail(ctx, imp, kept)
		return err
	})
	return detail, err
}

func (s *Service) found(ctx context.Context, id string) (Import, error) {
	imp, found, err := s.repo.GetImport(ctx, authz.ScopeOf(ctx), id)
	if err == nil && !found {
		err = errs.NotFound(msgImportGone)
	}
	return imp, err
}

func (s *Service) mayDelete(ctx context.Context, planID string) error {
	facts, found, err := s.planning.PlanFacts(ctx, planID)
	if err != nil || !found {
		return errors.Join(err, errs.NotFound(msgPlanGone))
	}
	return authz.RequireIn(ctx, authz.ActivitiesDelete, facts)
}

// deleteCreated deletes the created records whose parent the import did not create too: the others go with it.
func (s *Service) deleteCreated(ctx context.Context, a actor.Actor, imp Import) ([]Kept, error) {
	created := map[string]bool{}
	for _, r := range imp.Records {
		created[r.RecordID] = r.Action == ActionCreated
	}
	kept := []Kept{}
	for _, r := range imp.Records {
		if r.Action != ActionCreated || created[r.ParentID] {
			continue
		}
		exists, err := s.planning.Exists(ctx, r.Level, r.RecordID)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		err = s.planning.Delete(ctx, a, r.Level, r.RecordID)
		apiErr, refused := errors.AsType[*errs.Error](err)
		switch {
		case refused:
			kept = append(kept, Kept{ID: r.RecordID, Level: r.Level, Name: r.Name, Reason: apiErr.Message})
		case err != nil:
			return nil, err
		}
	}
	return kept, nil
}

// detail describes an import: its records as they are now and, until it is undone, what Undo would remove.
func (s *Service) detail(ctx context.Context, imp Import, kept []Kept) (Detail, error) {
	d := Detail{Import: summaryOf(imp), Problems: imp.Problems, Records: []RecordRef{}, Kept: kept}
	if d.Problems == nil {
		d.Problems = []Problem{}
	}
	undo := UndoPreview{}
	for _, r := range imp.Records {
		exists, err := s.planning.Exists(ctx, r.Level, r.RecordID)
		if err != nil {
			return Detail{}, err
		}
		d.Records = append(d.Records, RecordRef{ID: r.RecordID, Level: r.Level, Name: r.Name, Action: r.Action, Exists: exists})
		switch {
		case r.Action != ActionCreated:
		case exists:
			undo.add(r.Level, 1)
		default:
			undo.AlreadyDeleted++
		}
	}
	switch {
	case imp.UndoneAt == nil:
		d.Undo, d.Kept = &undo, []Kept{}
	case d.Kept == nil:
		d.Kept = s.stillThere(d.Records, imp)
	}
	return d, nil
}

func (s *Service) stillThere(records []RecordRef, imp Import) []Kept {
	created := map[string]bool{}
	for _, r := range imp.Records {
		created[r.RecordID] = r.Action == ActionCreated
	}
	kept := []Kept{}
	for i, r := range records {
		if r.Exists && r.Action == ActionCreated && !created[imp.Records[i].ParentID] {
			kept = append(kept, Kept{ID: r.ID, Level: r.Level, Name: r.Name, Reason: keptReason})
		}
	}
	return kept
}
