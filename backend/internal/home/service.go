package home

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// Service calculates the Home dashboard. It only reads: every operation takes the actor, reads its
// Sources inside one read transaction, so all of a card's figures come from the same moment, and changes
// nothing.
type Service struct {
	tx  store.Transactor
	src Sources
}

// NewService returns the Home service. It reads through src, inside read transactions of tx.
func NewService(tx store.Transactor, src Sources) *Service {
	return &Service{tx: tx, src: src}
}

// converter turns US dollars into the person's display currency: usd × rate, half up to the cent. Totals
// are added in US dollars and converted once (FR-017, D-010).
type converter struct {
	code string
	rate money.Rate
}

// converter returns the converter of a's display currency; one without a usable rate shows US dollars, as
// the plan page does.
func (s *Service) converter(ctx context.Context, a actor.Actor) (converter, error) {
	c, err := s.src.DisplayCurrency(ctx, a)
	if err != nil {
		return converter{}, err
	}
	if c.Code == "" || c.Rate.Micros() <= 0 {
		return converter{code: money.BaseCurrency, rate: money.One}, nil
	}
	return converter{code: c.Code, rate: c.Rate}, nil
}

// of converts usd.
func (c converter) of(usd money.Amount) money.Money {
	return money.Money{Amount: money.FromUSD(usd, c.rate), Currency: c.code}
}

// optional converts usd, or gives nil when there is no amount.
func (c converter) optional(usd *money.Amount) *money.Money {
	if usd == nil {
		return nil
	}
	converted := c.of(*usd)
	return &converted
}

// planScope is every plan and the person's teams: what the cards that follow the My teams / Everything
// switch start from.
type planScope struct {
	all     []Plan
	myTeams map[string]bool // the ids of the person's teams
}

// planScope reads every plan and a's teams.
func (s *Service) planScope(ctx context.Context, a actor.Actor) (planScope, error) {
	teams, err := s.src.MyTeams(ctx, a)
	if err != nil {
		return planScope{}, err
	}
	all, err := s.src.Plans(ctx, a)
	if err != nil {
		return planScope{}, err
	}
	ps := planScope{all: all, myTeams: map[string]bool{}}
	for _, t := range teams {
		ps.myTeams[t.ID] = true
	}
	return ps, nil
}

// in returns the plans of scope (FR-003, FR-004): every plan for Everything; for My teams the plans with at
// least one of the person's teams — each plan once, however many of its teams are the person's.
func (ps planScope) in(scope Scope) []Plan {
	if scope == ScopeEverything {
		return slices.Clone(ps.all)
	}
	return slices.DeleteFunc(slices.Clone(ps.all), func(p Plan) bool { return !ps.isTeamPlan(p) })
}

// isTeamPlan reports whether one of p's teams is one of the person's.
func (ps planScope) isTeamPlan(p Plan) bool {
	return slices.ContainsFunc(p.Teams, func(t Team) bool { return ps.myTeams[t.ID] })
}

// The plan and activity statuses Home counts (domain model §2.1, §2.3).
const (
	planPublished     = "published"
	activityDeclined  = "declined"
	activitySubmitted = "submitted"
)

// compareNames orders names as people read them: ignoring letter case.
func compareNames(x, y string) int {
	return strings.Compare(strings.ToLower(x), strings.ToLower(y))
}

// groupOrder is the order of the groups of My activities: the displayed statuses (FR-014, domain model §2.3).
var groupOrder = []string{"declined", "created", "submitted", "approved", "inProgress", "complete"}
