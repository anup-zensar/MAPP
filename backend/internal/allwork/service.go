package allwork

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
)

// Messages of the plan's "User-visible messages" table.
const (
	msgRecordInBin   = "This %s was deleted — it is in the recycle bin"
	msgRecordGone    = "This %s no longer exists — someone may have deleted it."
	msgSomethingGone = "This record no longer exists — someone may have deleted it."
)

// Service holds the rules of All work (feature 010). Every method takes the actor and runs in one
// transaction. Query, Options, Export and Locate only read: All work never creates, changes or deletes a
// business record (FR-002).
type Service struct {
	tx   store.Transactor
	repo Repository
	src  Source
	ids  *ids.Generator
	now  clock.Clock
}

// NewService returns the All work service. It reads the planning records through src, keeps saved views and
// preferences in repo, gives saved views readable IDs with generator, works in transactions of tx and reads
// the time from now.
func NewService(tx store.Transactor, repo Repository, src Source, generator *ids.Generator, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, src: src, ids: generator, now: now}
}

// Query returns everything one view shows for q (FR-001 – FR-017, FR-025, FR-026), read in one transaction:
// the year is q's or the current year (FR-004); money is in a's display currency (FR-017); the answer says
// when it was read ("Updated at", edge case).
func (s *Service) Query(ctx context.Context, a actor.Actor, q Query) (Result, error) {
	var result Result
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		in, err := s.input(ctx, a, q)
		if err != nil {
			return err
		}
		result = build(in).result
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

// input reads what the view of q is worked out from, inside the caller's transaction.
func (s *Service) input(ctx context.Context, a actor.Actor, q Query) (input, error) {
	now := s.now()
	today := date.Today(now)
	year := q.Year
	if year == 0 {
		year = today.Year()
	}
	if q.YearFrom != 0 {
		year = q.YearFrom
	}
	yearTo := max(q.YearTo, year)
	currency, err := s.src.Currency(ctx, a.ID)
	if err != nil {
		return input{}, err
	}
	data, err := s.rangeData(ctx, year, yearTo, today, shownColumns(q))
	if err != nil {
		return input{}, err
	}
	choices, err := s.src.Choices(ctx, a.ID)
	if err != nil {
		return input{}, err
	}
	attributes, err := s.src.Attributes(ctx)
	if err != nil {
		return input{}, err
	}
	return input{
		query: q, year: year, yearFrom: year, yearTo: yearTo, today: today, now: now, currency: currency, data: data,
		choices: choices, catalogue: forReader(catalogue(attributes), choices.SeesFinance), attributes: attributes,
	}, nil
}

// rangeData reads the nodes of every year from from to to (016 FR-034): Source.Year once per year, each node
// knowing its plan's year.
func (s *Service) rangeData(ctx context.Context, from, to int, today date.Date, columns []string) (YearData, error) {
	all := YearData{Nodes: []Node{}}
	for year := from; year <= to; year++ {
		data, err := s.src.Year(ctx, year, today, columns)
		if err != nil {
			return YearData{}, err
		}
		for i := range data.Nodes {
			if data.Nodes[i].PlanYear == 0 {
				data.Nodes[i].PlanYear = year
			}
		}
		if from == to { // one year: its nodes as they are, without copying them
			return data, nil
		}
		all.Nodes = append(all.Nodes, data.Nodes...)
	}
	return all, nil
}

// shownColumns returns the column keys the view of q shows or sorts by: its columns (the default ones when
// it names none) and its sort column.
func shownColumns(q Query) []string {
	keys := q.Columns
	if keys == nil {
		keys = DefaultColumns
	}
	keys = slices.Clone(keys)
	if q.Sort != nil {
		keys = append(keys, q.Sort.Column)
	}
	return keys
}

// catalogue returns every column of the grid, the Shell attributes of Settings included (FR-018), inside
// the caller's transaction.
func (s *Service) catalogue(ctx context.Context) ([]Column, error) {
	attributes, err := s.src.Attributes(ctx)
	if err != nil {
		return nil, err
	}
	return catalogue(attributes), nil
}

// Options returns the choices of the filter bar for year (0: the current year) and the column catalogue
// (FR-004, FR-018): years, plans of the year, owners, teams with a's own first, business units, activity
// types, the column groups and the default columns.
func (s *Service) Options(ctx context.Context, a actor.Actor, year int, yearTo ...int) (Options, error) {
	today := date.Today(s.now())
	if year == 0 {
		year = today.Year()
	}
	to := year
	if len(yearTo) > 0 && yearTo[0] > year {
		to = yearTo[0]
	}
	var options Options
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		years, err := s.src.Years(ctx)
		if err != nil {
			return err
		}
		data, err := s.rangeData(ctx, year, to, today, nil)
		if err != nil {
			return err
		}
		choices, err := s.src.Choices(ctx, a.ID)
		if err != nil {
			return err
		}
		attributes, err := s.src.Attributes(ctx)
		if err != nil {
			return err
		}
		columns := catalogue(attributes)
		prefs, err := s.preferences(ctx, a.ID)
		if err != nil {
			return err
		}
		named := slices.Clone(prefs.Columns)
		if prefs.Sort != nil {
			named = append(named, prefs.Sort.Column)
		}
		views, err := s.repo.ListViews(ctx, a.ID)
		if err != nil {
			return err
		}
		for _, v := range views {
			named = append(named, v.State.Columns...)
			if v.State.Sort != nil {
				named = append(named, v.State.Sort.Column)
			}
		}
		options = optionsOf(year, today.Year(), years, data, choices, offered(forReader(columns, choices.SeesFinance), named))
		options.YearTo = to
		options.Attributes = filterableAttributes(attributes)
		return nil
	})
	if err != nil {
		return Options{}, err
	}
	return options, nil
}

// Locate returns the record a row opens, when it still exists (FR-016, D-123). A record that is gone is
// not_found, with "This <level> was deleted — it is in the recycle bin" while a recycle-bin entry holds it,
// and "This <level> no longer exists — someone may have deleted it." otherwise (edge case).
func (s *Service) Locate(ctx context.Context, a actor.Actor, recordID string) (RecordLocation, error) {
	var location RecordLocation
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		found, known, err := s.src.Locate(ctx, recordID)
		if err != nil {
			return err
		}
		if known {
			location = RecordLocation{
				RecordID: recordID, Level: found.Level, Name: found.Name, PlanID: found.PlanID, ActivityID: found.ActivityID,
			}
			return nil
		}
		word, levelKnown := recordWord(recordID)
		if !levelKnown {
			return errs.NotFound(msgSomethingGone)
		}
		inBin, err := s.src.InRecycleBin(ctx, recordID)
		if err != nil {
			return err
		}
		if inBin {
			return errs.NotFound(fmt.Sprintf(msgRecordInBin, word))
		}
		return errs.NotFound(fmt.Sprintf(msgRecordGone, word))
	})
	if err != nil {
		return RecordLocation{}, err
	}
	return location, nil
}

// recordWords name the levels by their readable-ID prefixes, mid-sentence.
var recordWords = map[string]string{
	ids.PrefixPlan:       "plan",
	ids.PrefixCampaign:   "campaign",
	ids.PrefixInitiative: "initiative",
	ids.PrefixActivity:   "activity",
	ids.PrefixLineItem:   "line item",
}

// recordWord returns the level of a record ID, mid-sentence, and false for an ID of no planning level.
func recordWord(recordID string) (string, bool) {
	prefix, _, found := strings.Cut(recordID, "-")
	if !found {
		return "", false
	}
	word, known := recordWords[prefix]
	return word, known
}
