package purchaserequest

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

const (
	msgOrderCreated = "%s created GSAP PR %s and PO %s (%s)."
	msgOrderTopUp   = "%s updated PO %s: %d changed, %d added (top-up)."
	verbGSAPOrder   = "ran GSAP for"
)

// CreateGSAPOrder asks GSAP (the simulated GSAP now) for the GSAP PR and the purchase order of an approved,
// routed purchase request waiting for GSAP, and stores the answer, all in one Write: the
// adapter books the PO lines in the same transaction, so nothing is kept when any part fails.
func (s *Service) CreateGSAPOrder(ctx context.Context, a actor.Actor, id string, version int) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		pr, err := s.orderable(ctx, id, version)
		if err != nil {
			return err
		}
		pending := pendingLines(pr)
		sent, err := s.routedPR(ctx, a, pr, pending)
		if err != nil {
			return err
		}
		order, err := s.gsap.CreatePurchaseOrder(ctx, sent)
		if err != nil {
			return err
		}
		details := orderText(order, pending, !pr.hasPO())
		if pr, err = applyOrder(pr, order); err != nil {
			return err
		}
		if pr, err = s.storeOrder(ctx, a, pr, details); err != nil {
			return err
		}
		view, err = s.viewOf(ctx, a, pr)
		return err
	})
	return view, err
}

func (s *Service) orderable(ctx context.Context, id string, version int) (PurchaseRequest, error) {
	pr, err := s.visible(ctx, id)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if err = authz.Require(ctx, authz.FinanceImport); err != nil {
		return PurchaseRequest{}, err
	}
	config, err := s.repo.Configuration(ctx)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if reason := s.orderReason(pr, config); reason != "" {
		return PurchaseRequest{}, errs.NotAllowed(reason)
	}
	return pr, pr.CheckVersion(version, recordNoun)
}

// orderReason is why GSAP cannot be asked for pr now, or "".
func (s *Service) orderReason(pr PurchaseRequest, config Configuration) string {
	switch {
	case !config.PurchaseSupport:
		return msgSwitchedOff
	case s.gsap == nil:
		return msgNotConnected
	case pr.ChangePending:
		return msgChangeInReview
	case pr.Status != StatusApproved || !slices.ContainsFunc(pr.Lines, func(l Line) bool { return l.SentOn != nil }):
		return msgNotRoutedApproved
	case len(pendingLines(pr)) == 0:
		return msgAlreadyOrdered
	}
	return ""
}

// gsapOrderAvailability is the action Create GSAP PR and PO: hidden without finance.import.
func (s *Service) gsapOrderAvailability(ctx context.Context, f accessFacts) authz.Availability {
	if !authz.Has(ctx, authz.FinanceImport) {
		return hidden
	}
	if reason := s.orderReason(f.pr, f.config); reason != "" {
		return authz.Unavailable(reason)
	}
	return authz.Allowed
}

func pendingLines(pr PurchaseRequest) []Line {
	var list []Line
	for _, l := range pr.Lines {
		if l.gsapPending() {
			list = append(list, l)
		}
	}
	return list
}

// routedPR is what GSAP receives: the lines to order with their keys, and the numbers GSAP already gave.
func (s *Service) routedPR(ctx context.Context, a actor.Actor, pr PurchaseRequest, lines []Line) (RoutedPR, error) {
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return RoutedPR{}, err
	}
	roles, err := s.deps.Attributes.Roles(ctx, pr.Attributes, []string{settings.RoleShortText, settings.RoleVendorCode})
	if err != nil {
		return RoutedPR{}, err
	}
	sent := RoutedPR{ID: pr.ID, Name: pr.Name, RequestedBy: a, ShortText: first(roles[settings.RoleShortText]),
		VendorName: pr.Vendor, VendorCode: first(roles[settings.RoleVendorCode]), GSAPPRNumber: pr.GSAP.PRNumber,
		PONumber: poNumberOf(pr), LastPOLine: lastPOLine(pr)}
	for _, l := range lines {
		f := facts[l.LineItemID]
		sent.Lines = append(sent.Lines, RoutedLine{ID: l.ID, LineItemID: l.LineItemID, HXKey: HXKey(l.LineItemID),
			SwbsCode: f.SwbsCode, CostCentre: f.CostCentre, GLCode: f.GLCode, Currency: l.Currency, Amount: l.Amount,
			DeliveryDate: l.DeliveryDate, POLine: l.PO.Line})
	}
	return sent, nil
}

func lastPOLine(pr PurchaseRequest) int {
	last := 0
	for _, l := range pr.Lines {
		if n, err := strconv.Atoi(l.PO.Line); err == nil {
			last = max(last, n)
		}
	}
	return last
}

func poNumberOf(pr PurchaseRequest) string {
	for _, l := range pr.Lines {
		if l.PO.Number != "" {
			return l.PO.Number
		}
	}
	return ""
}

// applyOrder stores GSAP's answer: its PR number and date once, and each answered line's PO fields.
func applyOrder(pr PurchaseRequest, order GSAPOrder) (PurchaseRequest, error) {
	if pr.GSAP.PRNumber == "" {
		prDate := order.GSAPPRDate
		pr.GSAP = GSAPHeader{PRNumber: order.GSAPPRNumber, PRDate: &prDate, Source: order.Source}
	}
	for _, answer := range order.Lines {
		l := pr.line(answer.PRLineID)
		if l == nil {
			return PurchaseRequest{}, fmt.Errorf("purchaserequest: GSAP answered line %s, which %s does not have", answer.PRLineID, pr.ID)
		}
		poDate := answer.Date
		l.PO = PurchaseOrderLine{Number: order.PONumber, Line: answer.POLine, Status: answer.Status, Date: &poDate,
			Value: &money.Money{Amount: answer.Value, Currency: l.Currency}}
	}
	return pr, nil
}

func (s *Service) storeOrder(ctx context.Context, a actor.Actor, pr PurchaseRequest, details string) (PurchaseRequest, error) {
	pr.Touch(a, s.now())
	if err := s.repo.Put(ctx, pr); err != nil {
		return PurchaseRequest{}, err
	}
	_, err := s.history.Log(ctx, a, history.LogInput{Kind: recordKind, RecordID: pr.ID, RecordName: pr.Name,
		Action: history.ActionChanged, Verb: verbGSAPOrder, Details: details})
	return pr, err
}

// orderText is the log's account of a run: created, or a top-up with lines changed and added.
func orderText(order GSAPOrder, sent []Line, firstRun bool) string {
	who := "GSAP"
	if order.Source == SourceSimulatedGSAP {
		who = "Simulated GSAP"
	}
	if firstRun {
		return fmt.Sprintf(msgOrderCreated, who, order.GSAPPRNumber, order.PONumber, lineCount(len(order.Lines)))
	}
	changed := 0
	for _, l := range sent {
		if l.PO.Line != "" {
			changed++
		}
	}
	return fmt.Sprintf(msgOrderTopUp, who, order.PONumber, changed, len(sent)-changed)
}

func lineCount(n int) string {
	if n == 1 {
		return "1 line"
	}
	return fmt.Sprintf("%d lines", n)
}
