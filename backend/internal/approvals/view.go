package approvals

import (
	"context"

	"northstar/internal/platform/actor"
)

// Approval returns the activity's Approval section as a sees it (FR-024, FR-025): the chain of every round
// with its history, newest first, the latest decline while Declined (FR-016), a's relation to the activity
// and the actions available now (FR-009, FR-014, FR-021).
func (s *Service) Approval(ctx context.Context, a actor.Actor, activityID string) (ActivityApproval, error) {
	var approval ActivityApproval
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		approval, err = s.approval(ctx, a, facts)
		return err
	})
	if err != nil {
		return ActivityApproval{}, err
	}
	return approval, nil
}

// approval builds the Approval section of facts, inside the caller's transaction.
func (s *Service) approval(ctx context.Context, a actor.Actor, facts ActivityFacts) (ActivityApproval, error) {
	d, err := s.directory(ctx)
	if err != nil {
		return ActivityApproval{}, err
	}
	rounds, err := s.repo.RoundsOf(ctx, facts.ID)
	if err != nil {
		return ActivityApproval{}, err
	}
	entries, err := s.repo.ActionsOf(ctx, facts.ID)
	if err != nil {
		return ActivityApproval{}, err
	}

	view := ActivityApproval{
		ActivityID: facts.ID, ActivityName: facts.Name, ActivityVersion: facts.Version,
		Status: facts.Status, DisplayStatus: facts.DisplayStatus, OwnerID: facts.OwnerID,
		UpcomingLevels: []LevelView{}, Rounds: make([]RoundView, 0, len(rounds)),
		Viewer:  Viewer{IsOwner: facts.OwnerID != "" && facts.OwnerID == a.ID, OnBehalfOf: []PersonView{}},
		Actions: availableActions(facts.Status),
	}
	if facts.WorkflowID != "" {
		workflow, found, err := s.repo.GetWorkflow(ctx, facts.WorkflowID)
		if err != nil {
			return ActivityApproval{}, err
		}
		if found {
			view.Workflow = workflowRef(workflow)
			if len(rounds) == 0 {
				view.UpcomingLevels = levelViews(workflow.Levels, d) // "Not submitted yet"
			}
		}
	}
	for i := len(rounds) - 1; i >= 0; i-- {
		view.Rounds = append(view.Rounds, roundView(rounds[i], entriesOf(entries, rounds[i].ID), d))
	}
	if open, found := openRound(rounds); found {
		approvers := open.Levels[open.CurrentLevel-1].Approvers
		view.Viewer.IsCurrentApprover = includes(approvers, a.ID)
		if !view.Viewer.IsCurrentApprover {
			view.Viewer.OnBehalfOf = d.copiedAll(approvers)
		}
	}
	if facts.Status == StatusDeclined {
		view.LatestDecline = latestDecline(entries)
	}
	return view, nil
}

// availableActions says which approval actions an activity with status offers (FR-009, FR-014, FR-021).
// Only an unavailable recall of an Approved activity explains itself: the screen must say why.
func availableActions(status string) ActionSet {
	var set ActionSet
	switch status {
	case StatusCreated:
		set.Submit.Available = true
	case StatusDeclined:
		set.Resubmit.Available = true
	case StatusSubmitted:
		set.Approve.Available, set.Decline.Available, set.Recall.Available = true, true, true
	case StatusApproved:
		set.Recall.Reason = approvedNoRecallMessage
	}
	return set
}

// openRound returns the open round among rounds, and false when there is none (an activity has at most
// one, exactly while it is Submitted).
func openRound(rounds []Round) (Round, bool) {
	for _, r := range rounds {
		if r.Outcome == OutcomeOpen {
			return r, true
		}
	}
	return Round{}, false
}

// entriesOf returns the history entries of the round roundID, oldest first; never nil.
func entriesOf(entries []Action, roundID string) []Action {
	return where(entries, func(e Action) bool { return e.RoundID == roundID })
}

// decisionOn returns the decision (approved or declined) on level in a round's entries, and false when
// the level was not decided. Only the first decision on a level counts (FR-018), and there is only one.
func decisionOn(entries []Action, level int) (Action, bool) {
	for _, e := range entries {
		if e.Level == level && (e.Action == ActionApproved || e.Action == ActionDeclined) {
			return e, true
		}
	}
	return Action{}, false
}

// roundView returns round with its chain and its history, newest first (FR-024, FR-025). entries are the
// round's history entries, oldest first.
func roundView(round Round, entries []Action, d directory) RoundView {
	view := RoundView{
		ID: round.ID, Number: round.Number, Outcome: round.Outcome,
		Workflow:    RoundWorkflow{ID: round.WorkflowID, Name: round.WorkflowName},
		SubmittedAt: round.SubmittedAt, SubmittedBy: round.SubmittedBy, Comment: round.Comment, EndedAt: round.EndedAt,
		CurrentLevel: round.CurrentLevel, LevelCount: len(round.Levels),
		Levels: make([]ChainLevel, 0, len(round.Levels)), History: make([]HistoryEntryView, 0, len(entries)),
	}
	for _, l := range round.Levels {
		level := ChainLevel{Number: l.Number, Label: levelLabel(l.Name, l.Number), Approvers: d.copiedAll(l.Approvers)}
		decision, decided := decisionOn(entries, l.Number)
		switch {
		case decided:
			level.State = string(decision.Action)
			level.Decision = &Decision{
				Action: string(decision.Action), Actor: decision.Actor, OnBehalfOf: nonNil(decision.OnBehalfOf),
				At: decision.At, Comment: decision.Comment,
			}
		case l.Number == round.CurrentLevel && round.Outcome == OutcomeOpen:
			level.State = LevelWaiting
		case l.Number == round.CurrentLevel: // recalled, restarted or cancelled while it waited
			level.State = LevelNotDecided
		default:
			level.State = LevelNotReached
		}
		view.Levels = append(view.Levels, level)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		view.History = append(view.History, historyEntryView(entries[i], d))
	}
	return view
}

// historyEntryView returns one history entry as the API shows it, with the sentences of its changes.
func historyEntryView(e Action, d directory) HistoryEntryView {
	view := HistoryEntryView{
		ID: e.ID, RoundNumber: e.RoundNumber, Level: e.Level, Approvers: d.copiedAll(e.Approvers),
		Actor: e.Actor, OnBehalfOf: nonNil(e.OnBehalfOf), Action: string(e.Action), At: e.At, Comment: e.Comment,
		Changes: make([]ChangeView, 0, len(e.Changes)),
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
