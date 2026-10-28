package purchaserequest

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// Page reads every matching purchase request's id, vendor and USD amount in one statement, in page order,
// then the page's purchase requests with their lines and the PO lines of its vendors (research R17).
func (s *PostgresStore) Page(ctx context.Context, list authz.Scope, q ListQuery) (Page, error) {
	matches, err := s.db.Queries(ctx).PagePurchaseRequests(ctx, pageParams(list, q))
	if err != nil {
		return Page{}, fmt.Errorf("select the purchase requests page: %w", err)
	}
	from := min((q.Page-1)*q.PageSize, len(matches))
	onPage := matches[from:min(from+q.PageSize, len(matches))]
	ids := make([]string, 0, len(onPage))
	for _, m := range onPage {
		ids = append(ids, m.ID)
	}
	items, err := s.read(ctx, s.db.Queries(ctx).ListPurchaseRequests, ids)
	if err != nil {
		return Page{}, err
	}
	slices.SortFunc(items, func(a, b PurchaseRequest) int { return slices.Index(ids, a.ID) - slices.Index(ids, b.ID) })
	vendors, err := s.vendorGroups(ctx, matches, onPage)
	return Page{Items: items, Total: len(matches), Vendors: vendors}, err
}

func pageParams(list authz.Scope, q ListQuery) sqlcdb.PagePurchaseRequestsParams {
	p := sqlcdb.PagePurchaseRequestsParams{
		Sort: q.Sort, SeesEverything: list.Everything, ViewerID: list.PersonID, Holds: list.Holds,
		TeamIds: nonNil(list.TeamIDs), Search: q.Search, Statuses: []string{}, TypeIds: nonNil(q.TypeOptionIDs),
		TypeNotCompleted: q.TypeNotCompleted, TypeAttribute: q.TypeAttributeID, Vendors: nonNil(q.Vendors),
		VendorNotCompleted: q.VendorNotCompleted, OwnerIds: nonNil(q.OwnerIDs), WaitingForGsap: q.WaitingForGSAP,
		RateCurrencies: []string{}, RateMicros: []int64{},
	}
	for _, status := range q.Statuses {
		p.Statuses = append(p.Statuses, string(status))
	}
	for code, rate := range q.Rates {
		p.RateCurrencies, p.RateMicros = append(p.RateCurrencies, code), append(p.RateMicros, rate.Micros())
	}
	if q.CreatedFrom != nil {
		from := q.CreatedFrom.Time()
		p.CreatedFrom = &from
	}
	if q.CreatedTo != nil {
		before := q.CreatedTo.AddDays(1).Time()
		p.CreatedBefore = &before
	}
	return p
}

// vendorGroups counts, for each vendor on the page, its matching purchase requests, their USD total and PO lines.
func (s *PostgresStore) vendorGroups(ctx context.Context, matches, onPage []sqlcdb.PagePurchaseRequestsRow) ([]VendorGroup, error) {
	var groups []VendorGroup
	var prIDs []string
	for _, m := range onPage {
		if !slices.ContainsFunc(groups, func(g VendorGroup) bool { return strings.EqualFold(g.Vendor, m.Vendor) }) {
			groups = append(groups, VendorGroup{Vendor: m.Vendor, POLines: []POLineRef{}})
		}
	}
	for _, m := range matches {
		i := slices.IndexFunc(groups, func(g VendorGroup) bool { return strings.EqualFold(g.Vendor, m.Vendor) })
		if i >= 0 {
			groups[i].Count++
			groups[i].Amount = groups[i].Amount.Add(money.FromCents(m.AmountUsdMinor))
			prIDs = append(prIDs, m.ID)
		}
	}
	if len(prIDs) == 0 {
		return []VendorGroup{}, nil
	}
	return groups, s.addPOLines(ctx, groups, matches, prIDs)
}

// addPOLines adds to each vendor group the PO lines of its matching purchase requests, for its invoiced total.
func (s *PostgresStore) addPOLines(ctx context.Context, groups []VendorGroup, matches []sqlcdb.PagePurchaseRequestsRow, prIDs []string) error {
	lines, err := s.db.Queries(ctx).ListPurchaseRequestLines(ctx, sqlcdb.ListPurchaseRequestLinesParams{PurchaseRequestIds: prIDs})
	if err != nil {
		return fmt.Errorf("select the PO lines of the page's vendors: %w", err)
	}
	vendorOf := map[string]string{}
	for _, m := range matches {
		vendorOf[m.ID] = m.Vendor
	}
	for _, l := range lines {
		i := slices.IndexFunc(groups, func(g VendorGroup) bool { return strings.EqualFold(g.Vendor, vendorOf[l.PurchaseRequestID]) })
		if l.PoLine != "" && i >= 0 {
			groups[i].POLines = append(groups[i].POLines, POLineRef{PONumber: l.PoNumber, Line: l.PoLine})
		}
	}
	return nil
}

func (s *PostgresStore) AddRouting(ctx context.Context, r Routing) error {
	recipients, err := json.Marshal(r.Recipients)
	if err != nil {
		return err
	}
	form, err := json.Marshal(r.Form)
	if err != nil {
		return err
	}
	err = s.db.Queries(ctx).AddPurchaseRequestRouting(ctx, sqlcdb.AddPurchaseRequestRoutingParams{
		ID: r.ID, PurchaseRequestID: r.PurchaseRequestID, Type: string(r.Type), RoutedAt: r.RoutedAt,
		RoutedByID: r.RoutedBy.ID, RoutedByName: r.RoutedBy.Name, Recipients: recipients, Form: form,
	})
	if err != nil {
		return fmt.Errorf("add routing %s: %w", r.ID, err)
	}
	return nil
}

func (s *PostgresStore) Routings(ctx context.Context, purchaseRequestID string) ([]Routing, error) {
	rows, err := s.db.Queries(ctx).ListPurchaseRequestRoutings(ctx, sqlcdb.ListPurchaseRequestRoutingsParams{
		PurchaseRequestID: &purchaseRequestID,
	})
	if err != nil {
		return nil, fmt.Errorf("select the routings of %s: %w", purchaseRequestID, err)
	}
	list := make([]Routing, 0, len(rows))
	for _, row := range rows {
		r := Routing{ID: row.ID, PurchaseRequestID: row.PurchaseRequestID, Type: RoutingType(row.Type),
			RoutedAt: row.RoutedAt, RoutedBy: actor.Actor{ID: row.RoutedByID, Name: row.RoutedByName}}
		if err = json.Unmarshal(row.Recipients, &r.Recipients); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(row.Form, &r.Form); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func (s *PostgresStore) Configuration(ctx context.Context) (Configuration, error) {
	rows, err := s.db.Queries(ctx).GetPurchaseRequestConfiguration(ctx)
	if err != nil {
		return Configuration{}, fmt.Errorf("select the purchase request configuration: %w", err)
	}
	if len(rows) == 0 {
		return DefaultConfiguration(), nil
	}
	r := rows[0]
	return Configuration{PurchaseSupport: r.PurchaseSupport, LinesOnApproved: LinesOnApproved(r.LinesOnApproved),
		Meta: record.Meta{Version: r.Version, CreatedAt: r.CreatedAt, CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
			UpdatedAt: r.UpdatedAt, UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName}}}, nil
}

func (s *PostgresStore) PutConfiguration(ctx context.Context, c Configuration) error {
	created, by := cmp.Or(c.CreatedAt, c.UpdatedAt), cmp.Or(c.CreatedBy, c.UpdatedBy)
	err := s.db.Queries(ctx).PutPurchaseRequestConfiguration(ctx, sqlcdb.PutPurchaseRequestConfigurationParams{
		PurchaseSupport: c.PurchaseSupport, LinesOnApproved: string(c.LinesOnApproved), Version: c.Version,
		CreatedAt: created, CreatedByID: by.ID, CreatedByName: by.Name, UpdatedAt: c.UpdatedAt,
		UpdatedByID: c.UpdatedBy.ID, UpdatedByName: c.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save the purchase request configuration: %w", err)
	}
	return nil
}
