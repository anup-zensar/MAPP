package approval

import (
	"context"
	"fmt"
	"time"
)

const secondsPerDay = 86400

// RoundSummary is an approval round as feature 036's dashboards count it; WaitingSince is set on an open round.
type RoundSummary struct {
	SubjectKind  SubjectKind
	SubjectID    string
	Outcome      Outcome
	AutoApproved bool
	SubmittedAt  time.Time
	EndedAt      *time.Time
	WaitingSince *time.Time
}

// RoundSummaries returns the rounds of the records subjectIDs only, which the caller may see (D-356).
func (s *Service) RoundSummaries(ctx context.Context, subjectIDs []string) ([]RoundSummary, error) {
	var list []RoundSummary
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		rounds, err := s.repo.RoundsOfSubjects(ctx, subjectIDs)
		if err != nil {
			return err
		}
		actions, err := s.repo.ActionsOfSubjects(ctx, subjectIDs)
		if err != nil {
			return err
		}
		entries := map[string][]Action{}
		for _, a := range actions {
			entries[a.SubjectID] = append(entries[a.SubjectID], a)
		}
		list = make([]RoundSummary, 0, len(rounds))
		for _, r := range rounds {
			list = append(list, summaryOf(r, entries[r.SubjectID]))
		}
		return nil
	})
	return list, err
}

func summaryOf(r Round, entries []Action) RoundSummary {
	summary := RoundSummary{SubjectKind: r.kind(), SubjectID: r.SubjectID, Outcome: r.Outcome, AutoApproved: r.AutoApproved,
		SubmittedAt: r.SubmittedAt, EndedAt: r.EndedAt}
	if r.Outcome == OutcomeOpen && r.CurrentLevel >= 1 && r.CurrentLevel <= len(r.Levels) {
		summary.WaitingSince = waitingSince(r, r.Levels[r.CurrentLevel-1], entries)
		if summary.WaitingSince == nil {
			summary.WaitingSince = &summary.SubmittedAt
		}
	}
	return summary
}

// AverageDecisionDays is seconds ÷ decided in days, half up to one decimal ("2.5"); nil when nothing was decided.
func AverageDecisionDays(decided, seconds int64) *string {
	if decided <= 0 {
		return nil
	}
	denominator := decided * secondsPerDay
	tenths := (20*seconds + denominator) / (2 * denominator)
	average := fmt.Sprintf("%d.%d", tenths/10, tenths%10)
	return &average
}
