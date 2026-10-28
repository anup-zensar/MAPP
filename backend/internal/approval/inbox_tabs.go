package approval

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Inbox returns the rows of one tab of the Approvals page — waiting, open or decided — over the records a sees.
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
			rows, err = s.waitingFor(ctx, b, a)
		case TabOpen:
			rows = b.open()
		default:
			rows = b.decided(a)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// submitted calls fn for every record in approval with its open round, seen as the round's record type.
func (b *board) submitted(fn func(sub subject, round Round, entries []Action) error) error {
	for _, sub := range b.subjects {
		round, found := openRound(b.rounds[sub.ID])
		if !found || !sub.waiting() {
			continue
		}
		if err := fn(sub.as(round.kind()), round, entriesOf(b.entries[sub.ID], round.ID)); err != nil {
			return err
		}
	}
	return nil
}

func (b *board) open() []Row {
	rows := []Row{}
	_ = b.submitted(func(sub subject, round Round, _ []Action) error {
		rows = append(rows, b.row(sub, round))
		return nil
	})
	sortBySubmission(rows)
	return rows
}

func sortBySubmission(rows []Row) {
	slices.SortStableFunc(rows, func(x, y Row) int { return x.SubmittedAt.Compare(y.SubmittedAt) })
}

// decided returns a row for every record on which a approved or declined a level, with a's latest decision
// and the record's latest round, latest decision first.
func (b *board) decided(a actor.Actor) []Row {
	rows := []Row{}
	for _, sub := range b.subjects {
		rounds := b.rounds[sub.ID]
		var mine *Action
		for _, e := range b.entries[sub.ID] {
			if e.Actor.ID == a.ID && (e.Action == ActionApproved || e.Action == ActionDeclined) {
				mine = &e
			}
		}
		if mine == nil || len(rounds) == 0 {
			continue
		}
		latest := rounds[len(rounds)-1]
		row := b.row(sub.as(latest.kind()), latest)
		row.MyDecision = &MyDecision{Action: string(mine.Action), At: mine.At, RoundNumber: mine.RoundNumber, Level: mine.Level}
		rows = append(rows, row)
	}
	slices.SortStableFunc(rows, func(x, y Row) int { return y.MyDecision.At.Compare(x.MyDecision.At) })
	return rows
}
