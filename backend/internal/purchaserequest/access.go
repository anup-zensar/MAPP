package purchaserequest

import (
	"context"

	"mapp/internal/platform/authz"
)

const msgRecallToRemove = "Lines can be removed while the purchase request is Created or Declined — recall it first."

// accessFacts are what the actions of a purchase request depend on.
type accessFacts struct {
	pr     PurchaseRequest
	config Configuration
	routed bool
	plan   authz.PlanFacts
}

var hidden = authz.Availability{} //nolint:gochecknoglobals // the zero availability: not offered, nothing to explain

// access says what the person acting may do with the purchase request now, and why not.
func (s *Service) access(ctx context.Context, f accessFacts) AccessView {
	if !f.config.PurchaseSupport {
		off := authz.Unavailable(msgSwitchedOff)
		return AccessView{Edit: off, Delete: off, Submit: off, Resubmit: off, SubmitChanges: off, Recall: off, Close: off,
			Resend: f.resend(ctx), AddLine: off, GSAPOrder: s.gsapOrderAvailability(ctx, f)}
	}
	return AccessView{
		Edit: f.edit(ctx), Delete: f.delete(ctx), Submit: f.submit(ctx), Resubmit: f.resubmit(ctx),
		SubmitChanges: f.submitChanges(ctx), Recall: f.recall(ctx), Close: f.close(ctx), Resend: f.resend(ctx),
		AddLine: f.addLine(ctx), GSAPOrder: s.gsapOrderAvailability(ctx, f),
	}
}

func (f accessFacts) edit(ctx context.Context) authz.Availability {
	if f.pr.Status == StatusClosed {
		return authz.Unavailable(msgClosedNoChange)
	}
	return authz.Available(ctx, authz.PurchaseRequestsEdit, f.plan)
}

// delete gives the permission reason before any status guidance (038 FR-025).
func (f accessFacts) delete(ctx context.Context) authz.Availability {
	if permission := authz.Available(ctx, authz.PurchaseRequestsDelete, f.plan); !permission.Available {
		return permission
	}
	switch f.pr.Status {
	case StatusApproved:
		return authz.Unavailable(msgApprovedNoDelete)
	case StatusSubmitted, StatusInReview:
		return authz.Unavailable(msgAwaitingNoDelete)
	case StatusClosed:
		return authz.Unavailable(msgClosedNoChange)
	case StatusCreated, StatusDeclined:
	}
	return authz.Allowed
}

func (f accessFacts) submit(ctx context.Context) authz.Availability {
	switch f.pr.Status {
	case StatusCreated:
		return authz.Available(ctx, authz.PurchaseRequestsSubmit, f.plan)
	case StatusApproved:
		if changes := f.submitChanges(ctx); !changes.Available {
			return changes
		}
		return authz.Unavailable(msgEditToChange)
	case StatusSubmitted, StatusInReview, StatusDeclined, StatusClosed:
	}
	return hidden
}

func (f accessFacts) resubmit(ctx context.Context) authz.Availability {
	switch f.pr.Status {
	case StatusSubmitted, StatusInReview, StatusDeclined:
		return authz.Available(ctx, authz.PurchaseRequestsSubmit, f.plan)
	case StatusCreated, StatusApproved, StatusClosed:
	}
	return hidden
}

func (f accessFacts) submitChanges(ctx context.Context) authz.Availability {
	if f.pr.Status != StatusApproved {
		return hidden
	}
	if edit := authz.Available(ctx, authz.PurchaseRequestsEdit, f.plan); !edit.Available {
		return edit
	}
	return authz.Available(ctx, authz.PurchaseRequestsSubmit, f.plan)
}

// recall is offered while a round is open; the approval engine decides whether this person may.
func (f accessFacts) recall(ctx context.Context) authz.Availability {
	if f.pr.Status != StatusSubmitted && f.pr.Status != StatusInReview {
		return hidden
	}
	return authz.Available(ctx, authz.PurchaseRequestsSubmit, f.plan)
}

func (f accessFacts) close(ctx context.Context) authz.Availability {
	switch {
	case f.pr.Status == StatusInReview && f.pr.ChangePending:
		return authz.Unavailable(msgChangeInReview)
	case f.pr.Status != StatusApproved:
		return hidden
	}
	return authz.Available(ctx, authz.PurchaseRequestsEdit, f.plan)
}

func (f accessFacts) resend(ctx context.Context) authz.Availability {
	switch {
	case !f.routed:
		return authz.Unavailable(msgNoFormYet)
	case f.pr.Status != StatusApproved && f.pr.Status != StatusClosed:
		return hidden
	}
	return authz.Available(ctx, authz.PurchaseRequestsEdit, f.plan)
}

func (f accessFacts) addLine(ctx context.Context) authz.Availability {
	switch {
	case f.pr.Status == StatusClosed:
		return authz.Unavailable(msgClosedNoChange)
	case f.pr.Status == StatusApproved && f.config.LinesOnApproved == NotAllowed:
		return authz.Unavailable(msgNoLinesOnApproved)
	}
	return authz.Available(ctx, authz.PurchaseRequestsEdit, f.plan)
}

// removeAvailability is Remove line: Created and Declined purchase requests keep at least one line.
func removeAvailability(ctx context.Context, pr PurchaseRequest, c Configuration, plan authz.PlanFacts) authz.Availability {
	switch {
	case !c.PurchaseSupport:
		return authz.Unavailable(msgSwitchedOff)
	case pr.Status == StatusApproved || pr.Status == StatusClosed || pr.ChangePending:
		return authz.Unavailable(msgApprovedNoRemove)
	case pr.Status == StatusSubmitted || pr.Status == StatusInReview:
		return authz.Unavailable(msgRecallToRemove)
	case len(pr.Lines) < 2:
		return authz.Unavailable(msgLastLine)
	}
	return authz.Available(ctx, authz.PurchaseRequestsEdit, plan)
}
