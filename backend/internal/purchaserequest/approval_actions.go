package purchaserequest

import (
	"context"
	"strings"
	"unicode/utf8"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

func (s *Service) Approval(ctx context.Context, a actor.Actor, id string) (ApprovalView, error) {
	var view ApprovalView
	err := s.withEngine(ctx, s.tx.Read, id, func(ctx context.Context, pr PurchaseRequest) error {
		var err error
		view, err = s.approvalOf(ctx, a, pr)
		return err
	})
	return view, err
}

func (s *Service) Submission(ctx context.Context, a actor.Actor, id string) (SubmissionView, error) {
	var view SubmissionView
	err := s.withEngine(ctx, s.tx.Read, id, func(ctx context.Context, pr PurchaseRequest) error {
		facts, err := s.factsOf(ctx, pr)
		if err != nil {
			return err
		}
		if err = authz.RequireIn(ctx, authz.PurchaseRequestsSubmit, pr.planFacts(facts)); err != nil {
			return err
		}
		kind := submissionKind(pr)
		problems, err := s.gates(ctx, pr, facts, kind)
		if err != nil {
			return err
		}
		check, err := s.engine.SubmissionOf(ctx, a, approval.SubjectKind(kind), pr.ID)
		view = SubmissionView{PurchaseRequestID: pr.ID, Version: pr.Version, Kind: "submit", RoundKind: kind,
			Ready: len(problems) == 0, Problems: problems, Workflow: check.Workflow, Levels: check.Levels,
			Outcomes: check.Outcomes}
		view.Available, view.Reason = submitAllowed(pr)
		if pr.Status != StatusCreated && pr.Status != StatusApproved {
			view.Kind = "resubmit"
		}
		return err
	})
	return view, err
}

// Submit submits a Created purchase request, or re-submits a Submitted, In review or Declined one without
// changes, once the gates pass; an Approved one is changed with Submit changes.
func (s *Service) Submit(ctx context.Context, a actor.Actor, id string, in SubmitInput) (ApprovalView, error) {
	comment := strings.TrimSpace(in.Comment)
	if utf8.RuneCountInString(comment) > maxTextLength {
		return ApprovalView{}, errs.Validation(errs.FieldError{Field: "comment", Message: msgTooLong2000})
	}
	var view ApprovalView
	err := s.withEngine(ctx, s.tx.Write, id, func(ctx context.Context, pr PurchaseRequest) error {
		facts, err := s.submittable(ctx, pr, in.Version)
		if err != nil {
			return err
		}
		if _, err = s.submitAfterSave(ctx, a, toSubmit{pr: pr, facts: facts, comment: comment}); err != nil {
			return err
		}
		view, err = s.approvedView(ctx, a, pr.ID)
		return err
	})
	return view, err
}

func (s *Service) submittable(ctx context.Context, pr PurchaseRequest, version int) (map[string]LineItemFacts, error) {
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return nil, err
	}
	if err = authz.RequireIn(ctx, authz.PurchaseRequestsSubmit, pr.planFacts(facts)); err != nil {
		return nil, err
	}
	if _, err = s.switchedOn(ctx); err != nil {
		return nil, err
	}
	if available, reason := submitAllowed(pr); !available {
		return nil, errs.NotAllowed(reason)
	}
	return facts, pr.CheckVersion(version, recordNoun)
}

func submitAllowed(pr PurchaseRequest) (bool, string) {
	switch pr.Status {
	case StatusApproved:
		return false, msgEditToChange
	case StatusClosed:
		return false, msgClosedNoChange
	case StatusCreated, StatusSubmitted, StatusInReview, StatusDeclined:
	}
	return true, ""
}

// Decide approves or declines the waiting level of the open round as one of its approvers; the engine's rules
// decide who may, owners and MoA approvers included.
func (s *Service) Decide(ctx context.Context, a actor.Actor, id string, d approval.SubjectDecision) (ApprovalView, error) {
	var view ApprovalView
	err := s.withEngine(ctx, s.tx.Write, id, func(ctx context.Context, pr PurchaseRequest) error {
		if _, err := s.switchedOn(ctx); err != nil {
			return err
		}
		if _, err := s.engine.DecideSubject(ctx, a, approval.SubjectKind(openKind(pr)), pr.ID, d); err != nil {
			return err
		}
		var err error
		view, err = s.approvedView(ctx, a, pr.ID)
		return err
	})
	return view, err
}

// Recall takes the open round back as its owner or submitter: a first round returns the purchase request
// to Created, a change to its approved values.
func (s *Service) Recall(ctx context.Context, a actor.Actor, id string) (ApprovalView, error) {
	var view ApprovalView
	err := s.withEngine(ctx, s.tx.Write, id, func(ctx context.Context, pr PurchaseRequest) error {
		if _, err := s.switchedOn(ctx); err != nil {
			return err
		}
		if _, err := s.engine.RecallSubject(ctx, a, approval.SubjectKind(openKind(pr)), pr.ID); err != nil {
			return err
		}
		var err error
		view, err = s.approvedView(ctx, a, pr.ID)
		return err
	})
	return view, err
}

func (s *Service) Preview(ctx context.Context, a actor.Actor, id string) (PreviewView, error) {
	var view PreviewView
	err := s.withEngine(ctx, s.tx.Read, id, func(ctx context.Context, pr PurchaseRequest) error {
		r, err := s.readerFor(ctx, a, []PurchaseRequest{pr})
		if err != nil {
			return err
		}
		view.PurchaseRequest = r.row(pr)
		view.Approval, err = s.approvalOf(ctx, a, pr)
		return err
	})
	return view, err
}

func (s *Service) approvedView(ctx context.Context, a actor.Actor, id string) (ApprovalView, error) {
	pr, err := s.stored(ctx, id)
	if err != nil {
		return ApprovalView{}, err
	}
	return s.approvalOf(ctx, a, pr)
}

// withEngine runs fn on the purchase request the person sees, inside run (a Read or a Write); without the
// approval engine the approval operations answer not found.
func (s *Service) withEngine(ctx context.Context, run func(context.Context, func(context.Context) error) error, id string,
	fn func(ctx context.Context, pr PurchaseRequest) error) error {
	if s.engine == nil {
		return errs.NotFound(msgNotFound)
	}
	return run(ctx, func(ctx context.Context) error {
		pr, err := s.visible(ctx, id)
		if err != nil {
			return err
		}
		return fn(ctx, pr)
	})
}
