package plan

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
)

const (
	msgCopyIntoYear    = "Copy into %d or %d."
	msgYearsUpTo       = "Plans can only be created up to %d."
	noteBudgetCleared  = "Default finance budget cleared — it belonged to %d."
	copiedFromDetails  = "from '%s' (%s)" // + ", with 2 campaigns, 4 initiatives, …" when contents came along
	copySuffix         = " (copy)"
	numberedCopySuffix = " (copy %d)"
)

func (s *Service) CopyOptions(ctx context.Context, a actor.Actor, id string) (CopyOptions, error) {
	var options CopyOptions
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		source, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		all, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		facts, err := s.facts(ctx, []string{source.ID})
		if err != nil {
			return err
		}
		options = CopyOptions{
			Name:     copyName(source.Name, func(name string) bool { _, taken := planNamed(all, name, ""); return taken }),
			SameYear: source.Year,
			NextYear: s.nextYear(source.Year),
			Contents: contentsOf(*facts[source.ID]),
		}
		return nil
	})
	if err != nil {
		return CopyOptions{}, err
	}
	return options, nil
}

// Copy copies every field of the plan id (D-035) into a new Draft plan, all or nothing; a next-year copy clears the
// default finance budget of the plan's year, with a note. includeContents has every level below copy its records,
// top-down, moving dates by the years between (D-195); one "copied" entry on the new plan logs it (D-082).
func (s *Service) Copy(ctx context.Context, a actor.Actor, id string, in CopyRequest) (CopyResult, error) {
	var result CopyResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		source, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		// The copy has the source's teams and a as its creator: a needs "Create and copy plans" there.
		if err := authz.RequireIn(ctx, authz.PlansCreate, authz.PlanFacts{TeamIDs: source.TeamIDs, CreatedByID: a.ID}); err != nil {
			return err
		}
		var problems errs.Problems
		name := strings.TrimSpace(in.Name)
		if err := s.checkName(ctx, name, "", &problems); err != nil {
			return err
		}
		years := s.checkCopyYear(source.Year, in.Year, &problems)
		if err := problems.Err(); err != nil {
			return err
		}

		p := copyOfPlan(source)
		p.Name, p.Year, p.Status = name, source.Year+years, StatusDraft
		p.BudgetPhasing = rollup.ShiftPhasing(source.BudgetPhasing, years)
		notes := []string{}
		if years != 0 && p.DefaultFinanceBudgetID != "" {
			p.DefaultFinanceBudgetID = ""
			notes = append(notes, fmt.Sprintf(noteBudgetCleared, source.Year))
		}
		if p.ID, err = s.ids.Next(ctx, ids.PrefixPlan); err != nil {
			return err
		}
		p.Meta = record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, p); err != nil {
			return err
		}

		counts := map[string]int{}
		if in.IncludeContents {
			job := &CopyJob{
				Actor: a, Now: s.now(), SourcePlanID: source.ID, TargetPlanID: p.ID, Years: years,
				NewIDs: map[string]string{source.ID: p.ID}, Counts: counts,
			}
			for _, l := range s.levels {
				if err := l.Copy(ctx, job); err != nil {
					return err
				}
			}
		}
		details := fmt.Sprintf(copiedFromDetails, source.Name, source.ID)
		if with := s.history.CountsText(counts); with != "" {
			details += ", " + with
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: p.ID, RecordName: p.Name, Action: history.ActionCopied, Details: details,
		}); err != nil {
			return err
		}
		view, err := s.view(ctx, a, p)
		if err != nil {
			return err
		}
		result = CopyResult{Plan: view, Notes: notes}
		return nil
	})
	if err != nil {
		return CopyResult{}, err
	}
	return result, nil
}

func (s *Service) nextYear(year int) NextYearOption {
	_, last := s.years()
	option := NextYearOption{Year: year + 1, Available: year+1 <= last}
	if !option.Available {
		option.Reason = fmt.Sprintf(msgYearsUpTo, last)
	}
	return option
}

// checkCopyYear returns how many years the copy moves on: 0 for the same year, 1 for the next one.
func (s *Service) checkCopyYear(sourceYear int, year *int, problems *errs.Problems) int {
	switch {
	case year != nil && *year == sourceYear:
		return 0
	case year != nil && *year == sourceYear+1:
		if next := s.nextYear(sourceYear); !next.Available {
			problems.Add("year", next.Reason)
		}
		return 1
	default:
		problems.Add("year", fmt.Sprintf(msgCopyIntoYear, sourceYear, sourceYear+1))
		return 0
	}
}

// copyOfPlan returns a copy of p that shares no list, map, pointer or phasing month with it.
func copyOfPlan(p Plan) Plan {
	dup := p
	dup.TeamIDs = slices.Clone(p.TeamIDs)
	dup.Attributes = maps.Clone(p.Attributes)
	if p.Budget != nil {
		budget := *p.Budget
		dup.Budget = &budget
	}
	dup.BudgetPhasing = rollup.ShiftPhasing(p.BudgetPhasing, 0)
	return dup
}

// copyName suggests "<name> (copy)", then "(copy 2)", … — the first one not taken — shortening name to fit 120 characters.
func copyName(name string, taken func(string) bool) string {
	for n := 1; ; n++ {
		suffix := copySuffix
		if n > 1 {
			suffix = fmt.Sprintf(numberedCopySuffix, n)
		}
		base := name
		if room := maxNameLength - utf8.RuneCountInString(suffix); utf8.RuneCountInString(base) > room {
			base = strings.TrimRight(string([]rune(base)[:room]), " ")
		}
		if candidate := base + suffix; !taken(candidate) {
			return candidate
		}
	}
}
