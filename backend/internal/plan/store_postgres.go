package plan

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// keptArea is what the transaction memo keeps of plans.
const keptArea = "plans"

// PostgresStore is the Store on PostgreSQL. Beside the Store's reads of every plan (for internal
// work such as usage counts), ListInScope and GetInScope read only what the person acting may see.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Plan, bool, error) {
	rows, err := s.records().ByID(ctx, []string{id})
	if err != nil {
		return Plan{}, false, fmt.Errorf("select plan %s: %w", id, err)
	}
	return onePlan(plansFromRows(rows))
}

func (s *PostgresStore) List(ctx context.Context) ([]Plan, error) {
	return s.list(ctx, sqlcdb.ListPlansParams{})
}

func (s *PostgresStore) PlansOfYear(ctx context.Context, year int) ([]Plan, error) {
	return s.list(ctx, sqlcdb.ListPlansParams{Year: &year})
}

func (s *PostgresStore) ListInScope(ctx context.Context, scope authz.Scope) ([]Plan, error) {
	return s.listInScope(ctx, scope, nil)
}

// GetInScope finds the plan id only when the scope sees it: a plan outside it is reported as missing.
func (s *PostgresStore) GetInScope(ctx context.Context, scope authz.Scope, id string) (Plan, bool, error) {
	return onePlan(s.listInScope(ctx, scope, &id))
}

// Put saves p normalised, so a plan saved without feature 016's fields reads with their defaults.
func (s *PostgresStore) Put(ctx context.Context, p Plan) error {
	plan := p.Normalised()
	params, err := planParams(plan)
	if err != nil {
		return fmt.Errorf("encode plan %s: %w", plan.ID, err)
	}
	q := s.change(ctx)
	months, amounts := plan.BudgetPhasing.MonthColumns()
	err = q.PutPlan(ctx, params)
	if err == nil {
		err = q.DeletePlanTeams(ctx, plan.ID)
	}
	if err == nil {
		err = q.InsertPlanTeams(ctx, sqlcdb.InsertPlanTeamsParams{PlanID: plan.ID, TeamIds: plan.TeamIDs})
	}
	if err == nil {
		err = q.DeletePlanPhasingMonths(ctx, plan.ID)
	}
	if err == nil {
		err = q.InsertPlanPhasingMonths(ctx, sqlcdb.InsertPlanPhasingMonthsParams{PlanID: plan.ID, Months: months, Amounts: amounts})
	}
	if err != nil {
		return fmt.Errorf("save plan %s: %w", plan.ID, err)
	}
	return nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	if err := s.change(ctx).DeletePlan(ctx, id); err != nil {
		return fmt.Errorf("delete plan %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) list(ctx context.Context, filter sqlcdb.ListPlansParams) ([]Plan, error) {
	rows, err := s.db.Queries(ctx).ListPlans(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("select plans: %w", err)
	}
	s.records().Keep(ctx, rows)
	return plansFromRows(rows)
}

// records keeps the plans a transaction reads by id (memo): pages read a record's plan for each record.
func (s *PostgresStore) records() memo.Records[sqlcdb.ListPlansRow] {
	return memo.Records[sqlcdb.ListPlansRow]{
		Area: keptArea, ID: func(r sqlcdb.ListPlansRow) string { return r.Plan.ID },
		Read: func(ctx context.Context, ids, _ []string) ([]sqlcdb.ListPlansRow, error) {
			return s.db.Queries(ctx).ListPlans(ctx, sqlcdb.ListPlansParams{Ids: ids})
		},
	}
}

func (s *PostgresStore) change(ctx context.Context) *sqlcdb.Queries {
	memo.Forget(ctx, keptArea)
	return s.db.Queries(ctx)
}

func (s *PostgresStore) listInScope(ctx context.Context, scope authz.Scope, id *string) ([]Plan, error) {
	rows, err := s.db.Queries(ctx).ListPlansInScope(ctx, sqlcdb.ListPlansInScopeParams{
		ID: id, SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select plans in scope: %w", err)
	}
	listRows := make([]sqlcdb.ListPlansRow, 0, len(rows))
	for _, row := range rows {
		listRows = append(listRows, sqlcdb.ListPlansRow(row))
	}
	s.records().Keep(ctx, listRows)
	return plansFromRows(listRows)
}

func onePlan(plans []Plan, err error) (Plan, bool, error) {
	if err != nil || len(plans) == 0 {
		return Plan{}, false, err
	}
	return plans[0], true, nil
}

func planParams(p Plan) (sqlcdb.PutPlanParams, error) {
	encoded, err := p.Attributes.Column()
	params := sqlcdb.PutPlanParams{
		ID: p.ID, Name: p.Name, Year: p.Year, ModelID: p.ModelID, OwnerID: database.NullIfEmpty(p.OwnerID), Aop: p.AOP,
		Description: p.Description, DefaultFinanceBudgetID: database.NullIfEmpty(p.DefaultFinanceBudgetID),
		DefaultSpread: string(p.DefaultSpread), Bcpir: p.BCPIR, GoalStage: string(p.GoalStage), GoalAmount: p.GoalAmount,
		Attributes: encoded, Status: string(p.Status), Currency: p.Currency, PlanType: string(p.PlanType),
		PhasingMethod: string(p.BudgetPhasing.Method), Version: p.Version, CreatedAt: p.CreatedAt,
		CreatedByID: p.CreatedBy.ID, CreatedByName: p.CreatedBy.Name, UpdatedAt: p.UpdatedAt,
		UpdatedByID: p.UpdatedBy.ID, UpdatedByName: p.UpdatedBy.Name,
	}
	if p.Budget != nil {
		cents := p.Budget.Cents()
		params.BudgetMinor = &cents
	}
	return params, err
}

func plansFromRows(rows []sqlcdb.ListPlansRow) ([]Plan, error) {
	plans := make([]Plan, 0, len(rows))
	for _, row := range rows {
		r := row.Plan
		p := Plan{
			ID: r.ID, Name: r.Name, Year: r.Year, ModelID: r.ModelID, TeamIDs: slices.Clone(row.TeamIds),
			OwnerID: database.EmptyIfNull(r.OwnerID), AOP: r.Aop, Description: r.Description,
			DefaultFinanceBudgetID: database.EmptyIfNull(r.DefaultFinanceBudgetID), DefaultSpread: Spread(r.DefaultSpread),
			BCPIR: r.Bcpir, GoalStage: GoalStage(r.GoalStage), GoalAmount: r.GoalAmount, Status: Status(r.Status),
			Currency: r.Currency, PlanType: PlanType(r.PlanType),
			BudgetPhasing: rollup.PhasingOf(r.PhasingMethod, row.PhasingMonths, row.PhasingAmounts),
			Meta: record.Meta{
				Version:   r.Version,
				CreatedAt: r.CreatedAt,
				CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
				UpdatedAt: r.UpdatedAt,
				UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName},
			},
		}
		if r.BudgetMinor != nil {
			budget := money.FromCents(*r.BudgetMinor)
			p.Budget = &budget
		}
		attributes, err := settings.AttributeValuesOf(r.Attributes)
		if err != nil {
			return nil, fmt.Errorf("decode plan %s: %w", r.ID, err)
		}
		p.Attributes = attributes
		plans = append(plans, p.Normalised())
	}
	return plans, nil
}
