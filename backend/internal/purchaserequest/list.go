package purchaserequest

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// The sort orders and page sizes of the Purchase requests page (contract listPurchaseRequests).
var (
	sorts     = []string{"name", "-name", "id", "-id", "status", "-status", "amount", "-amount", "created", "-created"} //nolint:gochecknoglobals // allowlist
	pageSizes = []int{50, 100, 200}                                                                                     //nolint:gochecknoglobals // allowlist
)

// List is the Purchase requests page: the purchase requests of the plans where the person
// holds purchaseRequests.list, grouped by vendor with each vendor's count and totals over every match.
func (s *Service) List(ctx context.Context, a actor.Actor, q ListQuery) (ListPage, error) {
	if err := checkListQuery(q); err != nil {
		return ListPage{}, err
	}
	var page ListPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if _, err := s.switchedOn(ctx); err != nil {
			return err
		}
		found, err := s.page(ctx, q)
		if err != nil {
			return err
		}
		r, err := s.readerFor(ctx, a, found.Items)
		if err != nil {
			return err
		}
		page = ListPage{Items: make([]Row, 0, len(found.Items)), Total: found.Total, Page: q.Page, PageSize: q.PageSize,
			Vendors: make([]VendorGroupView, 0, len(found.Vendors)), PurchaseSupport: true}
		for _, pr := range found.Items {
			page.Items = append(page.Items, r.row(pr))
		}
		page.Vendors, err = s.vendorViews(ctx, r, found.Vendors)
		return err
	})
	return page, err
}

func (s *Service) page(ctx context.Context, q ListQuery) (Page, error) {
	rates, err := s.deps.Currencies.Rates(ctx)
	if err != nil {
		return Page{}, err
	}
	typeAttribute, err := s.deps.Attributes.AttributeID(ctx, settings.RolePRType)
	if err != nil {
		return Page{}, err
	}
	q.Rates, q.TypeAttributeID, q.Search = rates, typeAttribute, strings.ToLower(strings.TrimSpace(q.Search))
	vendors := make([]string, 0, len(q.Vendors))
	for _, v := range q.Vendors {
		vendors = append(vendors, strings.ToLower(strings.TrimSpace(v)))
	}
	q.Vendors = vendors
	return s.repo.Page(ctx, authz.PermissionScopeOf(ctx, authz.PurchaseRequestsList), q)
}

func (s *Service) vendorViews(ctx context.Context, r *reader, groups []VendorGroup) ([]VendorGroupView, error) {
	var refs []POLineRef
	for _, g := range groups {
		refs = append(refs, g.POLines...)
	}
	invoiced, err := s.deps.Invoices.InvoicedOn(ctx, refs)
	if err != nil {
		return nil, err
	}
	list := make([]VendorGroupView, 0, len(groups))
	for _, g := range groups {
		total, inv := r.shown(g.Amount), invoicedSum(invoiced, g.POLines)
		label := g.Vendor
		if label == "" {
			label = notCompleted
		}
		list = append(list, VendorGroupView{Vendor: g.Vendor, Label: label, PurchaseRequests: g.Count,
			AmountDisplay: total, InvoicedDisplay: r.shown(inv)})
	}
	return list, nil
}

func (r *reader) row(pr PurchaseRequest) Row {
	plan, _ := r.plan(pr)
	row := Row{ID: pr.ID, Name: pr.Name, Vendor: pr.Vendor, Status: pr.Status, ChangePending: pr.ChangePending,
		AmountDisplay: r.shown(r.usd(pr.Lines)), InvoicedDisplay: r.shown(r.invoicedTotal(pr.Lines)),
		Owner: r.person(pr.OwnerID), CreatedAt: pr.CreatedAt, ApprovalWorkflow: r.workflows[pr.WorkflowID],
		GSAPPRNumber: pr.GSAP.PRNumber, WaitingForGSAP: waiting(pr), Plan: plan, Lines: make([]RowLine, 0, len(pr.Lines))}
	for _, l := range pr.Lines {
		row.Lines = append(row.Lines, RowLine{ID: l.ID, LineItem: r.lineItemRef(l.LineItemID),
			LineItemOwner: r.person(r.facts[l.LineItemID].OwnerID), HXKey: HXKey(l.LineItemID), Amount: l.money(),
			DeliveryDate: l.DeliveryDate, SentOn: l.SentOn, PONumber: l.PO.Number, POLine: l.PO.Line,
			POLineStatus: l.PO.Status, Invoiced: r.lineInvoiced(l)})
	}
	return row
}

func checkListQuery(q ListQuery) error {
	var problems errs.Problems
	if !slices.Contains(sorts, q.Sort) {
		problems.Add("sort", msgSortUnknown)
	}
	if !slices.Contains(pageSizes, q.PageSize) {
		problems.Add("pageSize", msgPageSize)
	}
	if q.Page < 1 {
		problems.Add("page", msgPageNumber)
	}
	return problems.Err()
}

func invoicedSum(invoiced map[POLineRef]money.Amount, refs []POLineRef) money.Amount {
	var total money.Amount
	for _, ref := range refs {
		total = total.Add(invoiced[ref])
	}
	return total
}
