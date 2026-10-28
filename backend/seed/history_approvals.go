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

// addApprovalActions logs every approval action of the last 14 days as 008 logs it on the activity (step 3);
// older ones stay only in the activity's own approval history.
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

	// A stable sort keeps the id order of actions of the same moment, so each knows the status before it.
	slices.SortStableFunc(actions, func(x, y approval.Action) int { return x.At.Compare(y.At) })
	logStart := h.at.Add(-HistoryDays * day)
	status := map[string]activity.Status{}
	for _, action := range actions {
		round, found := roundByID[action.RoundID]
		act, stored := activityByID[action.SubjectID]
		if !found || !stored {
			return fmt.Errorf("seed: approval action %s has no round or activity", action.ID)
		}
		before := cmp.Or(status[action.SubjectID], activity.StatusCreated)
		after, verb, details := approvalStep(action, round, before)
		status[action.SubjectID] = after
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

// approvalStep returns the activity's status after one approval action, the verb and the details.
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
