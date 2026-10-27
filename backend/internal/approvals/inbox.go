package approvals

import (
	"context"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// Summary returns the four counts of the Approvals page and the sidebar badge (FR-027, FR-028, SC-007):
// the rows of the tabs Waiting for me and All open, and the rounds of stored activities that ended approved
// or declined in [monthStart, monthEnd) — the viewer's calendar month, or the clock's UTC month when
// neither bound is given. Give both bounds or neither.
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
		for _, facts := range b.activities {
			for _, r := range b.rounds[facts.ID] {
				if r.EndedAt == nil || r.EndedAt.Before(from) || !r.EndedAt.Before(until) {
					continue
				}
				switch r.Outcome {
				case OutcomeApproved:
					counts.ApprovedThisMonth++
				case OutcomeDeclined:
					counts.DeclinedThisMonth++
				}
			}
		}
		return nil
	})
	if err != nil {
		return Counts{}, err
	}
	return counts, nil
}

// Preview returns what the preview panel of the Approvals page shows of one activity (FR-032): its facts
// and completeness, its line items, its unassigned planned cost and its Approval section. Its amounts in
// USD are answered in a's display currency (D-213).
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
		approval, err := s.approval(ctx, a, facts)
		if err != nil {
			return err
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		missing := check.Missing
		if missing == nil {
			missing = []MissingFieldRef{}
		}
		shown, err := s.displayOf(ctx, a)
		if err != nil {
			return err
		}
		items = slices.Clone(items) // converted below: never change the list Activities gave
		if items == nil {
			items = []PreviewLineItem{}
		}
		for i := range items {
			items[i].PlannedCostDisplay = shown.optional(items[i].PlannedCostDisplay)
		}
		preview = Preview{
			Activity: PreviewActivity{
				ID: facts.ID, Name: facts.Name, Status: facts.Status, DisplayStatus: facts.DisplayStatus, Missing: missing,
				Path: nonNilPath(facts.Path), Owner: d.owner(facts.OwnerID), StartDate: facts.StartDate, EndDate: facts.EndDate,
				Ongoing: facts.Ongoing, PlannedCost: shown.optional(facts.PlannedCost), TypeName: facts.TypeName,
				Description: facts.Description,
			},
			LineItems: items, UnassignedPlannedCost: shown.optional(facts.UnassignedPlannedCost), Approval: approval,
		}
		return nil
	})
	if err != nil {
		return Preview{}, err
	}
	return preview, nil
}

// board is what the Approvals page is built from, read once per request: the stored activities with their
// rounds and history entries, the workflows, the people with whether each can approve, the delegations
// current today, the time and today's date.
type board struct {
	activities  []ActivityFacts
	rounds      map[string][]Round  // by activity id, by number
	entries     map[string][]Action // by activity id, oldest first
	workflows   map[string]Workflow
	people      directory
	canApprove  map[string]bool // 014: holds approvals.decide
	delegations []Delegation    // current today (FR-023)
	now         time.Time
	today       date.Date
}

// board reads the board inside the caller's transaction.
func (s *Service) board(ctx context.Context) (*board, error) {
	rounds, err := s.repo.ListRounds(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.activitiesWithRounds(ctx, rounds)
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
		activities: list, rounds: map[string][]Round{}, entries: map[string][]Action{}, workflows: map[string]Workflow{},
		people: rc.people, canApprove: rc.canApprove, delegations: dl, now: now, today: date.Today(now),
	}
	for _, r := range rounds {
		b.rounds[r.ActivityID] = append(b.rounds[r.ActivityID], r)
	}
	for id := range b.rounds {
		slices.SortFunc(b.rounds[id], func(x, y Round) int { return x.Number - y.Number })
	}
	for _, e := range entries {
		b.entries[e.ActivityID] = append(b.entries[e.ActivityID], e)
	}
	for _, w := range workflows {
		b.workflows[w.ID] = w
	}
	return b, nil
}

// activitiesWithRounds reads the activities of rounds in the scope: every tab and count is over them alone.
func (s *Service) activitiesWithRounds(ctx context.Context, rounds []Round) ([]ActivityFacts, error) {
	return s.activities.ListInScope(ctx, authz.ScopeOf(ctx), activitiesOf(rounds))
}

// activitiesOf returns the ids of the activities of rounds, each once, in the order of their ids.
func activitiesOf(rounds []Round) []string {
	seen := map[string]bool{}
	ids := []string{}
	for _, r := range rounds {
		if !seen[r.ActivityID] {
			seen[r.ActivityID] = true
			ids = append(ids, r.ActivityID)
		}
	}
	return inIDOrder(ids)
}

// waitingFor returns the rows of a's Waiting for me (017 FR-031): the Submitted activities a may decide as a
// waiting approver, as the current delegate of one — marked with the approvers a acts for — or as the
// backup approver of an escalated level, and not barred by segregation of duties; oldest submission first.
func (s *Service) waitingFor(ctx context.Context, b *board, a actor.Actor) ([]Row, error) {
	rule, err := s.accessRule(ctx)
	if err != nil {
		return nil, err
	}
	dc := decider{person: a, canApprove: b.canApprove[a.ID], rule: rule, delegations: b.delegations}
	rows := []Row{}
	err = b.submitted(func(facts ActivityFacts, round Round, entries []Action) error {
		au, err := s.authority(ctx, dc, facts, round, entries, false)
		if err != nil || !au.waits() {
			return err
		}
		row := b.row(facts, round)
		row.AsDelegateOf = b.people.copiedAll(au.asDelegateOf)
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortBySubmission(rows)
	return rows, nil
}

// row returns the row of facts at round (FR-030; 017 FR-031): the level it waits at (or ended at), with the
// level's approvers, its mode and progress; since when it waits there and for how many whole days, whether
// that is overdue, escalated or without anyone who can act; and whether its start date has passed while it
// waits (edge case "start date passed").
func (b *board) row(facts ActivityFacts, round Round) Row {
	level := round.Levels[round.CurrentLevel-1]
	entries := entriesOf(b.entries[facts.ID], round.ID)
	workflow := WorkflowRef{ID: round.WorkflowID, Name: round.WorkflowName}
	if w, found := b.workflows[round.WorkflowID]; found {
		workflow.Name, workflow.Active = w.Name, w.Active
	}
	row := Row{
		Activity: RowActivity{ID: facts.ID, Name: facts.Name, Status: facts.Status, DisplayStatus: facts.DisplayStatus},
		Path:     nonNilPath(facts.Path), Owner: b.people.owner(facts.OwnerID), PlannedCost: facts.PlannedCost, Workflow: workflow,
		Round: RowRound{
			ID: round.ID, Number: round.Number, Outcome: round.Outcome, CurrentLevel: round.CurrentLevel,
			LevelCount: len(round.Levels), LevelLabel: levelLabel(level.Name, level.Number), Approvers: b.people.copiedAll(level.Approvers),
			Mode: level.Mode, RequiredCount: 1,
		},
		SubmittedAt: round.SubmittedAt, StartDate: facts.StartDate, AsDelegateOf: []PersonView{},
		StartDatePassed: facts.Status == StatusSubmitted && facts.StartDate != nil && facts.StartDate.Before(b.today),
	}
	waiting := waitingOn(level, entries)
	if level.Mode == ModeAll {
		row.Round.RequiredCount = len(level.Approvers)
		row.Round.ApprovedCount = len(level.Approvers) - len(waiting)
	} else if levelDone(level, entries) {
		row.Round.ApprovedCount = 1
	}
	if round.Outcome != OutcomeOpen {
		return row
	}
	row.WaitingSince = waitingSince(round, level, entries)
	if row.WaitingSince != nil {
		days := waitedDays(*row.WaitingSince, b.now)
		row.WaitingDays = &days
		row.Overdue = overdue(round, days)
	}
	if level.EscalatedAt != nil && level.Backup != nil {
		backup := b.people.copied(*level.Backup)
		row.EscalatedTo = &backup
	}
	row.NoApprover = level.NoApprover || (len(level.Approvers) > 0 && !slices.ContainsFunc(waiting, func(a actor.Actor) bool {
		return b.people.byID[a.ID].Active && b.canApprove[a.ID]
	}))
	return row
}

// waitedDays is how many whole days of 24 hours have passed from since to now (017 FR-029).
func waitedDays(since, now time.Time) int {
	if now.Before(since) {
		return 0
	}
	return int(now.Sub(since) / (24 * time.Hour))
}

// overdue reports whether a level of round that has waited days is overdue (017 FR-029, D-287): it has
// waited at least the round's escalation days or, without escalation, its reminder days; never without both.
func overdue(round Round, days int) bool {
	switch {
	case round.EscalationDays != nil:
		return days >= *round.EscalationDays
	case round.ReminderDays != nil:
		return days >= *round.ReminderDays
	}
	return false
}

// owner returns the owner personID as the API shows them, nil when the activity has none.
func (d directory) owner(personID string) *PersonView {
	if personID == "" {
		return nil
	}
	p := d.person(personID)
	return &p
}

// nonNilPath returns path, or an empty list for nil: the API's lists are never null.
func nonNilPath(path []errs.RecordRef) []errs.RecordRef {
	if path == nil {
		return []errs.RecordRef{}
	}
	return path
}
