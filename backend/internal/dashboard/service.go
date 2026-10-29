package dashboard

import (
	"context"
	"time"

	"mapp/internal/allwork"
	"mapp/internal/platform/actor"
)

type Service struct {
	tx   Transactor
	deps Deps
	now  func() time.Time
}

func NewService(tx Transactor, deps Deps, now func() time.Time) *Service {
	return &Service{tx: tx, deps: deps, now: now}
}

// Executive is the dashboard of every market the person may see (FR-001 – FR-019).
func (s *Service) Executive(ctx context.Context, a actor.Actor, q Query) (View, error) {
	return s.view(ctx, a, request{kind: KindExecutive, query: q})
}

// Regional is the dashboard of one market, with the campaigns of the plan q.PlanID (FR-020 – FR-022).
func (s *Service) Regional(ctx context.Context, a actor.Actor, q Query) (View, error) {
	return s.view(ctx, a, request{kind: KindRegional, query: q})
}

func (s *Service) view(ctx context.Context, a actor.Actor, r request) (View, error) {
	var v View
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		in, err := s.read(ctx, a, r)
		if err != nil {
			return err
		}
		v = in.view()
		return nil
	})
	return v, err
}

func (in *input) view() View {
	b, d := in.board, in.display
	v := View{
		Kind: in.kind, Filters: in.filters(), Choices: in.choices, Team: in.team, PlanCount: len(b.plans),
		Finance:     Finance{Visible: b.financeCount() > 0, PlanCount: b.financeCount()},
		FinanceAsOf: in.data.FinanceAsOf, Totals: b.total().figures(d), Markets: []MarketRow{},
		Plans: b.planRows(in.teams, d), PlanCampaigns: b.planCampaigns(in.query.PlanID, d),
		Months:           b.months(allwork.PlannedByActivity(in.data.Nodes, in.query.Year), in.query.Year, d),
		Approvals:        b.approvals(in.rounds, in.purchases.Items),
		PurchaseRequests: b.purchaseRequests(in.purchases, d), Completeness: b.completeness(),
		GlobalBudgets: in.global, CanExport: in.canExport,
	}
	if in.kind == KindExecutive {
		v.Markets = b.markets(in.teams, d)
	}
	v.OverBudget, v.LeftToPlan = b.campaignLists(d)
	for _, p := range b.plans {
		if p.Budget == nil {
			v.PlansWithoutBudget++
		}
	}
	return v
}

func (in *input) filters() Filters {
	f := Filters{Year: in.query.Year, Currency: in.query.Currency}
	if in.query.PlanType != "" {
		f.PlanType = &in.query.PlanType
	}
	if in.kind == KindRegional {
		f.TeamID = &in.query.TeamID
		if in.query.PlanID != "" {
			f.PlanID = &in.query.PlanID
		}
	}
	return f
}
