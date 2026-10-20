package plans

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"northstar/internal/history"
	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
	"northstar/internal/rollup"
	"northstar/internal/settings"
)

// planGoneMessage answers a plan that does not exist (any more).
const planGoneMessage = "This plan no longer exists — someone may have deleted it."

// fallbackColour is the colour of a team that no longer resolves: a neutral one of the palette.
const fallbackColour = "slate"

// Service holds the rules of plans (feature 004): every check, calculation and change of a plan.
type Service struct {
	tx        store.Transactor
	repo      Repository
	ids       *ids.Generator
	history   *history.Service
	settings  Settings
	directory Directory
	now       clock.Clock
	levels    []Level // the levels below the plan, top-down (RegisterLevel)

	allocations Allocations // 014's global-budget allocations (UseAllocations); nil: no plan is allocated
}

// NewService returns the plans service. It keeps plans in repo, gives them readable IDs with generator,
// logs their changes and deletes them through hist, reads Settings and people and teams through settings
// and directory, works in transactions of tx and reads the time from now. It knows no level below the plan
// until RegisterLevel adds one.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service,
	settings Settings, directory Directory, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, history: hist, settings: settings, directory: directory, now: now}
}

// Defaults returns what a new plan starts with for a (FR-006): the current year and the years a plan may
// have, a's active teams by name, a as owner when they have one (R1), the only active model, No for AOP
// and BCPIR, Evenly and INQ.
func (s *Service) Defaults(ctx context.Context, a actor.Actor) (Defaults, error) {
	first, last := s.years()
	defaults := Defaults{
		CurrentYear: s.now().Year(), Year: s.now().Year(), Years: []int{}, TeamIDs: []string{},
		DefaultSpread: SpreadEvenly, GoalStage: StageINQ, Currency: money.BaseCurrency, PlanType: TypeOperational,
	}
	for year := first; year <= last; year++ {
		defaults.Years = append(defaults.Years, year)
	}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		teamIDs, err := s.directory.TeamIDsOf(ctx, a.ID)
		if err != nil {
			return err
		}
		teams, err := s.directory.TeamRefs(ctx, teamIDs)
		if err != nil {
			return err
		}
		active := []people.TeamRef{}
		for _, id := range teamIDs {
			if team, found := teams[id]; found && team.Active {
				active = append(active, team)
			}
		}
		for _, team := range sortedTeams(active) {
			defaults.TeamIDs = append(defaults.TeamIDs, team.ID)
		}
		if len(defaults.TeamIDs) > 0 {
			defaults.OwnerID = &a.ID
		}
		models, err := s.settings.Models(ctx)
		if err != nil {
			return err
		}
		var activeModels []string
		for _, m := range models {
			if m.Active {
				activeModels = append(activeModels, m.ID)
			}
		}
		if len(activeModels) == 1 {
			defaults.ModelID = &activeModels[0]
		}
		return nil
	})
	if err != nil {
		return Defaults{}, err
	}
	return defaults, nil
}

// Create creates a plan from the whole form in one Write (FR-010): every problem is reported at once and
// nothing is saved while there is one; otherwise the plan is Draft with a new permanent ID and a "created"
// log entry. Only name, year and model block saving; what is missing for publishing does not.
func (s *Service) Create(ctx context.Context, a actor.Actor, input PlanInput) (PlanView, error) {
	var view PlanView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var problems errs.Problems
		p, err := s.check(ctx, formOf(input), nil, &problems)
		if err != nil {
			return err
		}
		checkTeams(ctx, authz.PlansCreate, nil, authz.PlanFacts{TeamIDs: p.TeamIDs, CreatedByID: a.ID}, &problems) // 014 FR-013
		if err := problems.Err(); err != nil {
			return err
		}
		if p.ID, err = s.ids.Next(ctx, ids.PrefixPlan); err != nil {
			return err
		}
		p.Status = StatusDraft
		p.Meta = record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, p); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: p.ID, RecordName: p.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		view, err = s.view(ctx, a, p)
		return err
	})
	if err != nil {
		return PlanView{}, err
	}
	return view, nil
}

// The messages of the list's filters (plan "User-visible messages").
const (
	msgChooseScope    = "Choose all or myTeams."
	msgChooseStatus   = "Choose draft or published."
	msgChooseListYear = "Choose a year such as 2026."
)

// filterProblems checks the team scope, the status, the year range and the plan types of a list filter;
// empty means every plan.
func filterProblems(filter ListFilter) errs.Problems {
	var problems errs.Problems
	if !slices.Contains([]string{"", ScopeAll, ScopeMyTeams}, filter.Scope) {
		problems.Add("scope", msgChooseScope)
	}
	if !slices.Contains([]string{"", string(StatusDraft), string(StatusPublished)}, filter.Status) {
		problems.Add("status", msgChooseStatus)
	}
	if filter.YearFrom != nil && filter.YearTo != nil && *filter.YearTo < *filter.YearFrom { // 016 FR-033
		problems.Add("yearTo", msgYearToBeforeFrom)
	}
	for _, t := range filter.Types {
		if !slices.Contains(planTypes, PlanType(t)) {
			problems.Add("type", msgChoosePlanType)
			break
		}
	}
	return problems
}

// msgYearToBeforeFrom refuses a year range that ends before it starts (016 FR-033).
const msgYearToBeforeFrom = "Choose a last year that is not before the first year."

// inRange reports whether p matches the year range and the plan types of filter (016 FR-033): year is the
// alias of a one-year range.
func (filter ListFilter) inRange(p Plan) bool {
	from, to := filter.YearFrom, filter.YearTo
	if filter.Year != nil {
		from, to = filter.Year, filter.Year
	}
	if (from != nil && p.Year < *from) || (to != nil && p.Year > *to) {
		return false
	}
	return len(filter.Types) == 0 || slices.Contains(filter.Types, string(p.PlanType))
}

// List returns the plans of the Plans page (FR-001 – FR-004, D-080): only the plans a may see (014 FR-007).
// The team scope comes next: All plans, or My teams — plans with one of a's teams, which only narrows the
// list (Q-006). Total counts the plans in the scope; the summary covers the scope, the year range and the
// plan types (016 FR-033), whatever the status and the search; the items match every filter — the search is
// any part of the name or the ID, ignoring letter case — newest year first, then by name. Each level below
// is asked once for all the plans (SC-009). Each plan's amounts are in its plan currency with the display
// currency beside; the summary converts each plan's totals to USD before adding them (016 FR-014).
func (s *Service) List(ctx context.Context, a actor.Actor, filter ListFilter) (PlanList, error) {
	if err := filterProblems(filter).Err(); err != nil {
		return PlanList{}, err
	}
	list := PlanList{Items: []PlanSummary{}, Years: []int{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		scoped, err := s.inScope(ctx, a, filter.Scope)
		if err != nil {
			return err
		}
		scoped = visible(ctx, scoped)
		slices.SortFunc(scoped, func(x, y Plan) int {
			return cmp.Or(cmp.Compare(y.Year, x.Year), strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)),
				strings.Compare(x.ID, y.ID))
		})
		planIDs := make([]string, 0, len(scoped))
		for _, p := range scoped {
			planIDs = append(planIDs, p.ID)
			if !slices.Contains(list.Years, p.Year) {
				list.Years = append(list.Years, p.Year) // newest first, as the plans are sorted
			}
		}
		list.Total = len(scoped)
		facts, err := s.facts(ctx, planIDs)
		if err != nil {
			return err
		}
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		rates, err := s.settings.Rates(ctx)
		if err != nil {
			return err
		}
		l, err := s.lookupsFor(ctx, scoped)
		if err != nil {
			return err
		}

		var budgets, plannedCosts []money.Amount // in USD, each plan converted once
		query := strings.ToLower(strings.TrimSpace(filter.Q))
		for _, p := range scoped {
			if !filter.inRange(p) {
				continue
			}
			f := *facts[p.ID]
			list.Summary.Plans++
			if p.Status == StatusPublished {
				list.Summary.Published++
			} else {
				list.Summary.Draft++
			}
			if p.Budget != nil {
				budgets = append(budgets, usdOf(*p.Budget, p.Currency, rates))
			}
			if f.PlannedCost != nil {
				plannedCosts = append(plannedCosts, usdOf(*f.PlannedCost, p.Currency, rates))
			}
			if (filter.Status != "" && string(p.Status) != filter.Status) ||
				!(strings.Contains(strings.ToLower(p.Name), query) || strings.Contains(strings.ToLower(p.ID), query)) {
				continue
			}
			sh, err := s.shownOf(ctx, p, f, l, c.reading(p.Currency, rates))
			if err != nil {
				return err
			}
			own := identity(p.Currency)
			list.Items = append(list.Items, PlanSummary{
				ID: p.ID, Name: p.Name, Year: p.Year, Status: p.Status, AOP: p.AOP, Owner: sh.owner, Teams: sh.teams,
				BusinessUnit: sh.businessUnit, Figures: sh.figures, Completeness: sh.completeness,
				Access:   authz.PlanAccessOf(ctx, AccessFacts(p)), // 014 FR-011
				PlanType: p.PlanType, Currency: p.Currency, PlanBudget: own.optional(p.Budget), PlanPlannedCost: own.optional(f.PlannedCost),
			})
		}
		list.Summary.Budget = c.optional(sumOf(budgets))
		list.Summary.PlannedCost = c.optional(sumOf(plannedCosts))
		return nil
	})
	if err != nil {
		return PlanList{}, err
	}
	return list, nil
}

// inScope returns the plans of the team scope: every plan, or with myTeams those with one of a's teams.
func (s *Service) inScope(ctx context.Context, a actor.Actor, scope string) ([]Plan, error) {
	all, err := s.repo.List(ctx)
	if err != nil || scope != ScopeMyTeams {
		return all, err
	}
	mine, err := s.directory.TeamIDsOf(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	scoped := []Plan{}
	for _, p := range all {
		if slices.ContainsFunc(p.TeamIDs, func(id string) bool { return slices.Contains(mine, id) }) {
			scoped = append(scoped, p)
		}
	}
	return scoped, nil
}

// sumOf adds amounts up; nil when there are none, so a total of nothing shows "—", not 0.00.
func sumOf(amounts []money.Amount) *money.Amount {
	if len(amounts) == 0 {
		return nil
	}
	total := money.Sum(amounts...)
	return &total
}

// Get returns the plan id with its calculated figures, goals and completeness (FR-011 – FR-017).
func (s *Service) Get(ctx context.Context, a actor.Actor, id string) (PlanView, error) {
	var view PlanView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		p, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.view(ctx, a, p)
		return err
	})
	if err != nil {
		return PlanView{}, err
	}
	return view, nil
}

// Delete moves the plan id, with everything in it (the kinds of 005–007 registered below "plan"), to the
// recycle bin as one entry, in one Write; history logs one "deleted" entry naming what it held (FR-029).
// It returns the recycle-bin entry, so the person can undo the delete (FR-030).
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if _, err := s.stored(ctx, id); err != nil {
			return err
		}
		var err error
		entry, err = s.history.MoveToBin(ctx, a, kindName, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// stored reads the plan id inside the caller's transaction; a plan that does not exist is not_found.
func (s *Service) stored(ctx context.Context, id string) (Plan, error) {
	p, found, err := s.repo.Get(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	if !found {
		return Plan{}, errs.NotFound(planGoneMessage)
	}
	return p, nil
}

// Lookup returns the stored plan id inside the caller's transaction, and false when there is none: for the
// levels below the plan (005–007), which check their parent with it.
func (s *Service) Lookup(ctx context.Context, a actor.Actor, id string) (Plan, bool, error) {
	var p Plan
	var found bool
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		p, found, err = s.repo.Get(ctx, id)
		return err
	})
	return p, found, err
}

// References lists what every plan uses, the stored ones and those in the recycle bin, inside the
// caller's transaction: for the "Used by" counts of Settings (002) and of people and teams (003).
func (s *Service) References(ctx context.Context, a actor.Actor) ([]References, error) {
	var refs []References
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		refs = make([]References, 0, len(stored))
		for _, p := range stored {
			refs = append(refs, referencesOf(p, false))
		}
		bin, err := s.history.ListBin(authz.Unrestricted(ctx), a) // every plan in the bin, whoever asks (014)
		if err != nil {
			return err
		}
		for _, entry := range bin {
			for _, snapshot := range entry.Snapshots {
				if snapshot.Kind != kindName {
					continue
				}
				r, err := ReferencesFromSnapshot(snapshot.Data)
				if err != nil {
					return err
				}
				refs = append(refs, r)
			}
		}
		return nil
	})
	return refs, err
}

// ReferencesFromSnapshot returns what the plan of a recycle-bin snapshot uses (the snapshot's Data).
func ReferencesFromSnapshot(data json.RawMessage) (References, error) {
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return References{}, err
	}
	return referencesOf(p, true), nil
}

// referencesOf returns what the plan p uses; inBin says whether it is in the recycle bin.
func referencesOf(p Plan, inBin bool) References {
	p = p.Normalised() // a snapshot of before 016 has no currency
	refs := References{
		PlanID: p.ID, Name: p.Name, ModelID: p.ModelID, OwnerID: p.OwnerID, DefaultFinanceBudgetID: p.DefaultFinanceBudgetID,
		TeamIDs: slices.Clone(p.TeamIDs), Attributes: maps.Clone(p.Attributes), InRecycleBin: inBin, Currency: p.Currency,
	}
	if p.Budget != nil {
		budget := *p.Budget
		refs.Budget = &budget
	}
	return refs
}

// converter returns the converter of a's display currency (FR-015, D-079).
func (s *Service) converter(ctx context.Context, a actor.Actor) (converter, error) {
	rate, err := s.directory.DisplayRateOf(ctx, a.ID)
	if err != nil {
		return converter{}, err
	}
	return converterOf(rate), nil
}

// lookups are the people, teams and team members the views of some plans refer to, read once for all of
// them.
type lookups struct {
	people  map[string]people.PersonRef
	teams   map[string]people.TeamRef
	members map[string][]string
}

// lookupsFor reads the owners, teams and team members of plans.
func (s *Service) lookupsFor(ctx context.Context, plans []Plan) (lookups, error) {
	var ownerIDs, teamIDs []string
	for _, p := range plans {
		if p.OwnerID != "" && !slices.Contains(ownerIDs, p.OwnerID) {
			ownerIDs = append(ownerIDs, p.OwnerID)
		}
		for _, id := range p.TeamIDs {
			if !slices.Contains(teamIDs, id) {
				teamIDs = append(teamIDs, id)
			}
		}
	}
	var l lookups
	var err error
	if l.people, err = s.directory.PersonRefs(ctx, ownerIDs); err != nil {
		return lookups{}, err
	}
	if l.teams, err = s.directory.TeamRefs(ctx, teamIDs); err != nil {
		return lookups{}, err
	}
	if l.members, err = s.directory.MemberIDs(ctx, teamIDs); err != nil {
		return lookups{}, err
	}
	return l, nil
}

// shown is what the Plans list and the plan page both show of a plan.
type shown struct {
	owner        *people.PersonRef
	ownerProblem string
	teams        []people.TeamRef
	businessUnit []OptionRef
	figures      Figures
	completeness Completeness
}

// shownOf works out what the list and the plan page show of p, from what the levels below report (f).
func (s *Service) shownOf(ctx context.Context, p Plan, f Facts, l lookups, c converter) (shown, error) {
	sh := shown{teams: []people.TeamRef{}, figures: figuresOf(p, f, c)}
	for _, id := range p.TeamIDs {
		team, found := l.teams[id]
		if !found { // a team that no longer resolves is shown by its id (plan "API types")
			team = people.TeamRef{ID: id, Name: id, Colour: fallbackColour}
		}
		sh.teams = append(sh.teams, team)
	}
	sh.teams = sortedTeams(sh.teams)
	ownerOK := false
	if p.OwnerID != "" {
		owner, found := l.people[p.OwnerID]
		if !found {
			owner = people.PersonRef{ID: p.OwnerID, Name: p.OwnerID}
		}
		sh.owner = &owner
		if ownerOK = ownerEligible(p.OwnerID, p.TeamIDs, l.members); !ownerOK {
			sh.ownerProblem = fmt.Sprintf(msgOwnerNotInTeams, owner.Name) // R1, FR-021
		}
	}
	unitIDs, err := s.settings.BusinessUnit(ctx, p.Attributes)
	if err != nil {
		return shown{}, err
	}
	if sh.businessUnit, err = s.optionRefs(ctx, unitIDs); err != nil {
		return shown{}, err
	}
	attributes, err := s.settings.MissingAttributes(ctx, settings.LevelPlan, p.Attributes, unitIDs)
	if err != nil {
		return shown{}, err
	}
	sh.completeness = Completeness{Missing: ownMissing(p, ownerOK, attributes), ItemsBelow: itemsBelow(f)}
	return sh, nil
}

// optionRefs returns the Shell attribute options with the ids, in that order; an option that no longer
// resolves is shown by its id, inactive. Never nil.
func (s *Service) optionRefs(ctx context.Context, optionIDs []string) ([]OptionRef, error) {
	refs := make([]OptionRef, 0, len(optionIDs))
	for _, id := range optionIDs {
		o, found, err := s.settings.Option(ctx, id)
		if err != nil {
			return nil, err
		}
		if !found {
			refs = append(refs, OptionRef{ID: id, Label: id})
			continue
		}
		refs = append(refs, OptionRef{ID: o.ID, Label: o.Label, Active: o.Active})
	}
	return refs, nil
}

// sortedTeams returns teams sorted by name ignoring letter case, then by id.
func sortedTeams(teams []people.TeamRef) []people.TeamRef {
	slices.SortFunc(teams, func(x, y people.TeamRef) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
	})
	return teams
}

// view returns p as the API returns it, inside the caller's transaction: its references by name, its
// figures, goals and completeness in a's display currency, its figures in the plan currency too (016
// FR-013), and its contents, from one call per level below (FR-011 – FR-017, SC-009).
func (s *Service) view(ctx context.Context, a actor.Actor, p Plan) (PlanView, error) {
	facts, err := s.facts(ctx, []string{p.ID})
	if err != nil {
		return PlanView{}, err
	}
	f := *facts[p.ID]
	c, err := s.converter(ctx, a)
	if err != nil {
		return PlanView{}, err
	}
	rates, err := s.settings.Rates(ctx)
	if err != nil {
		return PlanView{}, err
	}
	l, err := s.lookupsFor(ctx, []Plan{p})
	if err != nil {
		return PlanView{}, err
	}
	sh, err := s.shownOf(ctx, p, f, l, c.reading(p.Currency, rates))
	if err != nil {
		return PlanView{}, err
	}
	m, found, err := s.settings.Model(ctx, p.ModelID)
	if err != nil {
		return PlanView{}, err
	}
	if !found { // a model that no longer resolves is shown by its id, inactive, without rates
		m = settings.Model{ID: p.ModelID, Name: p.ModelID, AverageDealSize: money.USD(money.Amount{})}
	}
	view := PlanView{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: p.Status,
		Model: ModelSummary{
			ID: m.ID, Name: m.Name, Active: m.Active, InqToMqlRate: m.InqToMqlRate, MqlToDealRate: m.MqlToDealRate,
			AverageDealSize: money.USD(m.AverageDealSize.Amount),
		},
		Teams: sh.teams, Owner: sh.owner, OwnerProblem: sh.ownerProblem, AOP: p.AOP, Description: p.Description,
		DefaultSpread: p.DefaultSpread, BCPIR: p.BCPIR, GoalStage: p.GoalStage, Attributes: p.Attributes,
		BusinessUnit: sh.businessUnit, Figures: sh.figures, Goals: goalsOf(p, m, f.ActivityGoals, c),
		Completeness: sh.completeness, Contents: contentsOf(f), Meta: p.Meta,
		Access:   authz.PlanAccessOf(ctx, AccessFacts(p)), // 014 FR-011
		Currency: p.Currency, PlanType: p.PlanType, BudgetPhasing: rollup.ViewOf(p.BudgetPhasing, p.Budget, p.Year, p.Currency),
		PlanFigures: figuresOf(p, f, identity(p.Currency)), CurrencyLock: currencyLock(f),
	}
	if view.Attributes == nil {
		view.Attributes = settings.AttributeValues{}
	}
	if p.Budget != nil {
		budget := money.Money{Amount: *p.Budget, Currency: p.Currency}
		view.Budget = &budget
	}
	if p.GoalAmount != "" {
		goal := p.GoalAmount
		view.GoalAmount = &goal
	}
	if p.DefaultFinanceBudgetID != "" {
		b, found, err := s.settings.FinanceBudget(ctx, p.DefaultFinanceBudgetID)
		if err != nil {
			return PlanView{}, err
		}
		if !found {
			b = settings.FinanceBudget{ID: p.DefaultFinanceBudgetID, Year: p.Year, Name: p.DefaultFinanceBudgetID}
		}
		view.DefaultFinanceBudget = &FinanceBudgetRef{ID: b.ID, Year: b.Year, Name: b.Name}
	}
	if view.Allocation, err = s.allocationView(ctx, p); err != nil { // where the budget comes from (014 FR-031)
		return PlanView{}, err
	}
	return view, nil
}
