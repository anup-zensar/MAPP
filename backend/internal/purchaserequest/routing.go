package purchaserequest

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/settings"
)

var routingWords = map[RoutingType]string{ //nolint:gochecknoglobals // a fixed lookup of the routing types' words
	RoutingOriginal: "Original", RoutingModification: "Modification", RoutingResend: "Resend", RoutingClose: "Close",
}

// routing is one routing to make: its type and, for a modification, the approved version it changes.
type routing struct {
	typ      RoutingType
	previous *Version
}

// route hands pr to the requisitioner inside MAPP: a routing with its recipients — the Email
// group and PR recipients values, else the requisitioner queue — and the PR form rows as routed; the lines get
// their Sent date at their first routing. The caller stores pr.
func (s *Service) route(ctx context.Context, a actor.Actor, pr *PurchaseRequest, rt routing) error {
	rows, recipients, err := s.formOf(ctx, *pr, rt)
	if err != nil {
		return err
	}
	id, err := s.ids.Next(ctx, ids.PrefixPurchaseRequestRouting)
	if err != nil {
		return err
	}
	r := Routing{ID: id, PurchaseRequestID: pr.ID, Type: rt.typ, RoutedAt: s.now(), RoutedBy: a, Recipients: recipients, Form: rows}
	if err = s.repo.AddRouting(ctx, r); err != nil {
		return err
	}
	if rt.typ == RoutingOriginal || rt.typ == RoutingModification {
		today := date.Today(s.now())
		for i := range pr.Lines {
			if pr.Lines[i].SentOn == nil {
				pr.Lines[i].SentOn = &today
			}
		}
	}
	_, err = s.history.Log(ctx, a, history.LogInput{Kind: recordKind, RecordID: pr.ID, RecordName: pr.Name,
		Action: history.ActionChanged, Verb: "routed",
		Details: fmt.Sprintf("%s · routed to %s", routingWords[rt.typ], strings.Join(recipients, ", "))})
	return err
}

// formOf is the PR form of pr as routed now (spec "Requisitioner Attachment") and its recipients.
func (s *Service) formOf(ctx context.Context, pr PurchaseRequest, rt routing) ([]FormRow, []string, error) {
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return nil, nil, err
	}
	roles, err := s.deps.Attributes.Roles(ctx, pr.Attributes, []string{settings.RoleEmailGroup, settings.RolePRRecipients,
		settings.RoleShortText, settings.RoleVendor, settings.RoleVendorCode})
	if err != nil {
		return nil, nil, err
	}
	found, err := s.deps.People.People(ctx, []string{pr.MoAApproverID})
	if err != nil {
		return nil, nil, err
	}
	moa := found[pr.MoAApproverID]
	rows := make([]FormRow, 0, len(pr.Lines))
	for _, l := range pr.Lines {
		f := facts[l.LineItemID]
		rows = append(rows, FormRow{
			PurchaseRequestID: pr.ID, PurchaseRequestName: pr.Name, LineID: l.ID, HXKey: HXKey(l.LineItemID),
			ShortText: first(roles[settings.RoleShortText]), Vendor: pr.Vendor, VendorCode: first(roles[settings.RoleVendorCode]),
			CostCentre: f.CostCentre, SwbsCode: f.SwbsCode, GLCode: f.GLCode, Amount: l.Amount, Currency: l.Currency,
			DeliveryDate: l.DeliveryDate, MoAApproverName: moa.Name, MoAApproverEmail: moa.Email,
			Changed: rt.typ == RoutingModification && l.changedSince(rt.previous),
		})
	}
	recipients := append(append([]string{}, roles[settings.RoleEmailGroup]...), roles[settings.RolePRRecipients]...)
	if len(recipients) == 0 {
		recipients = []string{msgRequisitionerQueue}
	}
	return rows, recipients, nil
}

// Resend routes the PR form again with today's values; Sent dates do not change.
func (s *Service) Resend(ctx context.Context, a actor.Actor, id string, in VersionInput) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		pr, err := s.resendable(ctx, id, in.Version)
		if err != nil {
			return err
		}
		if err = s.route(ctx, a, &pr, routing{typ: RoutingResend}); err != nil {
			return err
		}
		view, err = s.viewOf(ctx, a, pr)
		return err
	})
	return view, err
}

func (s *Service) resendable(ctx context.Context, id string, version int) (PurchaseRequest, error) {
	pr, err := s.visible(ctx, id)
	if err != nil {
		return PurchaseRequest{}, err
	}
	routings, err := s.repo.Routings(ctx, id)
	switch {
	case err != nil:
		return PurchaseRequest{}, err
	case len(routings) == 0:
		return PurchaseRequest{}, errs.NotAllowed(msgNoFormYet)
	case pr.Status != StatusApproved && pr.Status != StatusClosed:
		return PurchaseRequest{}, errs.NotAllowed(msgChangeInReview)
	}
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if err = authz.RequireIn(ctx, authz.PurchaseRequestsEdit, pr.planFacts(facts)); err != nil {
		return PurchaseRequest{}, err
	}
	return pr, pr.CheckVersion(version, recordNoun)
}
