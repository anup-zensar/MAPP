package allwork

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

const (
	msgRecordInBin   = "This %s was deleted — it is in the recycle bin"
	msgRecordGone    = "This %s no longer exists — someone may have deleted it."
	msgSomethingGone = "This record no longer exists — someone may have deleted it."
)

// Service never creates, changes or deletes a business record: Query, Options, Export and Locate only read.
type Service struct {
	tx   Transactor
	repo Store
	src  Source
	ids  *ids.Generator
	now  clock.Clock
}

// NewService leaves the tools' log alone: All work writes no activity log.
func NewService(tools history.Tools, repo Store, src Source) *Service {
	return &Service{tx: tools.Tx, repo: repo, src: src, ids: tools.IDs, now: tools.Now}
}

// Query reads q's year, or the current year, in one transaction; money is in a's display currency.
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

// rangeData calls Source.Year once per year from from to to, each node knowing its plan's year.
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

// shownColumns are q's columns, the default ones when it names none, and its sort column.
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

// catalogue includes the Shell attributes of Settings.
func (s *Service) catalogue(ctx context.Context) ([]Column, error) {
	attributes, err := s.src.Attributes(ctx)
	if err != nil {
		return nil, err
	}
	return catalogue(attributes), nil
}

// Options are the filter bar's choices for year (0: the current year), with a's own teams first.
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

// Locate answers a record that is gone with not_found, saying whether the recycle bin holds it.
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

var recordWords = map[string]string{
	ids.PrefixPlan:       "plan",
	ids.PrefixCampaign:   "campaign",
	ids.PrefixInitiative: "initiative",
	ids.PrefixActivity:   "activity",
	ids.PrefixLineItem:   "line item",
}

// recordWord is false for an ID of no planning level.
func recordWord(recordID string) (string, bool) {
	prefix, _, found := strings.Cut(recordID, "-")
	if !found {
		return "", false
	}
	word, known := recordWords[prefix]
	return word, known
}
