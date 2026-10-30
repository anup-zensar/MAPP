package dashboard

import (
	"context"
	"slices"
	"time"

	"mapp/internal/allwork"
	"mapp/internal/approval"
	"mapp/internal/financials"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
)

// completenessColumn makes All work read each plan's own needed fields too.
const completenessColumn = "completeness"

// request is what one dashboard or export covers; plan, when set, keeps that plan only (a plan's export).
type request struct {
	kind  Kind
	query Query
	plan  string
}

// input is everything a dashboard is worked out from, read in one transaction.
type input struct {
	request
	now       time.Time
	today     date.Date
	display   money.Display
	choices   Choices
	team      *people.TeamRef
	data      allwork.YearData
	board     *board
	teams     map[string]people.TeamRef
	spend     map[string]map[string]financials.Facts
	rounds    []approval.RoundSummary
	purchases purchaserequest.Summaries
	global    []GlobalBudget
	canExport bool
}

func (s *Service) read(ctx context.Context, a actor.Actor, r request) (*input, error) {
	now := s.now()
	in := &input{request: r, now: now, today: date.Today(now)}
	if in.query.Year == 0 {
		in.query.Year = in.today.Year()
	}
	work, err := s.deps.Planning.Choices(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	if err = s.readChoices(ctx, a, in, work); err != nil {
		return nil, err
	}
	if in.data, err = s.deps.Planning.Year(ctx, in.query.Year, in.today, []string{completenessColumn}); err != nil {
		return nil, err
	}
	in.board = newBoard(in.data.Nodes, in.keeps, work.SeesFinance, work.FinancePlans)
	if err = s.readPanels(ctx, in); err != nil {
		return nil, err
	}
	in.teams, err = s.deps.Directory.TeamRefs(ctx, in.board.teamIDs())
	return in, err
}

// readChoices reads the filters' choices and takes the currency and the market from them.
func (s *Service) readChoices(ctx context.Context, a actor.Actor, in *input, work allwork.Choices) error {
	currencies, err := s.deps.Directory.Currencies(ctx, a.ID)
	if err != nil {
		return err
	}
	years, err := s.deps.Planning.Years(ctx)
	if err != nil {
		return err
	}
	in.choices = Choices{Years: yearChoices(years, in.today.Year(), in.query.Year), Teams: teamChoices(work.Teams)}
	in.chooseCurrency(currencies)
	if in.kind != KindRegional {
		return nil
	}
	in.chooseTeam()
	refs, err := s.deps.Directory.TeamRefs(ctx, []string{in.query.TeamID})
	ref := teamRef(refs, in.query.TeamID)
	in.team = &ref
	return err
}

// readPanels reads, one statement each, what the board's records need: their purchase orders and payments,
// purchase requests and approval rounds, and the year's global budgets for a person who may see them.
func (s *Service) readPanels(ctx context.Context, in *input) error {
	var err error
	if ids := in.board.financeLineItemIDs(); len(ids) > 0 {
		if in.spend, err = s.deps.Spending.SpendByMonth(ctx, ids); err != nil {
			return err
		}
	}
	in.board.countSpend(in.spend, in.query.Year)
	if in.purchases, err = s.deps.Purchases.Summaries(ctx, in.board.lineItemIDs()); err != nil {
		return err
	}
	if in.rounds, err = s.deps.Rounds.RoundSummaries(ctx, in.board.subjectIDs(in.purchases.Items)); err != nil {
		return err
	}
	grants, signedIn := authz.From(ctx)
	in.canExport = !signedIn || grants.Has(authz.DataExport)
	if signedIn && !grants.Has(authz.GlobalBudgetsView) {
		return nil
	}
	year, err := s.deps.GlobalBudgets.YearFigures(ctx, in.query.Year)
	in.global = globalBudgets(year, in.display)
	return err
}

// chooseCurrency takes the asked currency of Settings, else the display currency, else US dollars.
func (in *input) chooseCurrency(c Currencies) {
	asked := in.query.Currency
	in.query.Currency, in.display = money.BaseCurrency, money.DisplayUSD
	for _, currency := range c.Items {
		in.choices.Currencies = append(in.choices.Currencies, CurrencyChoice{Code: currency.Code, Name: currency.Name})
	}
	for _, code := range []string{asked, c.Default} {
		if i := slices.IndexFunc(c.Items, func(x Currency) bool { return x.Code == code }); code != "" && i >= 0 {
			in.query.Currency, in.display = code, money.DisplayIn(code, c.Items[i].Rate)
			return
		}
	}
}

// chooseTeam takes the asked market of the choices, else the person's first team by name, else the first team.
func (in *input) chooseTeam() {
	teams := in.choices.Teams
	if slices.ContainsFunc(teams, func(t TeamChoice) bool { return t.ID == in.query.TeamID }) {
		return
	}
	in.query.TeamID = ""
	if i := slices.IndexFunc(teams, func(t TeamChoice) bool { return t.Mine }); i >= 0 {
		in.query.TeamID = teams[i].ID
	} else if len(teams) > 0 {
		in.query.TeamID = teams[0].ID
	}
}

// keeps says whether a plan of the year is on the dashboard: its type, its market and, for an export, the plan.
func (in *input) keeps(plan *allwork.Node) bool {
	switch {
	case in.query.PlanType != "" && plan.PlanType != in.query.PlanType:
		return false
	case in.kind == KindRegional && !slices.Contains(plan.TeamIDs, in.query.TeamID):
		return false
	}
	return in.plan == "" || plan.ID == in.plan
}

func yearChoices(years []int, current, chosen int) []int {
	all := slices.Clone(years)
	for _, year := range []int{current, chosen} {
		if !slices.Contains(all, year) {
			all = append(all, year)
		}
	}
	slices.Sort(all)
	slices.Reverse(all)
	return all
}

func teamChoices(teams []allwork.Option) []TeamChoice {
	out := make([]TeamChoice, 0, len(teams))
	for _, t := range teams {
		out = append(out, TeamChoice{ID: t.ID, Name: t.Label, Active: !t.Inactive, Mine: t.Mine})
	}
	slices.SortFunc(out, func(x, y TeamChoice) int { return byName(x.Name, y.Name, x.ID, y.ID) })
	return out
}
