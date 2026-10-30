package purchaserequest

import (
	"context"
	"encoding/json"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// PostgresStore keeps purchase requests on PostgreSQL. The reads that serve a person filter in SQL with
// purchase_request_in_scope (migration 00020); Get, Claims and All read whoever asks.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (PurchaseRequest, bool, error) {
	return firstOf(s.read(ctx, s.db.Queries(ctx).ListPurchaseRequests, []string{id}))
}

func (s *PostgresStore) All(ctx context.Context) ([]PurchaseRequest, error) {
	return s.read(ctx, s.db.Queries(ctx).ListPurchaseRequests, nil)
}

func (s *PostgresStore) GetInScope(ctx context.Context, scope ViewScope, id string) (PurchaseRequest, bool, error) {
	return firstOf(s.ListInScope(ctx, scope, []string{id}))
}

func (s *PostgresStore) ListInScope(ctx context.Context, scope ViewScope, ids []string) ([]PurchaseRequest, error) {
	if ids == nil {
		ids = []string{}
	}
	return s.inScope(ctx, scopeParams(scope, ids, nil))
}

func (s *PostgresStore) OnLineItems(ctx context.Context, scope ViewScope, lineItemIDs []string) ([]PurchaseRequest, error) {
	if lineItemIDs == nil {
		lineItemIDs = []string{}
	}
	return s.inScope(ctx, scopeParams(scope, nil, lineItemIDs))
}

func scopeParams(scope ViewScope, ids, lineItemIDs []string) sqlcdb.ListPurchaseRequestsInScopeParams {
	return sqlcdb.ListPurchaseRequestsInScopeParams{
		Ids: ids, LineItemIds: lineItemIDs, SeesEverything: scope.Plans.Everything || scope.List.Everything,
		ViewerID: scope.Plans.PersonID, ViewsPlans: scope.Plans.Holds, ViewerTeamIds: nonNil(scope.Plans.TeamIDs),
		Lists: scope.List.Holds, ListTeamIds: nonNil(scope.List.TeamIDs),
	}
}

func (s *PostgresStore) inScope(ctx context.Context, params sqlcdb.ListPurchaseRequestsInScopeParams) ([]PurchaseRequest, error) {
	rows, err := s.db.Queries(ctx).ListPurchaseRequestsInScope(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("select purchase requests in scope: %w", err)
	}
	return s.withLines(ctx, rows)
}

func (s *PostgresStore) read(ctx context.Context, query func(context.Context, []string) ([]sqlcdb.PurchaseRequest, error),
	ids []string) ([]PurchaseRequest, error) {
	rows, err := query(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("select purchase requests: %w", err)
	}
	return s.withLines(ctx, rows)
}

// withLines turns rows into purchase requests with their lines, read in one more statement.
func (s *PostgresStore) withLines(ctx context.Context, rows []sqlcdb.PurchaseRequest) ([]PurchaseRequest, error) {
	list := make([]PurchaseRequest, 0, len(rows))
	if len(rows) == 0 {
		return list, nil
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	lines, err := s.db.Queries(ctx).ListPurchaseRequestLines(ctx, sqlcdb.ListPurchaseRequestLinesParams{PurchaseRequestIds: ids})
	if err != nil {
		return nil, fmt.Errorf("select purchase request lines: %w", err)
	}
	byPR := map[string][]Line{}
	for _, row := range lines {
		l, err := lineFromRow(row)
		if err != nil {
			return nil, err
		}
		byPR[l.PurchaseRequestID] = append(byPR[l.PurchaseRequestID], l)
	}
	for _, r := range rows {
		pr, err := fromRow(r)
		if err != nil {
			return nil, err
		}
		pr.Lines = byPR[pr.ID]
		if pr.Lines == nil {
			pr.Lines = []Line{}
		}
		list = append(list, pr)
	}
	return list, nil
}

func (s *PostgresStore) Claims(ctx context.Context, lineItemIDs []string) ([]Claim, error) {
	rows, err := s.db.Queries(ctx).ListPurchaseRequestClaims(ctx, nonNil(lineItemIDs))
	if err != nil {
		return nil, fmt.Errorf("select purchase request claims: %w", err)
	}
	claims := make([]Claim, 0, len(rows))
	for _, r := range rows {
		claims = append(claims, Claim{LineID: r.ID, PurchaseRequestID: r.PurchaseRequestID, PurchaseRequest: r.Name,
			LineItemID: r.LineItemID, Status: Status(r.Status),
			Amount: money.Money{Amount: money.FromCents(r.AmountMinor), Currency: r.Currency}})
	}
	changing, err := s.db.Queries(ctx).ListPurchaseRequestChangesInReview(ctx, nonNil(lineItemIDs))
	if err != nil {
		return nil, fmt.Errorf("select purchase requests with a change in review: %w", err)
	}
	prs := make([]PurchaseRequest, 0, len(changing))
	for _, r := range changing {
		pr, err := fromRow(r)
		if err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	return WithChangesInReview(claims, prs, lineItemIDs), nil
}

func (s *PostgresStore) Put(ctx context.Context, pr PurchaseRequest) error {
	params, err := prParams(pr)
	if err != nil {
		return err
	}
	if err = s.db.Queries(ctx).PutPurchaseRequest(ctx, params); err != nil {
		return fmt.Errorf("save purchase request %s: %w", pr.ID, err)
	}
	for _, l := range pr.Lines {
		if err = s.PutLine(ctx, l); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeletePurchaseRequest(ctx, id); err != nil {
		return fmt.Errorf("delete purchase request %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetLine(ctx context.Context, id string) (Line, bool, error) {
	rows, err := s.db.Queries(ctx).ListPurchaseRequestLines(ctx, sqlcdb.ListPurchaseRequestLinesParams{Ids: []string{id}})
	if err != nil || len(rows) == 0 {
		return Line{}, false, err
	}
	l, err := lineFromRow(rows[0])
	return l, err == nil, err
}

func (s *PostgresStore) PutLine(ctx context.Context, l Line) error {
	params, err := lineParams(l)
	if err != nil {
		return err
	}
	if err = s.db.Queries(ctx).PutPurchaseRequestLine(ctx, params); err != nil {
		return fmt.Errorf("save purchase request line %s: %w", l.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteLine(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeletePurchaseRequestLine(ctx, id); err != nil {
		return fmt.Errorf("delete purchase request line %s: %w", id, err)
	}
	return nil
}

func prParams(pr PurchaseRequest) (sqlcdb.PutPurchaseRequestParams, error) {
	attributes, err := nonNilValues(pr.Attributes).Column()
	if err != nil {
		return sqlcdb.PutPurchaseRequestParams{}, err
	}
	var approved []byte
	if pr.Approved != nil {
		if approved, err = json.Marshal(pr.Approved); err != nil {
			return sqlcdb.PutPurchaseRequestParams{}, err
		}
	}
	return sqlcdb.PutPurchaseRequestParams{
		ID: pr.ID, PlanID: pr.PlanID, Name: pr.Name, Description: pr.Description, OwnerID: pr.OwnerID,
		MoaApproverID: database.NullIfEmpty(pr.MoAApproverID), ApprovalWorkflowID: pr.WorkflowID,
		ModificationWorkflowID: pr.ModificationWorkflowID, Status: string(pr.Status), ChangePending: pr.ChangePending,
		Attributes: attributes, Vendor: pr.Vendor, ApprovedVersion: approved, GsapPrNumber: pr.GSAP.PRNumber,
		GsapPrDate: database.DateColumn(pr.GSAP.PRDate), GsapSource: pr.GSAP.Source, ClosedAt: pr.ClosedAt,
		Version: pr.Version, CreatedAt: pr.CreatedAt, CreatedByID: pr.CreatedBy.ID, CreatedByName: pr.CreatedBy.Name,
		UpdatedAt: pr.UpdatedAt, UpdatedByID: pr.UpdatedBy.ID, UpdatedByName: pr.UpdatedBy.Name,
	}, nil
}

func fromRow(r sqlcdb.PurchaseRequest) (PurchaseRequest, error) {
	attributes, err := settings.AttributeValuesOf(r.Attributes)
	if err != nil {
		return PurchaseRequest{}, err
	}
	pr := PurchaseRequest{
		ID: r.ID, PlanID: r.PlanID, Name: r.Name, Description: r.Description, OwnerID: r.OwnerID,
		MoAApproverID: database.EmptyIfNull(r.MoaApproverID), WorkflowID: r.ApprovalWorkflowID,
		ModificationWorkflowID: r.ModificationWorkflowID, Status: Status(r.Status), ChangePending: r.ChangePending,
		Attributes: attributes, Vendor: r.Vendor, ClosedAt: r.ClosedAt,
		GSAP: GSAPHeader{PRNumber: r.GsapPrNumber, PRDate: database.DateOf(r.GsapPrDate), Source: r.GsapSource},
		Meta: record.Meta{Version: r.Version, CreatedAt: r.CreatedAt, CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
			UpdatedAt: r.UpdatedAt, UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName}},
	}
	if len(r.ApprovedVersion) > 0 {
		pr.Approved = &Version{}
		if err = json.Unmarshal(r.ApprovedVersion, pr.Approved); err != nil {
			return PurchaseRequest{}, err
		}
	}
	return pr, nil
}

func lineParams(l Line) (sqlcdb.PutPurchaseRequestLineParams, error) {
	attributes, err := nonNilValues(l.Attributes).Column()
	if err != nil {
		return sqlcdb.PutPurchaseRequestLineParams{}, err
	}
	p := sqlcdb.PutPurchaseRequestLineParams{
		ID: l.ID, PurchaseRequestID: l.PurchaseRequestID, Position: l.Position, LineItemID: l.LineItemID,
		Currency: l.Currency, AmountMinor: l.Amount.Cents(), DeliveryDate: l.DeliveryDate.Time(),
		OverallLimitMinor: minorOf(l.OverallLimit), Description: l.Description, Attributes: attributes,
		SentOn: database.DateColumn(l.SentOn), PoNumber: l.PO.Number, PoLine: l.PO.Line, PoLineStatus: l.PO.Status,
		PoDate: database.DateColumn(l.PO.Date), Version: l.Version, CreatedAt: l.CreatedAt, CreatedByID: l.CreatedBy.ID,
		CreatedByName: l.CreatedBy.Name, UpdatedAt: l.UpdatedAt, UpdatedByID: l.UpdatedBy.ID, UpdatedByName: l.UpdatedBy.Name,
	}
	if l.PO.Value != nil {
		cents := l.PO.Value.Amount.Cents()
		p.PoValueMinor, p.PoCurrency = &cents, l.PO.Value.Currency
	}
	return p, nil
}

func lineFromRow(r sqlcdb.PurchaseRequestLine) (Line, error) {
	attributes, err := settings.AttributeValuesOf(r.Attributes)
	if err != nil {
		return Line{}, err
	}
	l := Line{
		ID: r.ID, PurchaseRequestID: r.PurchaseRequestID, Position: r.Position, LineItemID: r.LineItemID,
		Currency: r.Currency, Amount: money.FromCents(r.AmountMinor), DeliveryDate: date.Today(r.DeliveryDate),
		Description: r.Description, Attributes: attributes, SentOn: database.DateOf(r.SentOn),
		PO: PurchaseOrderLine{Number: r.PoNumber, Line: r.PoLine, Status: r.PoLineStatus, Date: database.DateOf(r.PoDate)},
		Meta: record.Meta{Version: r.Version, CreatedAt: r.CreatedAt, CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
			UpdatedAt: r.UpdatedAt, UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName}},
	}
	if r.OverallLimitMinor != nil {
		limit := money.FromCents(*r.OverallLimitMinor)
		l.OverallLimit = &limit
	}
	if r.PoValueMinor != nil {
		l.PO.Value = &money.Money{Amount: money.FromCents(*r.PoValueMinor), Currency: r.PoCurrency}
	}
	return l, nil
}

func minorOf(a *money.Amount) *int64 {
	if a == nil {
		return nil
	}
	cents := a.Cents()
	return &cents
}

func firstOf(list []PurchaseRequest, err error) (PurchaseRequest, bool, error) {
	if err != nil || len(list) == 0 {
		return PurchaseRequest{}, false, err
	}
	return list[0], true, nil
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
