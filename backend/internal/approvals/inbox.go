package approvals

import (
	"context"
	"slices"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
)

// Inbox returns the rows of one tab of the Approvals page as a sees it (FR-029, FR-030): waiting — the
// Submitted activities whose current level lists a; open — every Submitted activity; both oldest
// submission first; decided — the activities on which a approved or declined a level in any round, latest
// decision first, with that decision and the activity's latest round. Only stored activities are listed:
// one in the recycle bin is not (FR-023).
func (s *Service) Inbox(ctx context.Context, a actor.Actor, tab string) ([]Row, error) {
	if tab != TabWaiting && tab != TabOpen && tab != TabDecided {
		return nil, errs.Validation(errs.FieldError{Field: "tab", Message: chooseTabMessage})
	}
	var rows []Row
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		b, err := s.board(ctx)
		if err != nil {
			return err
		}
		switch tab {
		case TabWaiting:
			rows = b.open(func(r Round) bool { return includes(r.Levels[r.CurrentLevel-1].Approvers, a.ID) })
		case TabOpen:
			rows = b.open(func(Round) bool { return true })
		default:
			rows = b.decided(a)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// Summary returns the four counts of the Approvals page and the sidebar badge (FR-027, FR-028, SC-007):
// the rows of the tabs Waiting for me and All open, and the rounds of stored activities that ended approved
// or declined in [monthStart, monthEnd) — the viewer's calendar month, or the clock's UTC month when
// neither bound is given. Give both bounds or neither.
func (s *Service) Summary(ctx context.Context, a actor.Actor, monthStart, monthEnd *time.Time) (Counts, error) {
	switch {
	case (monthStart == nil) != (monthEnd == nil):
		return Counts{}, errs.Validation(errs.FieldError{Field: "monthStart", Message: reloadMessage})
	case monthStart != nil && !monthEnd.After(*monthStart):
		return Counts{}, errs.Validation(errs.FieldError{Field: "monthEnd", Message: reloadMessage})
	}
	var from, until time.Time
	if monthStart != nil {
		from, until = *monthStart, *monthEnd
	} else {
		now := s.now().UTC()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		until = from.AddDate(0, 1, 0)
	}

	var counts Counts
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		b, err := s.board(ctx)
		if err != nil {
			return err
		}
		counts.WaitingForMe = len(b.open(func(r Round) bool { return includes(r.Levels[r.CurrentLevel-1].Approvers, a.ID) }))
		counts.AllOpen = len(b.open(func(Round) bool { return true }))
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
// and completeness, its line items, its unassigned planned cost and its Approval section.
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
		if items == nil {
			items = []PreviewLineItem{}
		}
		preview = Preview{
			Activity: PreviewActivity{
				ID: facts.ID, Name: facts.Name, Status: facts.Status, DisplayStatus: facts.DisplayStatus, Missing: missing,
				Path: nonNilPath(facts.Path), Owner: d.owner(facts.OwnerID), StartDate: facts.StartDate, EndDate: facts.EndDate,
				Ongoing: facts.Ongoing, PlannedCost: facts.PlannedCost, TypeName: facts.TypeName, Description: facts.Description,
			},
			LineItems: items, UnassignedPlannedCost: facts.UnassignedPlannedCost, Approval: approval,
		}
		return nil
	})
	if err != nil {
		return Preview{}, err
	}
	return preview, nil
}

// board is what the Approvals page is built from, read once per request: the stored activities with their
// rounds and history entries, the workflows, the people and today's date.
type board struct {
	activities []ActivityFacts
	rounds     map[string][]Round  // by activity id, by number
	entries    map[string][]Action // by activity id, oldest first
	workflows  map[string]Workflow
	people     directory
	today      date.Date
}

// board reads the board inside the caller's transaction.
func (s *Service) board(ctx context.Context) (*board, error) {
	list, err := s.activities.List(ctx)
	if err != nil {
		return nil, err
	}
	rounds, err := s.repo.ListRounds(ctx)
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
	d, err := s.directory(ctx)
	if err != nil {
		return nil, err
	}
	b := &board{
		activities: list, rounds: map[string][]Round{}, entries: map[string][]Action{}, workflows: map[string]Workflow{},
		people: d, today: date.Today(s.now()),
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

// open returns a row for every Submitted activity whose open round keep accepts, oldest submission first
// (FR-029, FR-030).
func (b *board) open(keep func(Round) bool) []Row {
	rows := []Row{}
	for _, facts := range b.activities {
		if facts.Status != StatusSubmitted {
			continue
		}
		round, found := openRound(b.rounds[facts.ID])
		if !found || !keep(round) {
			continue
		}
		rows = append(rows, b.row(facts, round))
	}
	slices.SortStableFunc(rows, func(x, y Row) int { return x.SubmittedAt.Compare(y.SubmittedAt) })
	return rows
}

// decided returns a row for every activity on which a approved or declined a level, with a's latest
// decision and the activity's latest round, latest decision first (FR-029, FR-030).
func (b *board) decided(a actor.Actor) []Row {
	rows := []Row{}
	for _, facts := range b.activities {
		rounds := b.rounds[facts.ID]
		var mine *Action
		for _, e := range b.entries[facts.ID] {
			if e.Actor.ID == a.ID && (e.Action == ActionApproved || e.Action == ActionDeclined) {
				mine = &e // entries are oldest first: the last one is the latest
			}
		}
		if mine == nil || len(rounds) == 0 {
			continue
		}
		row := b.row(facts, rounds[len(rounds)-1])
		row.MyDecision = &MyDecision{Action: string(mine.Action), At: mine.At, RoundNumber: mine.RoundNumber, Level: mine.Level}
		rows = append(rows, row)
	}
	slices.SortStableFunc(rows, func(x, y Row) int { return y.MyDecision.At.Compare(x.MyDecision.At) })
	return rows
}

// row returns the row of facts at round (FR-030): the level it waits at (or ended at), with the level's
// designated approvers; since when it waits there; and whether its start date has passed while it waits
// (edge case "start date passed").
func (b *board) row(facts ActivityFacts, round Round) Row {
	level := round.Levels[round.CurrentLevel-1]
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
		},
		SubmittedAt: round.SubmittedAt, StartDate: facts.StartDate,
		StartDatePassed: facts.Status == StatusSubmitted && facts.StartDate != nil && facts.StartDate.Before(b.today),
	}
	if round.Outcome == OutcomeOpen {
		since := round.SubmittedAt
		if previous, found := decisionOn(entriesOf(b.entries[facts.ID], round.ID), round.CurrentLevel-1); found {
			since = previous.At
		}
		row.WaitingSince = &since
	}
	return row
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
