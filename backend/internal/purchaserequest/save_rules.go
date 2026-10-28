package purchaserequest

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"mapp/internal/platform/authz"
)

var thenWords = map[string]string{ //nolint:gochecknoglobals // a fixed lookup of the buttons' words
	ThenSave: "Save", ThenResubmit: "Save and re-submit", ThenSubmitChanges: "Submit changes",
}

// checkSave is check plus the rules a save of before must follow by its status (data-model §3.2); addsOnly
// says the save only adds lines to an Approved purchase request without approval.
func (s *Service) checkSave(ctx context.Context, in Input, before PurchaseRequest, config Configuration) (checked, bool, error) {
	ck, err := s.check(ctx, in, before)
	if err != nil {
		return checked{}, false, err
	}
	addsOnly := before.Status == StatusApproved && config.LinesOnApproved == WithoutApproval && addsLinesOnly(before, ck.pr)
	if err = ck.then(ctx, before, in.Then, addsOnly); err != nil {
		return checked{}, false, err
	}
	ck.lineRules(before, config)
	if err = s.invoicedRule(ctx, &ck, before); err != nil {
		return checked{}, false, err
	}
	return ck, addsOnly && in.Then == ThenSave, ck.problems.Err()
}

// then checks the save's next step fits the status: Save, Save and re-submit or Submit changes.
func (ck *checked) then(ctx context.Context, before PurchaseRequest, then string, addsOnly bool) error {
	want := ThenSave
	switch before.Status {
	case StatusSubmitted, StatusInReview:
		want = ThenResubmit
	case StatusApproved:
		if !addsOnly || then != ThenSave {
			want = ThenSubmitChanges
		}
	case StatusCreated, StatusDeclined, StatusClosed:
	}
	if then != want {
		ck.problems.Add("then", fmt.Sprintf(msgChooseThen, thenWords[want]))
		return nil
	}
	if then == ThenSave {
		return nil
	}
	return authz.RequireIn(ctx, authz.PurchaseRequestsSubmit, ck.plan.Access)
}

// lineRules: lines leave only Created and Declined purchase requests; lines join Approved ones as the
// setting says.
func (ck *checked) lineRules(before PurchaseRequest, config Configuration) {
	removing := slices.ContainsFunc(before.Lines, func(l Line) bool { return ck.pr.line(l.ID) == nil })
	if removing && before.Status != StatusCreated && before.Status != StatusDeclined {
		ck.problems.Add(fieldLines, removeReason(before))
	}
	adding := slices.ContainsFunc(ck.pr.Lines, func(l Line) bool { return l.ID == "" })
	approved := before.Status == StatusApproved || before.ChangePending
	if adding && approved && config.LinesOnApproved == NotAllowed {
		ck.problems.Add(fieldLines, msgNoLinesOnApproved)
	}
}

func removeReason(pr PurchaseRequest) string {
	if pr.Status == StatusApproved || pr.ChangePending {
		return msgApprovedNoRemove
	}
	return msgRecallToRemove
}

// invoicedRule refuses a changed amount below what GSAP already invoiced on the line.
func (s *Service) invoicedRule(ctx context.Context, ck *checked, before PurchaseRequest) error {
	if before.Approved == nil {
		return nil
	}
	invoiced, err := s.deps.Invoices.InvoicedOn(ctx, poLinesOf([]PurchaseRequest{before}))
	if err != nil {
		return err
	}
	for i, l := range ck.pr.Lines {
		stored := before.line(l.ID)
		if stored == nil || stored.PO.Line == "" {
			continue
		}
		usd := invoiced[POLineRef{PONumber: stored.PO.Number, Line: stored.PO.Line}]
		inLine, err := ck.balance.rates.FromUSD(usd, l.Currency)
		if err != nil {
			return err
		}
		if l.Amount.Cmp(inLine.Amount) < 0 {
			ck.problems.Add(fmt.Sprintf("lines[%d].amount", i), fmt.Sprintf(msgBelowInvoiced, inLine.Amount.Format(), l.Currency))
		}
	}
	return nil
}

func addsLinesOnly(before, after PurchaseRequest) bool {
	hb, ha := headerOf(before), headerOf(after)
	sameAttrs := sameAttributes(hb.Attributes, ha.Attributes)
	hb.Attributes, ha.Attributes = nil, nil
	if !sameAttrs || !reflect.DeepEqual(hb, ha) || len(after.Lines) <= len(before.Lines) {
		return false
	}
	for _, l := range before.Lines {
		changed := after.line(l.ID)
		if changed == nil || !changed.sameAs(l) {
			return false
		}
	}
	return true
}
