package approval

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// SubmitSubject submits or re-submits the record id of kind; the comment is optional, at most 2,000 characters.
func (s *Service) SubmitSubject(ctx context.Context, a actor.Actor, kind SubjectKind, id, comment string) (Section, error) { //nolint:revive // argument-limit 5: the interface agreed with feature 032 (spec 031 contracts/go-api.md)
	comment = strings.TrimSpace(comment)
	if tooLong(comment, maxCommentLength) {
		return Section{}, errs.Validation(errs.FieldError{Field: "comment", Message: tooLong2000Message})
	}
	var sec Section
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		sub, err := s.subjectOf(ctx, kind, id)
		if err != nil {
			return err
		}
		updated, err := s.submit(ctx, a, sub, comment)
		if err != nil {
			return err
		}
		sec, err = s.section(ctx, a, updated)
		return err
	})
	if err != nil {
		return Section{}, err
	}
	return sec, nil
}

func (s *Service) RecallSubject(ctx context.Context, a actor.Actor, kind SubjectKind, id string) (Section, error) {
	var sec Section
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		sub, err := s.subjectOf(ctx, kind, id)
		if err != nil {
			return err
		}
		rounds, err := s.repo.RoundsOf(ctx, sub.ID)
		if err != nil {
			return err
		}
		open, found := openRound(rounds)
		switch {
		case found:
		case sub.Kind == KindActivity && sub.Status == StatusApproved:
			return errs.BadRequest(approvedNoRecallMessage)
		default:
			return errs.BadRequest(fmt.Sprintf(nothingToRecallMessage, sub.Kind.word()))
		}
		updated, err := s.recall(ctx, a, sub, recalling{roundID: open.ID})
		if err != nil {
			return err
		}
		sec, err = s.section(ctx, a, updated)
		return err
	})
	if err != nil {
		return Section{}, err
	}
	return sec, nil
}

// DecideSubject approves or declines the current level of the record's open round, as the decision names it.
func (s *Service) DecideSubject(ctx context.Context, a actor.Actor, kind SubjectKind, id string, decision SubjectDecision) (Section, error) { //nolint:revive // argument-limit 5: the interface agreed with feature 032 (spec 031 contracts/go-api.md)
	comment := strings.TrimSpace(decision.Comment)
	if err := decisionProblems(decision, comment, kind); err != nil {
		return Section{}, err
	}
	var sec Section
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		sub, err := s.subjectOf(ctx, kind, id)
		if err != nil {
			return err
		}
		if err = s.mayDecide(ctx, a, sub); err != nil {
			return err
		}
		round, err := s.roundToDecide(ctx, sub, decision.DecisionInput)
		if err != nil {
			return err
		}
		dg, err := s.decidingOn(ctx, a, sub.as(round.kind()), round)
		if err != nil {
			return err
		}
		dg.kind, dg.comment, dg.now = decision.Action, comment, s.now()
		if err = s.refuseOverage(ctx, dg); err != nil {
			return err
		}
		updated, err := s.recordDecision(ctx, dg)
		if err != nil {
			return err
		}
		sec, err = s.section(ctx, a, updated)
		return err
	})
	if err != nil {
		return Section{}, err
	}
	return sec, nil
}

// SectionOf returns the record's approval section: every round of the record, newest first, and the actions
// of kind available to a now (data-model §9).
func (s *Service) SectionOf(ctx context.Context, a actor.Actor, kind SubjectKind, id string) (Section, error) {
	var sec Section
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		sub, err := s.subjectOf(ctx, kind, id)
		if err != nil {
			return err
		}
		sec, err = s.section(ctx, a, sub)
		return err
	})
	if err != nil {
		return Section{}, err
	}
	return sec, nil
}

// SubmissionOf is the record's submission check as for an activity: the approvers of each level resolved as if a
// submitted now, those named by a field of the record included.
func (s *Service) SubmissionOf(ctx context.Context, a actor.Actor, kind SubjectKind, id string) (Submission, error) {
	var submission Submission
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		sub, err := s.subjectOf(ctx, kind, id)
		if err != nil {
			return err
		}
		submission, err = s.submission(ctx, a, sub)
		return err
	})
	return submission, err
}
