package purchaserequest

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// viewOf is the details page of pr as a sees it, inside the caller's transaction.
func (s *Service) viewOf(ctx context.Context, a actor.Actor, pr PurchaseRequest) (View, error) {
	r, err := s.readerFor(ctx, a, []PurchaseRequest{pr})
	if err != nil {
		return View{}, err
	}
	routings, err := s.repo.Routings(ctx, pr.ID)
	if err != nil {
		return View{}, err
	}
	problems, err := s.gates(ctx, pr, r.facts, submissionKind(pr))
	if err != nil {
		return View{}, err
	}
	missing, err := s.headerMissing(ctx, pr, r.facts)
	if err != nil {
		return View{}, err
	}
	v := r.header(pr)
	v.Readiness = Readiness{Ready: len(problems) == 0, Problems: problems}
	v.Missing, v.Routings = missing, routingViews(routings)
	v.Access = s.access(ctx, accessFacts{pr: pr, config: r.config, routed: len(routings) > 0, plan: pr.planFacts(r.facts)})
	if v.Lines, err = s.lineViews(ctx, pr, r); err != nil {
		return View{}, err
	}
	return v, nil
}

func (r *reader) header(pr PurchaseRequest) View {
	plan, path := r.plan(pr)
	v := View{
		ID: pr.ID, Name: pr.Name, Description: pr.Description, Owner: r.person(pr.OwnerID),
		MoAApprover: r.person(pr.MoAApproverID), ApprovalWorkflow: r.workflows[pr.WorkflowID],
		ModificationWorkflow: r.workflows[pr.ModificationWorkflowID], Status: pr.Status, ChangePending: pr.ChangePending,
		Plan: plan, Path: path, CostCentres: []string{}, HXKeys: []string{}, Attributes: nonNilValues(pr.Attributes),
		Vendor: pr.Vendor, AmountDisplay: r.shown(r.usd(pr.Lines)), InvoicedDisplay: r.shown(r.invoicedTotal(pr.Lines)),
		GSAPPRNumber: pr.GSAP.PRNumber, GSAPPRDate: pr.GSAP.PRDate, GSAPSource: pr.GSAP.Source,
		WaitingForGSAP: waiting(pr), Notice: notice(pr, r.config), Meta: pr.Meta,
	}
	for _, l := range pr.Lines {
		f := r.facts[l.LineItemID]
		if v.CompanyCode == "" {
			v.CompanyCode = f.CompanyCode
		}
		if f.CostCentre != "" && !slices.Contains(v.CostCentres, f.CostCentre) {
			v.CostCentres = append(v.CostCentres, f.CostCentre)
		}
		v.HXKeys = append(v.HXKeys, HXKey(l.LineItemID))
	}
	if pr.ChangePending {
		v.ApprovedVersion = r.version(pr.Approved)
	}
	return v
}

func (s *Service) lineViews(ctx context.Context, pr PurchaseRequest, r *reader) ([]LineView, error) {
	lines := make([]LineView, 0, len(pr.Lines))
	for _, l := range pr.Lines {
		missing, err := s.lineMissing(ctx, l, r.facts[l.LineItemID].BusinessUnit)
		if err != nil {
			return nil, err
		}
		lines = append(lines, LineView{
			ID: l.ID, Position: l.Position, LineItem: r.lineItemRef(l.LineItemID), HXKey: HXKey(l.LineItemID),
			Currency: l.Currency, Amount: l.money(), DeliveryDate: l.DeliveryDate, OverallLimit: inCurrency(l.OverallLimit, l.Currency),
			Description: l.Description, Attributes: nonNilValues(l.Attributes), SentOn: l.SentOn, PONumber: l.PO.Number,
			POLine: l.PO.Line, POLineStatus: l.PO.Status, PODate: l.PO.Date, POValue: l.PO.Value,
			Invoiced: r.lineInvoiced(l), Changed: pr.ChangePending && l.changedSince(pr.Approved), Missing: missing,
			Access: LineAccess{Remove: removeAvailability(ctx, pr, r.config, pr.planFacts(r.facts))}, GSAPPending: l.gsapPending(),
			Version: l.Version,
		})
	}
	return lines, nil
}

func (r *reader) version(v *Version) *VersionView {
	if v == nil {
		return nil
	}
	lines := make([]VersionLineView, 0, len(v.Lines))
	asLines := make([]Line, 0, len(v.Lines))
	for _, l := range v.Lines {
		lines = append(lines, VersionLineView{
			ID: l.ID, LineItemID: l.LineItemID, HXKey: HXKey(l.LineItemID),
			Amount: money.Money{Amount: l.Amount, Currency: l.Currency}, DeliveryDate: l.DeliveryDate,
			OverallLimit: inCurrency(l.OverallLimit, l.Currency), Description: l.Description,
		})
		asLines = append(asLines, Line{Amount: l.Amount, Currency: l.Currency})
	}
	return &VersionView{
		Name: v.Name, Description: v.Description, Owner: r.person(v.OwnerID), MoAApprover: r.person(v.MoAApproverID),
		Attributes: nonNilValues(v.Attributes), AmountDisplay: r.shown(r.usd(asLines)), Lines: lines, ApprovedAt: v.ApprovedAt,
	}
}

func routingViews(routings []Routing) []RoutingView {
	list := make([]RoutingView, 0, len(routings))
	for _, r := range routings {
		list = append(list, RoutingView{ID: r.ID, Type: r.Type, RoutedAt: r.RoutedAt, RoutedBy: r.RoutedBy,
			Recipients: r.Recipients, LineCount: len(r.Form)})
	}
	return list
}

func notice(pr PurchaseRequest, c Configuration) string {
	switch {
	case !c.PurchaseSupport:
		return msgSwitchedOff
	case pr.Status == StatusClosed:
		return msgClosedNoChange
	case pr.Status == StatusSubmitted || pr.Status == StatusInReview:
		return msgAwaitingNotice
	}
	return ""
}

func inCurrency(a *money.Amount, currency string) *money.Money {
	if a == nil {
		return nil
	}
	return &money.Money{Amount: *a, Currency: currency}
}

func nonNilValues(v settings.AttributeValues) settings.AttributeValues {
	if v == nil {
		return settings.AttributeValues{}
	}
	return v
}
