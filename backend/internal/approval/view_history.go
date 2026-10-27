package approval

import (
	"mapp/internal/platform/actor"
)

// entriesOf returns the history entries of the round roundID, oldest first; never nil.
func entriesOf(entries []Action, roundID string) []Action {
	return where(entries, func(e Action) bool { return e.RoundID == roundID })
}

// decisionOn returns the first decision (approved or declined) on level in a round's entries, and false
// when the level has none. Only the first decision of each approver counts (FR-018, 017 FR-020).
func decisionOn(entries []Action, level int) (Action, bool) {
	for _, e := range entries {
		if e.Level == level && (e.Action == ActionApproved || e.Action == ActionDeclined) {
			return e, true
		}
	}
	return Action{}, false
}

// decisionsOn returns every decision on level in a round's entries, oldest first; never nil.
func decisionsOn(entries []Action, level int) []Action {
	return where(entries, func(e Action) bool {
		return e.Level == level && (e.Action == ActionApproved || e.Action == ActionDeclined)
	})
}

// decisionView returns a decision entry as the chain shows it.
func decisionView(e Action) Decision {
	return Decision{
		Action: string(e.Action), Actor: e.Actor, OnBehalfOf: nonNil(e.OnBehalfOf), At: e.At, Comment: e.Comment,
		Capacity: e.Capacity, AsDelegateOf: e.AsDelegateOf,
	}
}

// historyEntryView returns one history entry as the API shows it, with the sentences of its changes.
func historyEntryView(e Action, d directory) HistoryEntryView {
	view := HistoryEntryView{
		ID: e.ID, RoundNumber: e.RoundNumber, Level: e.Level, Approvers: d.copiedAll(e.Approvers),
		Actor: e.Actor, OnBehalfOf: nonNil(e.OnBehalfOf), Action: string(e.Action), At: e.At, Comment: e.Comment,
		Changes: make([]ChangeView, 0, len(e.Changes)), Capacity: e.Capacity, AsDelegateOf: e.AsDelegateOf, Rule: e.Rule,
		EscalatedTo: e.EscalatedTo, WaitedDays: e.WaitedDays,
	}
	if e.Level > 0 {
		view.LevelLabel = levelLabel(e.LevelName, e.Level)
	}
	for _, c := range e.Changes {
		view.Changes = append(view.Changes, ChangeView{Label: c.Label, OldText: c.OldText, NewText: c.NewText, Text: changeText(c)})
	}
	return view
}

// latestDecline returns the activity's latest decline (FR-016), nil when it was never declined. entries are
// the activity's history entries, oldest first.
func latestDecline(entries []Action) *DeclineNote {
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Action != ActionDeclined {
			continue
		}
		return &DeclineNote{
			RoundNumber: e.RoundNumber, Level: e.Level, LevelLabel: levelLabel(e.LevelName, e.Level),
			Actor: e.Actor, OnBehalfOf: nonNil(e.OnBehalfOf), At: e.At, Comment: e.Comment,
			Capacity: e.Capacity, AsDelegateOf: e.AsDelegateOf,
		}
	}
	return nil
}

// nonNil returns actors, or an empty list for nil: the API's lists are never null.
func nonNil(actors []actor.Actor) []actor.Actor {
	if actors == nil {
		return []actor.Actor{}
	}
	return actors
}
