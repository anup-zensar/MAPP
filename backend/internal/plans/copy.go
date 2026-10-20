package plans

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
)

// The messages of a copy (plan "User-visible messages": API validation, copy notes, activity log).
const (
	msgCopyIntoYear    = "Copy into %d or %d."
	msgYearsUpTo       = "Plans can only be created up to %d."
	noteBudgetCleared  = "Default finance budget cleared — it belonged to %d."
	copiedFromDetails  = "from '%s' (%s)" // + ", with 2 campaigns, 4 initiatives, …" when contents came along
	copySuffix         = " (copy)"
	numberedCopySuffix = " (copy %d)"
)

// CopyOptions returns what the copy dialog offers for the plan id (FR-026, FR-028): the suggested name of
// the copy, the plan's own year, the next year — available only when it is a permitted plan year (D-084) —
// and the counts of what the copy can take with it.
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

// Copy copies the plan id as a new Draft plan (FR-026 – FR-028, R13), completely or not at all, in one
// Write. The name is checked as on create (required, at most 120 characters, unique); the year must be the
// plan's own or, when it is a permitted plan year, the next one — every problem is reported at once. The
// copy has every field of the plan (owner, teams, budget, goal, Shell attributes, AOP, BCPIR, spread:
// D-035), a new ID and the copier as its creator. A next-year copy clears the default finance budget, which
// belongs to the plan's year, and says so in a note. With includeContents, every level below copies its
// records in turn, top-down, with every date moved by the years between (D-195). One "copied" log entry on
// the new plan names the plan it came from and what came with it (D-082).
func (s *Service) Copy(ctx context.Context, a actor.Actor, id string, in CopyRequest) (CopyResult, error) {
	var result CopyResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		source, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		// The copy has the source's teams and a as its creator: a needs "Create and copy plans" there (014).
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
			for _, l := range s.levels { // top-down: each level copies below the records copied above it
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

// nextYear says whether a copy of a plan of year may go into the next year (FR-028, D-084): only when that
// year is a permitted plan year, at most the current year + 5; otherwise it says why.
func (s *Service) nextYear(year int) NextYearOption {
	_, last := s.years()
	option := NextYearOption{Year: year + 1, Available: year+1 <= last}
	if !option.Available {
		option.Reason = fmt.Sprintf(msgYearsUpTo, last)
	}
	return option
}

// checkCopyYear checks the year of a copy of a plan of sourceYear and returns how many years the copy moves
// on: 0 for the same year, 1 for the next one (FR-028). Any other year, or a next year beyond the permitted
// plan years, is a problem on "year".
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

// copyOfPlan returns a copy of the plan p that shares no list, map or pointer with it.
func copyOfPlan(p Plan) Plan {
	dup := p
	dup.TeamIDs = slices.Clone(p.TeamIDs)
	dup.Attributes = maps.Clone(p.Attributes)
	if p.Budget != nil {
		budget := *p.Budget
		dup.Budget = &budget
	}
	return dup
}

// copyName returns the name a copy of the plan called name suggests (R13, FR-026): "<name> (copy)", then
// "<name> (copy 2)", "<name> (copy 3)", … — the first that taken says no plan has. When the name and the
// suffix together pass 120 characters, the name is shortened, without trailing spaces, so the suffix fits
// (edge case).
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
