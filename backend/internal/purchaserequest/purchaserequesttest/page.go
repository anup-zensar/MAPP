package purchaserequesttest

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
)

var statusOrder = []purchaserequest.Status{ //nolint:gochecknoglobals // the status model's order, as the SQL sorts it
	purchaserequest.StatusCreated, purchaserequest.StatusSubmitted, purchaserequest.StatusInReview,
	purchaserequest.StatusDeclined, purchaserequest.StatusApproved, purchaserequest.StatusClosed,
}

type match struct {
	pr  purchaserequest.PurchaseRequest
	usd money.Amount
}

// Page filters, sorts and pages as PagePurchaseRequests does in SQL.
func (s Store) Page(ctx context.Context, list authz.Scope, q purchaserequest.ListQuery) (purchaserequest.Page, error) {
	all, err := s.All(ctx)
	if err != nil {
		return purchaserequest.Page{}, err
	}
	var matches []match
	for _, pr := range all {
		p, err := s.planFacts(ctx, pr.PlanID)
		if err != nil {
			return purchaserequest.Page{}, err
		}
		if list.Covers(p) && matchesQuery(pr, q) {
			matches = append(matches, match{pr: pr, usd: usdOf(pr, q.Rates)})
		}
	}
	slices.SortStableFunc(matches, func(a, b match) int { return compare(a, b, q.Sort) })
	from := min((q.Page-1)*q.PageSize, len(matches))
	onPage := matches[from:min(from+q.PageSize, len(matches))]
	page := purchaserequest.Page{Items: []purchaserequest.PurchaseRequest{}, Total: len(matches), Vendors: []purchaserequest.VendorGroup{}}
	for _, m := range onPage {
		page.Items = append(page.Items, m.pr)
		if !slices.ContainsFunc(page.Vendors, func(g purchaserequest.VendorGroup) bool { return strings.EqualFold(g.Vendor, m.pr.Vendor) }) {
			page.Vendors = append(page.Vendors, vendorGroup(m.pr.Vendor, matches))
		}
	}
	return page, nil
}

func vendorGroup(vendor string, matches []match) purchaserequest.VendorGroup {
	g := purchaserequest.VendorGroup{Vendor: vendor, POLines: []purchaserequest.POLineRef{}}
	for _, m := range matches {
		if !strings.EqualFold(m.pr.Vendor, vendor) {
			continue
		}
		g.Count++
		g.Amount = g.Amount.Add(m.usd)
		for _, l := range m.pr.Lines {
			if l.PO.Line != "" {
				g.POLines = append(g.POLines, purchaserequest.POLineRef{PONumber: l.PO.Number, Line: l.PO.Line})
			}
		}
	}
	return g
}

func matchesQuery(pr purchaserequest.PurchaseRequest, q purchaserequest.ListQuery) bool {
	return fieldsMatch(pr, q) && searched(pr, q.Search) && vendorOK(pr, q) && typeOK(pr, q) && waitingOK(pr, q)
}

func fieldsMatch(pr purchaserequest.PurchaseRequest, q purchaserequest.ListQuery) bool {
	created := date.Today(pr.CreatedAt)
	from := q.CreatedFrom == nil || !created.Before(*q.CreatedFrom)
	to := q.CreatedTo == nil || !created.After(*q.CreatedTo)
	statusOK := len(q.Statuses) == 0 || slices.Contains(q.Statuses, pr.Status)
	ownerOK := len(q.OwnerIDs) == 0 || slices.Contains(q.OwnerIDs, pr.OwnerID)
	return from && to && statusOK && ownerOK
}

func searched(pr purchaserequest.PurchaseRequest, search string) bool {
	if search == "" {
		return true
	}
	texts := []string{pr.ID, pr.Name, pr.Vendor, pr.GSAP.PRNumber}
	for _, l := range pr.Lines {
		texts = append(texts, purchaserequest.HXKey(l.LineItemID), l.PO.Number)
	}
	return slices.ContainsFunc(texts, func(t string) bool { return strings.Contains(strings.ToLower(t), search) })
}

func vendorOK(pr purchaserequest.PurchaseRequest, q purchaserequest.ListQuery) bool {
	if len(q.Vendors) == 0 && !q.VendorNotCompleted {
		return true
	}
	return slices.Contains(q.Vendors, strings.ToLower(pr.Vendor)) || (q.VendorNotCompleted && pr.Vendor == "")
}

func typeOK(pr purchaserequest.PurchaseRequest, q purchaserequest.ListQuery) bool {
	if len(q.TypeOptionIDs) == 0 && !q.TypeNotCompleted {
		return true
	}
	var chosen []string
	if raw, found := pr.Attributes[q.TypeAttributeID]; found {
		_ = jsonStrings(raw, &chosen)
	}
	if len(chosen) == 0 {
		return q.TypeNotCompleted
	}
	return slices.ContainsFunc(chosen, func(id string) bool { return slices.Contains(q.TypeOptionIDs, id) })
}

func waitingOK(pr purchaserequest.PurchaseRequest, q purchaserequest.ListQuery) bool {
	if !q.WaitingForGSAP {
		return true
	}
	routed := slices.ContainsFunc(pr.Lines, func(l purchaserequest.Line) bool { return l.SentOn != nil })
	pending := slices.ContainsFunc(pr.Lines, func(l purchaserequest.Line) bool {
		return l.PO.Value == nil || l.PO.Value.Currency != l.Currency || l.PO.Value.Amount.Cmp(l.Amount) != 0
	})
	return pr.Status == purchaserequest.StatusApproved && routed && pending
}

func usdOf(pr purchaserequest.PurchaseRequest, rates money.Rates) money.Amount {
	var total money.Amount
	for _, l := range pr.Lines {
		converted, err := rates.Convert(money.Money{Amount: l.Amount, Currency: l.Currency}, money.BaseCurrency)
		if err != nil {
			converted.Amount = l.Amount
		}
		total = total.Add(converted.Amount)
	}
	return total
}

func compare(a, b match, sort string) int {
	if byVendor := cmp.Or(cmp.Compare(boolInt(a.pr.Vendor == ""), boolInt(b.pr.Vendor == "")),
		strings.Compare(strings.ToLower(a.pr.Vendor), strings.ToLower(b.pr.Vendor))); byVendor != 0 {
		return byVendor
	}
	key, descending := strings.TrimPrefix(sort, "-"), strings.HasPrefix(sort, "-")
	c := 0
	switch key {
	case "name":
		c = strings.Compare(strings.ToLower(a.pr.Name), strings.ToLower(b.pr.Name))
	case "status":
		c = slices.Index(statusOrder, a.pr.Status) - slices.Index(statusOrder, b.pr.Status)
	case "amount":
		c = a.usd.Cmp(b.usd)
	case "created":
		c = a.pr.CreatedAt.Compare(b.pr.CreatedAt)
	}
	if descending {
		c = -c
	}
	return cmp.Or(c, compareIDs(a.pr.ID, b.pr.ID, key == "id" && descending))
}

func compareIDs(a, b string, descending bool) int {
	c := cmp.Or(cmp.Compare(len(a), len(b)), strings.Compare(a, b))
	if descending {
		return -c
	}
	return c
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
