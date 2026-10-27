package seed

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/history"
)

// addApprovalActions keeps one "statusChanged" entry for every approval action of 008's seed in the last 14
// days, as 008 logs it on the activity: the verb ("submitted", "re-submitted", "approved", "approved level
// 1 of", "declined"), the actor, the time, the comment, the round's details and the status change (step 3).
// Rounds that ended before the log starts are left out; the activity's own approval history keeps them.
func (h *demoHistory) addApprovalActions(ctx context.Context) error {
	repo := h.stores.Approvals
	rounds, err := repo.ListRounds(ctx)
	if err != nil {
		return err
	}
	actions, err := repo.ListActions(ctx)
	if err != nil {
		return err
	}
	roundByID, activityByID := h.approvalLookups(rounds)

	// Every action in time order, so each one knows the status the activity had before it.
	// The repository lists them by ID; a stable sort keeps that order for actions of the same moment.
	slices.SortStableFunc(actions, func(x, y approval.Action) int { return x.At.Compare(y.At) })
	logStart := h.at.Add(-HistoryDays * day)
	status := map[string]activity.Status{} // activity ID → its status after the actions so far
	for _, action := range actions {
		round, found := roundByID[action.RoundID]
		act, stored := activityByID[action.ActivityID]
		if !found || !stored {
			return fmt.Errorf("seed: approval action %s has no round or activity", action.ID)
		}
		before := cmp.Or(status[action.ActivityID], activity.StatusCreated)
		after, verb, details := approvalStep(action, round, before)
		status[action.ActivityID] = after
		if action.At.Before(logStart) {
			continue
		}
		changes := []history.Change{}
		if after != before {
			change, err := statusChange(before, after, activityStatusWords())
			if err != nil {
				return err
			}
			changes = append(changes, change)
		}
		h.add(action.At, action.Actor, history.ActionStatusChanged, verb, act.kind, act.id, act.name,
			act.path, changes, action.Comment, details)
	}
	return nil
}

// approvalLookups returns the rounds and the stored activities by ID.
func (h *demoHistory) approvalLookups(rounds []approval.Round) (map[string]approval.Round, map[string]treeRecord) {
	roundByID := map[string]approval.Round{}
	for _, round := range rounds {
		roundByID[round.ID] = round
	}
	activityByID := map[string]treeRecord{}
	for _, r := range h.records {
		if r.activity != nil {
			activityByID[r.id] = r
		}
	}
	return roundByID, activityByID
}

// approvalStep returns what one approval action did, as 008 logs it (approvals rounds.go): the activity's
// status after it, the verb and the details.
func approvalStep(action approval.Action, round approval.Round, before activity.Status) (activity.Status, string, string) {
	approverNames := func(level int) string {
		list := make([]string, 0, len(round.Levels[level-1].Approvers))
		for _, a := range round.Levels[level-1].Approvers {
			list = append(list, a.Name)
		}
		return strings.Join(list, ", ")
	}
	if action.Action == approval.ActionSubmitted || action.Action == approval.ActionResubmitted {
		verb := "submitted"
		if action.Action == approval.ActionResubmitted {
			verb = "re-submitted"
		}
		return activity.StatusSubmitted, verb,
			fmt.Sprintf("Round %d · %s · waiting for level 1 (%s)", round.Number, round.WorkflowName, approverNames(1))
	}
	decided := fmt.Sprintf("Round %d · level %d of %d", round.Number, action.Level, len(round.Levels))
	suffix := ""
	if len(action.OnBehalfOf) > 0 {
		names := make([]string, 0, len(action.OnBehalfOf))
		for _, a := range action.OnBehalfOf {
			names = append(names, a.Name)
		}
		suffix = "; on behalf of " + strings.Join(names, ", ")
	}
	switch {
	case action.Action == approval.ActionDeclined:
		return activity.StatusDeclined, "declined", decided + suffix
	case action.Level == len(round.Levels):
		return activity.StatusApproved, "approved", decided + suffix
	default:
		next := action.Level + 1
		return before, fmt.Sprintf("approved level %d of", action.Level),
			fmt.Sprintf("%s · now waiting for level %d (%s)%s", decided, next, approverNames(next), suffix)
	}
}
