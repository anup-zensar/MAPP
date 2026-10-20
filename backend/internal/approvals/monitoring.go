package approvals

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
)

// The ageing buckets in order, with their labels (data-model §11).
var ageingBuckets = []struct {
	bucket, label string
	upTo          int // the most whole days of the bucket; -1 for no limit
}{
	{BucketUnder1, "Under 1 day", 0},
	{BucketDays1to2, "1–2 days", 2},
	{BucketDays3to5, "3–5 days", 5},
	{BucketDays6to10, "6–10 days", 10},
	{BucketOver10, "More than 10 days", -1},
}

// Monitoring returns the Monitoring tab of the Approvals page (017 FR-029, FR-030, data-model §11), computed
// on read over the rows of All open (D-293): the overdue and escalated counts, the average time to decision
// and the approvals at submission of the calendar month [monthStart, monthEnd) (the clock's UTC month when
// neither is given; give both or neither), how long open approvals have waited in five buckets, the open
// approvals by workflow and level, and the overdue rows, oldest waiting first, with their amounts in a's
// display currency. canRunEscalations says whether a holds workflows.manage.
func (s *Service) Monitoring(ctx context.Context, a actor.Actor, monthStart, monthEnd *time.Time) (Monitoring, error) {
	from, until, err := s.month(monthStart, monthEnd)
	if err != nil {
		return Monitoring{}, err
	}
	var m Monitoring
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		b, err := s.board(ctx)
		if err != nil {
			return err
		}
		grant, err := s.access.Check(ctx, a.ID, PermissionManageWorkflows, nil)
		if err != nil {
			return err
		}
		shown, err := s.displayOf(ctx, a)
		if err != nil {
			return err
		}
		m = b.monitoring(from, until)
		m.CanRunEscalations = grant.Allowed
		for i := range m.Overdue {
			m.Overdue[i].PlannedCost = shown.optional(m.Overdue[i].PlannedCost)
		}
		return nil
	})
	if err != nil {
		return Monitoring{}, err
	}
	return m, nil
}

// month returns the bounds of the month the counts use: [monthStart, monthEnd), or the clock's UTC calendar
// month when neither is given (as Summary).
func (s *Service) month(monthStart, monthEnd *time.Time) (time.Time, time.Time, error) {
	switch {
	case (monthStart == nil) != (monthEnd == nil):
		return time.Time{}, time.Time{}, errs.Validation(errs.FieldError{Field: "monthStart", Message: reloadMessage})
	case monthStart != nil && !monthEnd.After(*monthStart):
		return time.Time{}, time.Time{}, errs.Validation(errs.FieldError{Field: "monthEnd", Message: reloadMessage})
	case monthStart != nil:
		return *monthStart, *monthEnd, nil
	}
	now := s.now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return from, from.AddDate(0, 1, 0), nil
}

// monitoring computes the Monitoring values of the board for the month [from, until).
func (b *board) monitoring(from, until time.Time) Monitoring {
	m := Monitoring{Ageing: make([]AgeingBucket, 0, len(ageingBuckets)), ByLevel: []WaitingByLevel{}, Overdue: []Row{}}
	for _, a := range ageingBuckets {
		m.Ageing = append(m.Ageing, AgeingBucket{Bucket: a.bucket, Label: a.label})
	}

	type groupKey struct {
		workflowID string
		level      int
	}
	groups := map[groupKey]*WaitingByLevel{}
	oldestName := map[groupKey]time.Time{} // when the round whose workflow name the group shows was submitted
	for _, row := range b.open() {
		if row.EscalatedTo != nil {
			m.Cards.Escalated++
		}
		if row.Overdue {
			m.Cards.Overdue++
			m.Overdue = append(m.Overdue, row)
		}
		if row.WaitingDays != nil {
			for i, a := range ageingBuckets {
				if a.upTo < 0 || *row.WaitingDays <= a.upTo {
					m.Ageing[i].Count++
					break
				}
			}
		}
		if row.WaitingSince == nil {
			continue
		}
		key := groupKey{row.Workflow.ID, row.Round.CurrentLevel}
		g, found := groups[key]
		if !found {
			g = &WaitingByLevel{Level: key.level, LevelLabel: row.Round.LevelLabel, OldestWaitingSince: *row.WaitingSince}
			groups[key] = g
		}
		g.Waiting++
		if row.Overdue {
			g.Overdue++
		}
		if row.WaitingSince.Before(g.OldestWaitingSince) {
			g.OldestWaitingSince = *row.WaitingSince
		}
		if submitted, named := oldestName[key]; !named || row.SubmittedAt.Before(submitted) {
			oldestName[key] = row.SubmittedAt
			g.Workflow = RoundWorkflow{ID: key.workflowID, Name: b.roundWorkflowName(row)}
			g.LevelLabel = row.Round.LevelLabel
		}
	}
	for _, g := range groups {
		m.ByLevel = append(m.ByLevel, *g)
	}
	slices.SortStableFunc(m.ByLevel, func(x, y WaitingByLevel) int {
		return cmp.Or(cmp.Compare(strings.ToLower(x.Workflow.Name), strings.ToLower(y.Workflow.Name)),
			cmp.Compare(x.Workflow.ID, y.Workflow.ID), cmp.Compare(x.Level, y.Level))
	})
	slices.SortStableFunc(m.Overdue, func(x, y Row) int { return x.WaitingSince.Compare(*y.WaitingSince) })

	var decided, seconds int64
	for _, facts := range b.activities {
		for _, r := range b.rounds[facts.ID] {
			if r.EndedAt == nil || r.EndedAt.Before(from) || !r.EndedAt.Before(until) {
				continue
			}
			switch {
			case r.AutoApproved:
				m.Cards.AutoApprovedThisMonth++
			case r.Outcome == OutcomeApproved || r.Outcome == OutcomeDeclined:
				decided++
				seconds += int64(r.EndedAt.Sub(r.SubmittedAt) / time.Second)
			}
		}
	}
	m.Cards.DecidedThisMonth = int(decided)
	if decided > 0 {
		// The mean in days, half up to one decimal, in whole numbers: tenths = round(10 × seconds / (n × 86,400)).
		denominator := decided * 86400
		tenths := (20*seconds + denominator) / (2 * denominator)
		average := fmt.Sprintf("%d.%d", tenths/10, tenths%10)
		m.Cards.AverageDecisionDays = &average
	}
	return m
}

// roundWorkflowName is the workflow name of a row's round at submission (data-model §11).
func (b *board) roundWorkflowName(row Row) string {
	for _, r := range b.rounds[row.Activity.ID] {
		if r.ID == row.Round.ID {
			return r.WorkflowName
		}
	}
	return row.Workflow.Name
}
