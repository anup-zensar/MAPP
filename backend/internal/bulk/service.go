package bulk

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

// errDiscard ends the Write of a preview, so that nothing the run did is saved (D-126).
var errDiscard = errors.New("bulk: preview only, discard the changes")

type Service struct {
	tx       Transactor
	repo     Store
	ids      *ids.Generator
	history  *history.Service
	now      clock.Clock
	planning Planning
}

func NewService(tools history.Tools, repo Store, planning Planning) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now, planning: planning}
}

// Preview answers what Run would do with req — every outcome, re-approval, effect and the delete summary —
// and saves nothing.
func (s *Service) Preview(ctx context.Context, a actor.Actor, req Request) (Result, error) {
	return s.run(ctx, a, req, false)
}

// Run changes each selected record through its single operation in one Write: done, skipped with the
// operation's reasons, or failed when it no longer exists (FR-021, FR-022).
func (s *Service) Run(ctx context.Context, a actor.Actor, req Request) (Result, error) {
	return s.run(ctx, a, req, true)
}

func (s *Service) run(ctx context.Context, a actor.Actor, req Request, commit bool) (Result, error) {
	recordIDs := uniqueIDs(req.RecordIDs)
	if err := checkRequest(req, recordIDs); err != nil {
		return Result{}, err
	}
	selected := countByLevel(recordIDs)
	field, err := s.checkChoices(ctx, req, selected)
	if err != nil {
		return Result{}, err
	}
	r := &runner{s: s, a: a, req: req, field: field, result: newResult(req, selected, !commit)}
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		if commit {
			id, idErr := s.ids.Next(ctx, ids.PrefixBulkAction)
			if idErr != nil {
				return idErr
			}
			r.result.ID, r.result.Note = &id, noteOf(req.Action, selected, id)
			ctx = history.WithNote(ctx, r.result.Note)
		}
		if runErr := r.process(ctx, recordIDs); runErr != nil {
			return runErr
		}
		if !commit {
			return errDiscard
		}
		return s.repo.AddAction(ctx, r.stored())
	})
	if err != nil && !errors.Is(err, errDiscard) {
		return Result{}, err
	}
	return r.result, nil
}

// checkChoices checks the chosen person or field against the lists they come from.
func (s *Service) checkChoices(ctx context.Context, req Request, selected LevelCounts) (Field, error) {
	switch req.Action {
	case ActionChangeOwner:
		name, active, found, err := s.planning.Person(ctx, strings.TrimSpace(req.OwnerID))
		switch {
		case err != nil:
			return Field{}, err
		case !found:
			return Field{}, errs.Validation(errs.FieldError{Field: fieldOwnerID, Message: msgChoosePerson})
		case !active:
			return Field{}, errs.Validation(errs.FieldError{Field: fieldOwnerID, Message: fmt.Sprintf(msgInactivePerson, name)})
		}
	case ActionChangeField:
		level := levelOfCounts(selected)
		offered, err := s.planning.Fields(ctx, level)
		if err != nil {
			return Field{}, err
		}
		return fieldOf(offered, *req.Field, level)
	case ActionSubmit, ActionCopy, ActionMove, ActionDelete:
	}
	return Field{}, nil
}

func levelOfCounts(c LevelCounts) Level {
	switch {
	case c.Campaigns > 0:
		return LevelCampaign
	case c.Initiatives > 0:
		return LevelInitiative
	}
	return LevelActivity
}

func newResult(req Request, selected LevelCounts, preview bool) Result {
	return Result{
		Preview: preview, Action: req.Action, Verb: verbs[req.Action], Note: noteOf(req.Action, selected, ""),
		Selected: selected, Outcomes: []Outcome{}, Reapproval: []RecordRef{}, Effects: []Effect{},
		RecycleBinEntries: []history.BinEntryView{},
	}
}
