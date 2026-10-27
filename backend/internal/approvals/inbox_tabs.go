package approvals

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Inbox returns the rows of one tab of the Approvals page as a sees it (FR-029, FR-030): waiting — the
// Submitted activities a may decide as approver, delegate or backup approver, not on behalf (017 FR-031);
// open — every Submitted activity; both oldest
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

// submitted calls fn for every Submitted activity with its open round and the round's history entries.
func (b *board) submitted(fn func(facts ActivityFacts, round Round, entries []Action) error) error {
	for _, facts := range b.activities {
		if facts.Status != StatusSubmitted {
			continue
		}
		round, found := openRound(b.rounds[facts.ID])
		if !found {
			continue
		}
		if err := fn(facts, round, entriesOf(b.entries[facts.ID], round.ID)); err != nil {
			return err
		}
	}
	return nil
}

// open returns a row for every Submitted activity, oldest submission first (FR-029, FR-030).
func (b *board) open() []Row {
	rows := []Row{}
	_ = b.submitted(func(facts ActivityFacts, round Round, _ []Action) error {
		rows = append(rows, b.row(facts, round))
		return nil
	})
	sortBySubmission(rows)
	return rows
}

// sortBySubmission orders rows by submission, oldest first.
func sortBySubmission(rows []Row) {
	slices.SortStableFunc(rows, func(x, y Row) int { return x.SubmittedAt.Compare(y.SubmittedAt) })
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
