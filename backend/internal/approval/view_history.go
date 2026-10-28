package approval

import (
	"mapp/internal/platform/actor"
)

func entriesOf(entries []Action, roundID string) []Action {
	return where(entries, func(e Action) bool { return e.RoundID == roundID })
}

func decisionOn(entries []Action, level int) (Action, bool) {
	for _, e := range entries {
		if e.Level == level && (e.Action == ActionApproved || e.Action == ActionDeclined) {
			return e, true
		}
	}
	return Action{}, false
}

func decisionsOn(entries []Action, level int) []Action {
	return where(entries, func(e Action) bool {
		return e.Level == level && (e.Action == ActionApproved || e.Action == ActionDeclined)
	})
}

func decisionView(e Action) Decision {
	return Decision{
		Action: string(e.Action), Actor: e.Actor, OnBehalfOf: nonNil(e.OnBehalfOf), At: e.At, Comment: e.Comment,
		Capacity: e.Capacity, AsDelegateOf: e.AsDelegateOf, SelfApproval: e.SelfApproval,
	}
}

func historyEntryView(e Action, d directory) HistoryEntryView {
	view := HistoryEntryView{
		ID: e.ID, RoundNumber: e.RoundNumber, Level: e.Level, Approvers: d.copiedAll(e.Approvers),
		Actor: e.Actor, OnBehalfOf: nonNil(e.OnBehalfOf), Action: string(e.Action), At: e.At, Comment: e.Comment,
		Changes: make([]ChangeView, 0, len(e.Changes)), Capacity: e.Capacity, AsDelegateOf: e.AsDelegateOf, Rule: e.Rule,
		EscalatedTo: e.EscalatedTo, WaitedDays: e.WaitedDays, SelfApproval: e.SelfApproval,
	}
	if e.Level > 0 {
		view.LevelLabel = levelLabel(e.LevelName, e.Level)
	}
	for _, c := range e.Changes {
		view.Changes = append(view.Changes, ChangeView{Label: c.Label, OldText: c.OldText, NewText: c.NewText, Text: changeText(c)})
	}
	return view
}

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

func nonNil(actors []actor.Actor) []actor.Actor {
	if actors == nil {
		return []actor.Actor{}
	}
	return actors
}
