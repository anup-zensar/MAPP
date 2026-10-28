package approval

import (
	"context"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
)

// Summary returns the counts of the Approvals page and the sidebar badge: the
// rows of Waiting for me and All open, and the rounds that ended approved or declined in [monthStart,
// monthEnd) — the clock's UTC month when neither is given.
func (s *Service) Summary(ctx context.Context, a actor.Actor, monthStart, monthEnd *time.Time) (Counts, error) {
	from, until, err := s.month(monthStart, monthEnd)
	if err != nil {
		return Counts{}, err
	}
	var counts Counts
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		b, err := s.board(ctx)
		if err != nil {
			return err
		}
		waiting, err := s.waitingFor(ctx, b, a)
		if err != nil {
			return err
		}
		counts.WaitingForMe = len(waiting)
		counts.AllOpen = len(b.open())
		for _, r := range b.endedRounds(from, until) {
			switch r.Outcome { //nolint:exhaustive // only approvals and declines are counted
			case OutcomeApproved:
				counts.ApprovedThisMonth++
			case OutcomeDeclined:
				counts.DeclinedThisMonth++
			}
		}
		return nil
	})
	if err != nil {
		return Counts{}, err
	}
	return counts, nil
}

func (b *board) endedRounds(from, until time.Time) []Round {
	var ended []Round
	for _, sub := range b.subjects {
		for _, r := range b.rounds[sub.ID] {
			if r.EndedAt != nil && !r.EndedAt.Before(from) && r.EndedAt.Before(until) {
				ended = append(ended, r)
			}
		}
	}
	return ended
}

// Preview returns what the preview panel of the Approvals page shows of one activity, its amounts
// in a's display currency (D-213).
func (s *Service) Preview(ctx context.Context, a actor.Actor, activityID string) (Preview, error) {
	var preview Preview
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		check, err := s.activities.ReadyCheck(ctx, a, facts.ID)
		if err != nil {
			return err
		}
		items, err := s.activities.LineItems(ctx, a, facts.ID)
		if err != nil {
			return err
		}
		sec, err := s.section(ctx, a, activitySubject(facts))
		if err != nil {
			return err
		}
		shown, err := s.displayOf(ctx, a)
		if err != nil {
			return err
		}
		preview = Preview{
			Activity: previewActivity(facts, check, sec.Owner, shown), LineItems: shownItems(items, shown),
			UnassignedPlannedCost: shown.optional(facts.UnassignedPlannedCost), Approval: activityApproval(sec),
		}
		return nil
	})
	if err != nil {
		return Preview{}, err
	}
	return preview, nil
}

func previewActivity(facts ActivityFacts, check ReadyCheck, owner *PersonView, shown display) PreviewActivity {
	missing := check.Missing
	if missing == nil {
		missing = []MissingFieldRef{}
	}
	return PreviewActivity{
		ID: facts.ID, Name: facts.Name, Status: facts.Status, DisplayStatus: facts.DisplayStatus, Missing: missing,
		Path: nonNilPath(facts.Path), Owner: owner, StartDate: facts.StartDate, EndDate: facts.EndDate,
		Ongoing: facts.Ongoing, PlannedCost: shown.optional(facts.PlannedCost), TypeName: facts.TypeName,
		Description: facts.Description,
	}
}

// shownItems converts a copy of the line items' USD planned costs: the list Activities gave is never changed.
func shownItems(items []PreviewLineItem, shown display) []PreviewLineItem {
	items = slices.Clone(items)
	if items == nil {
		items = []PreviewLineItem{}
	}
	for i := range items {
		items[i].PlannedCostDisplay = shown.optional(items[i].PlannedCostDisplay)
	}
	return items
}

// board is what the Approvals page is built from, read once per request: the records with rounds that the
// person's scope sees, their rounds and history entries, the workflows, the people and today's delegations.
type board struct {
	subjects    []subject
	rounds      map[string][]Round
	entries     map[string][]Action
	workflows   map[string]Workflow
	people      directory
	canApprove  map[string]bool
	delegations []Delegation
	now         time.Time
	today       date.Date
}

func (s *Service) board(ctx context.Context) (*board, error) {
	rounds, err := s.repo.ListRounds(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.subjectsWithRounds(ctx, rounds)
	if err != nil {
		return nil, err
	}
	entries, err := s.repo.ListActions(ctx)
	if err != nil {
		return nil, err
	}
	workflows, err := s.repo.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	rc, err := s.ruleContext(ctx)
	if err != nil {
		return nil, err
	}
	dl, err := s.currentDelegations(ctx, rc.people)
	if err != nil {
		return nil, err
	}
	now := s.now()
	b := &board{
		subjects: list, rounds: map[string][]Round{}, entries: map[string][]Action{}, workflows: map[string]Workflow{},
		people: rc.people, canApprove: rc.canApprove, delegations: dl, now: now, today: date.Today(now),
	}
	b.index(rounds, entries, workflows)
	return b, nil
}

func (b *board) index(rounds []Round, entries []Action, workflows []Workflow) {
	for _, r := range rounds {
		b.rounds[r.SubjectID] = append(b.rounds[r.SubjectID], r)
	}
	for id := range b.rounds {
		slices.SortFunc(b.rounds[id], func(x, y Round) int { return x.Number - y.Number })
	}
	for _, e := range entries {
		b.entries[e.SubjectID] = append(b.entries[e.SubjectID], e)
	}
	for _, w := range workflows {
		b.workflows[w.ID] = w
	}
}
