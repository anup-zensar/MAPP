package purchaserequest

import (
	"context"
	"slices"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// reader is what showing purchase requests needs, read once per request for all of them (B5: no read per row).
type reader struct {
	config    Configuration
	rates     money.Rates
	display   money.Display
	facts     map[string]LineItemFacts
	people    map[string]Person
	invoiced  map[POLineRef]money.Amount
	workflows map[string]*WorkflowRef
}

func (s *Service) readerFor(ctx context.Context, a actor.Actor, prs []PurchaseRequest) (*reader, error) {
	r := &reader{workflows: map[string]*WorkflowRef{}}
	var err error
	if r.config, err = s.repo.Configuration(ctx); err != nil {
		return nil, err
	}
	if r.rates, err = s.deps.Currencies.Rates(ctx); err != nil {
		return nil, err
	}
	shown, err := s.deps.People.DisplayRateOf(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	r.display = money.DisplayIn(shown.Code, shown.Rate).WithRates(r.rates)
	if r.facts, err = s.deps.LineItems.Facts(ctx, lineItemsOf(prs)); err != nil {
		return nil, err
	}
	if r.people, err = s.deps.People.People(ctx, peopleOf(prs, r.facts)); err != nil {
		return nil, err
	}
	if r.invoiced, err = s.deps.Invoices.InvoicedOn(ctx, poLinesOf(prs)); err != nil {
		return nil, err
	}
	return r, s.readWorkflows(ctx, r, prs)
}

func (s *Service) readWorkflows(ctx context.Context, r *reader, prs []PurchaseRequest) error {
	for _, pr := range prs {
		for _, id := range []string{pr.WorkflowID, pr.ModificationWorkflowID} {
			if _, read := r.workflows[id]; read {
				continue
			}
			w, found, err := s.deps.Workflows.Workflow(ctx, id)
			if err != nil {
				return err
			}
			r.workflows[id] = nil
			if found {
				r.workflows[id] = &WorkflowRef{ID: w.ID, Name: w.Name, Active: w.Active}
			}
		}
	}
	return nil
}

func lineItemsOf(prs []PurchaseRequest) []string {
	var list []string
	for _, pr := range prs {
		list = append(list, pr.lineItemIDs()...)
	}
	return list
}

func peopleOf(prs []PurchaseRequest, facts map[string]LineItemFacts) []string {
	var list []string
	for _, pr := range prs {
		list = append(list, pr.OwnerID, pr.MoAApproverID)
		if pr.Approved != nil {
			list = append(list, pr.Approved.OwnerID, pr.Approved.MoAApproverID)
		}
	}
	for _, f := range facts {
		list = append(list, f.OwnerID)
	}
	return list
}

func poLinesOf(prs []PurchaseRequest) []POLineRef {
	var list []POLineRef
	for _, pr := range prs {
		for _, l := range pr.Lines {
			if l.PO.Line != "" {
				list = append(list, POLineRef{PONumber: l.PO.Number, Line: l.PO.Line})
			}
		}
	}
	return list
}

func (r *reader) person(id string) *people.PersonRef {
	p, found := r.people[id]
	if id == "" || !found {
		return nil
	}
	ref := p.PersonRef
	return &ref
}

// usd is the purchase request's lines in USD at today's rates; a currency without a rate counts at 1.
func (r *reader) usd(lines []Line) money.Amount {
	var total money.Amount
	for _, l := range lines {
		converted, err := r.rates.Convert(l.money(), money.BaseCurrency)
		if err != nil {
			converted = money.USD(l.Amount)
		}
		total = total.Add(converted.Amount)
	}
	return total
}

func (r *reader) shown(usd money.Amount) money.Money {
	return r.display.Money(money.USD(usd))
}

// invoicedUSD is the line's invoices minus credit notes in USD; 0.00 without a PO line.
func (r *reader) invoicedUSD(l Line) money.Amount {
	if l.PO.Line == "" {
		return money.Amount{}
	}
	return r.invoiced[POLineRef{PONumber: l.PO.Number, Line: l.PO.Line}]
}

// lineInvoiced is the line's invoiced amount in its own currency.
func (r *reader) lineInvoiced(l Line) money.Money {
	m, err := r.rates.FromUSD(r.invoicedUSD(l), l.Currency)
	if err != nil {
		return money.Money{Amount: r.invoicedUSD(l), Currency: l.Currency}
	}
	return m
}

func (r *reader) invoicedTotal(lines []Line) money.Amount {
	var total money.Amount
	for _, l := range lines {
		total = total.Add(r.invoicedUSD(l))
	}
	return total
}

func (r *reader) lineItemRef(id string) LineItemRef {
	f, found := r.facts[id]
	if !found {
		return LineItemRef{ID: id, PlannedCost: money.USD(money.Amount{}), Currency: money.BaseCurrency}
	}
	return LineItemRef{
		ID: f.ID, Name: f.Name, ActivityID: f.ActivityID, ActivityName: f.ActivityName, Currency: f.Currency,
		PlannedCost: money.Money{Amount: f.PlannedCost, Currency: f.Currency}, SwbsCode: f.SwbsCode,
		SwbsStatus: f.SwbsStatus, CompanyCode: f.CompanyCode, CostCentre: f.CostCentre, GLCode: f.GLCode,
	}
}

// plan is the plan of the purchase request's first line item, as a reference.
func (r *reader) plan(pr PurchaseRequest) (errs.RecordRef, []errs.RecordRef) {
	for _, l := range pr.Lines {
		if f, found := r.facts[l.LineItemID]; found {
			return errs.RecordRef{Type: planType, ID: f.Plan.Access.ID, Name: f.Plan.Name}, slices.Clone(f.Path)
		}
	}
	return errs.RecordRef{Type: planType, ID: pr.PlanID, Name: pr.PlanID}, []errs.RecordRef{}
}

// waiting is 032 FR-032: approved, routed (its lines got their Sent date) and a line GSAP has not answered.
func waiting(pr PurchaseRequest) bool {
	if pr.Status != StatusApproved || !slices.ContainsFunc(pr.Lines, func(l Line) bool { return l.SentOn != nil }) {
		return false
	}
	return slices.ContainsFunc(pr.Lines, Line.gsapPending)
}
