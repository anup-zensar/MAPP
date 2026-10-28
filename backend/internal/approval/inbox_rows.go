package approval

import (
	"context"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// waitingFor returns the rows of a's Waiting for me: the records a may decide as a waiting
// approver, a current delegate of one or the backup approver of an escalated level; oldest submission first.
func (s *Service) waitingFor(ctx context.Context, b *board, a actor.Actor) ([]Row, error) {
	rule, err := s.accessRule(ctx)
	if err != nil {
		return nil, err
	}
	dc := decider{person: a, canApprove: b.canApprove[a.ID], rule: rule, delegations: b.delegations}
	rows := []Row{}
	err = b.submitted(func(sub subject, round Round, entries []Action) error {
		au, authErr := s.authority(ctx, dc, decisionPoint{sub: sub, round: round, entries: entries})
		if authErr != nil || !au.waits() {
			return authErr
		}
		row := b.row(sub, round)
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

// row returns the row of sub at round: the level it waits or ended at,
// how long it has waited, and whether that is overdue, escalated or without anyone who can act.
func (b *board) row(sub subject, round Round) Row {
	level := round.Levels[round.CurrentLevel-1]
	entries := entriesOf(b.entries[sub.ID], round.ID)
	workflow := WorkflowRef{ID: round.WorkflowID, Name: round.WorkflowName}
	if w, found := b.workflows[round.WorkflowID]; found {
		workflow.Name, workflow.Active = w.Name, w.Active
	}
	view := sub.view()
	view.Version = 0
	row := Row{
		Subject: view, Path: nonNilPath(sub.Path), Owner: b.people.owner(sub.OwnerID), PlannedCost: sub.Amount,
		Workflow: workflow, Round: rowRound(round, level, b.people), SubmittedAt: round.SubmittedAt, StartDate: sub.StartDate,
		AsDelegateOf:    []PersonView{},
		StartDatePassed: sub.Status == StatusSubmitted && sub.StartDate != nil && sub.StartDate.Before(b.today),
	}
	if sub.Kind == KindActivity {
		row.Activity = &RowActivity{ID: sub.ID, Name: sub.Name, Status: sub.Status, DisplayStatus: sub.DisplayStatus}
	}
	waiting := waitingOn(level, entries)
	if level.Mode == ModeAll {
		row.Round.RequiredCount = len(level.Approvers)
		row.Round.ApprovedCount = len(level.Approvers) - len(waiting)
	} else if levelDone(level, entries) {
		row.Round.ApprovedCount = 1
	}
	if round.Outcome == OutcomeOpen {
		b.waitingRow(&row, round, entries)
	}
	return row
}

func rowRound(round Round, level RoundLevel, d directory) RowRound {
	return RowRound{
		ID: round.ID, Number: round.Number, Outcome: round.Outcome, CurrentLevel: round.CurrentLevel,
		LevelCount: len(round.Levels), LevelLabel: levelLabel(level.Name, level.Number), Approvers: d.copiedAll(level.Approvers),
		Mode: level.Mode, RequiredCount: 1,
	}
}

func (b *board) waitingRow(row *Row, round Round, entries []Action) {
	level := round.Levels[round.CurrentLevel-1]
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
	canAct := slices.ContainsFunc(waitingOn(level, entries), func(a actor.Actor) bool {
		return b.people.byID[a.ID].Active && b.canApprove[a.ID]
	})
	row.NoApprover = level.NoApprover || (len(level.Approvers) > 0 && !canAct)
}

// waitedDays is how many whole days of 24 hours have passed from since to now.
func waitedDays(since, now time.Time) int {
	if now.Before(since) {
		return 0
	}
	return int(now.Sub(since) / (24 * time.Hour))
}

// overdue: the level has waited the round's escalation days or, without escalation, its reminder days (D-287).
func overdue(round Round, days int) bool {
	switch {
	case round.EscalationDays != nil:
		return days >= *round.EscalationDays
	case round.ReminderDays != nil:
		return days >= *round.ReminderDays
	}
	return false
}

func (d directory) owner(personID string) *PersonView {
	if personID == "" {
		return nil
	}
	p := d.person(personID)
	return &p
}

func nonNilPath(path []errs.RecordRef) []errs.RecordRef {
	if path == nil {
		return []errs.RecordRef{}
	}
	return path
}
