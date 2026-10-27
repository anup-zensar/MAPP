package home

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// Service only reads, each operation in one transaction, so all of a card's figures come from one moment.
type Service struct {
	tx  Transactor
	src Sources
}

func NewService(tx Transactor, src Sources) *Service {
	return &Service{tx: tx, src: src}
}

// converter turns US dollars into the display currency, half up to the cent; totals are converted once.
type converter struct {
	code string
	rate money.Rate
}

// A display currency without a usable rate shows US dollars, as the plan page does.
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

func (c converter) of(usd money.Amount) money.Money {
	return money.Money{Amount: money.FromUSD(usd, c.rate), Currency: c.code}
}

func (c converter) optional(usd *money.Amount) *money.Money {
	if usd == nil {
		return nil
	}
	converted := c.of(*usd)
	return &converted
}

type planScope struct {
	all     []Plan
	myTeams map[string]bool
}

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

func (ps planScope) in(scope Scope) []Plan {
	if scope == ScopeEverything {
		return slices.Clone(ps.all)
	}
	return slices.DeleteFunc(slices.Clone(ps.all), func(p Plan) bool { return !ps.isTeamPlan(p) })
}

func (ps planScope) isTeamPlan(p Plan) bool {
	return slices.ContainsFunc(p.Teams, func(t Team) bool { return ps.myTeams[t.ID] })
}

const (
	planPublished     = "published"
	activityDeclined  = "declined"
	activitySubmitted = "submitted"
)

func compareNames(x, y string) int {
	return strings.Compare(strings.ToLower(x), strings.ToLower(y))
}

// groupOrder is the order of the groups of My activities.
var groupOrder = []string{"declined", "created", "submitted", "approved", "inProgress", "complete"}
