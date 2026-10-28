package purchaserequest

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// Delete moves a Created or Declined purchase request with its lines to the recycle bin, which releases what
// they reserved.
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		pr, err := s.visible(ctx, id)
		if err != nil {
			return err
		}
		facts, err := s.factsOf(ctx, pr)
		if err != nil {
			return err
		}
		if err = authz.RequireIn(ctx, authz.PurchaseRequestsDelete, pr.planFacts(facts)); err != nil {
			return err
		}
		if _, err = s.switchedOn(ctx); err != nil {
			return err
		}
		if reason := deleteRefusal(pr.Status); reason != "" {
			return errs.NotAllowed(reason)
		}
		entry, err = s.history.MoveToBin(ctx, a, recordKind, id)
		return err
	})
	return entry, err
}

func deleteRefusal(status Status) string {
	switch status {
	case StatusApproved:
		return msgApprovedNoDelete
	case StatusSubmitted, StatusInReview:
		return msgAwaitingNoDelete
	case StatusClosed:
		return msgClosedNoChange
	case StatusCreated, StatusDeclined:
	}
	return ""
}

// afterRestore lets a purchase request come back only when every line still fits what its line item has left;
// a Restore brings it back Created, an Undo as it was.
func (s *Service) afterRestore(ctx context.Context, rc history.RestoreContext, top history.Snapshot) ([]string, error) {
	if _, err := s.switchedOn(ctx); err != nil {
		return nil, err
	}
	pr, err := s.stored(ctx, top.ID)
	if err != nil {
		return nil, err
	}
	if err = s.fits(ctx, pr, pr.Lines); err != nil {
		return nil, err
	}
	if rc.Undo || pr.Status == StatusCreated {
		return nil, nil
	}
	pr.Status = StatusCreated
	pr.Touch(rc.Actor, rc.Now)
	return []string{"Restored as Created."}, s.repo.Put(ctx, pr)
}

func (s *Service) fits(ctx context.Context, pr PurchaseRequest, lines []Line) error {
	facts, err := s.deps.LineItems.Facts(ctx, pr.lineItemIDs())
	if err != nil {
		return err
	}
	b, err := s.balanceOf(ctx, pr.lineItemIDs())
	if err != nil {
		return err
	}
	var problems []string
	for _, l := range lines {
		problem, err := lineFit(b, facts, pr.ID, l)
		if err != nil {
			return err
		}
		if problem != "" {
			problems = append(problems, problem)
		}
	}
	if len(problems) > 0 {
		return errs.CannotRestore(fmt.Sprintf(msgRestoreNoRoom, strings.Join(problems, "; ")))
	}
	return nil
}

func lineFit(b balance, facts map[string]LineItemFacts, prID string, l Line) (string, error) {
	f, found := facts[l.LineItemID]
	if !found {
		return fmt.Sprintf(msgRestoreGone, l.LineItemID), nil
	}
	problem, err := b.claimProblem(f, l.money(), prID)
	if err != nil || problem == "" {
		return "", err
	}
	left, err := b.available(f, prID)
	return fmt.Sprintf(msgRestoreLineItem, f.ID, max0(left).Format(), f.Currency), err
}

// prepareLineRestore lets a removed line come back to its Created or Declined purchase request when its line item
// is not on it again and still has room for it; it goes last.
func (s *Service) prepareLineRestore(ctx context.Context, rc history.RestoreContext, snap *history.Snapshot) ([]string, error) {
	l, err := decodeLine(snap.Data)
	if err != nil {
		return nil, err
	}
	pr, err := s.stored(ctx, l.PurchaseRequestID)
	if err != nil {
		return nil, err
	}
	if pr.Status != StatusCreated && pr.Status != StatusDeclined {
		return nil, errs.CannotRestore(msgRestoreLineStatus)
	}
	if pr.line(l.ID) != nil || slices.Contains(pr.lineItemIDs(), l.LineItemID) {
		return nil, errs.CannotRestore(fmt.Sprintf(msgOnceOnPR, l.LineItemID))
	}
	pr.Lines = append(pr.Lines, l)
	if err = s.fits(ctx, pr, []Line{l}); err != nil {
		return nil, err
	}
	l.Position = len(pr.Lines)
	if !rc.Undo {
		l.Touch(rc.Actor, rc.Now)
	}
	snap.Data, err = encode(l)
	return nil, err
}
