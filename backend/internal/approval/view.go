package approval

import (
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

func openRound(rounds []Round) (Round, bool) {
	for _, r := range rounds {
		if r.Outcome == OutcomeOpen {
			return r, true
		}
	}
	return Round{}, false
}

func waitingSince(round Round, level RoundLevel, entries []Action) *time.Time {
	if level.StartedAt != nil {
		return level.StartedAt
	}
	if level.Resolution != ResolutionDecide || level.Number > round.CurrentLevel || round.Facts != nil {
		return nil
	}
	since := round.SubmittedAt
	if previous, found := decisionOn(entries, level.Number-1); found {
		since = previous.At
	}
	return &since
}

func roundView(round Round, entries []Action, rc ruleContext) RoundView {
	d := rc.people
	view := RoundView{
		ID: round.ID, Kind: round.kind(), Number: round.Number, Outcome: round.Outcome,
		Workflow:    RoundWorkflow{ID: round.WorkflowID, Name: round.WorkflowName},
		SubmittedAt: round.SubmittedAt, SubmittedBy: round.SubmittedBy, Comment: round.Comment, EndedAt: round.EndedAt,
		CurrentLevel: round.CurrentLevel, LevelCount: len(round.Levels),
		Levels: make([]ChainLevel, 0, len(round.Levels)), History: make([]HistoryEntryView, 0, len(entries)),
		Facts: factsView(round.Facts, rc), ReminderDays: round.ReminderDays, EscalationDays: round.EscalationDays,
		AutoApproved: round.AutoApproved,
	}
	for _, l := range round.Levels {
		view.Levels = append(view.Levels, chainLevel(round, l, entries, rc))
	}
	for i := len(entries) - 1; i >= 0; i-- {
		view.History = append(view.History, historyEntryView(entries[i], d))
	}
	return view
}

func chainLevel(round Round, l RoundLevel, entries []Action, rc ruleContext) ChainLevel {
	d := rc.people
	level := ChainLevel{
		Number: l.Number, Label: levelLabel(l.Name, l.Number), Approvers: d.copiedAll(l.Approvers),
		Mode: l.Mode, Source: l.Source, RuleText: l.RuleText, Decisions: []Decision{}, RequiredCount: 1,
		Excluded: excludedViews(l.Excluded, d), BackupUsed: l.BackupUsed, WaitingSince: waitingSince(round, l, entries),
		RemindedAt: l.RemindedAt, EscalatedAt: l.EscalatedAt, NoApprover: l.NoApprover,
	}
	if l.Mode == ModeAll {
		level.RequiredCount = len(l.Approvers)
	}
	if l.Backup != nil {
		backup := d.copied(*l.Backup)
		level.Backup = &backup
		if l.EscalatedAt != nil {
			level.EscalatedTo = &backup
		}
	}
	decisions := decisionsOn(entries, l.Number)
	for _, e := range decisions {
		level.Decisions = append(level.Decisions, decisionView(e))
	}
	done := levelDone(l, entries)
	if l.Mode == ModeAll {
		level.ApprovedCount = len(l.Approvers) - len(waitingOn(l, entries))
	} else if done {
		level.ApprovedCount = 1
	}
	declined := slices.IndexFunc(decisions, func(e Action) bool { return e.Action == ActionDeclined })
	switch {
	case l.Resolution == ResolutionSkipped:
		level.State = LevelSkipped
	case l.Resolution == ResolutionAutoApproved:
		level.State = LevelAutoApproved
	case declined >= 0:
		level.State = LevelDeclined
		decision := decisionView(decisions[declined])
		level.Decision = &decision
	case done:
		level.State = LevelApproved
		decision := decisionView(decisions[len(decisions)-1])
		level.Decision = &decision
	case l.Number == round.CurrentLevel && round.Outcome == OutcomeOpen:
		level.State = LevelWaiting
		if !level.NoApprover && len(l.Approvers) > 0 {
			level.NoApprover = !slices.ContainsFunc(waitingOn(l, entries), func(a actor.Actor) bool {
				return d.byID[a.ID].Active && rc.canApprove[a.ID]
			})
		}
	case l.Number == round.CurrentLevel:
		level.State = LevelNotDecided
	default:
		level.State = LevelNotReached
	}
	if level.NoApprover {
		if l.NoApprover {
			level.Advice = noApproverAdvice(l, round.SubmittedBy.ID)
		} else {
			derived := l
			derived.NoApprover, derived.Excluded = true, nil
			level.Advice = noApproverAdvice(derived, round.SubmittedBy.ID)
		}
	}
	return level
}

func excludedViews(excluded []Excluded, d directory) []ExcludedView {
	views := make([]ExcludedView, 0, len(excluded))
	for _, e := range excluded {
		views = append(views, ExcludedView{Person: d.copied(e.Person), Reason: e.Reason})
	}
	return views
}

func factsView(f *RuleFacts, rc ruleContext) *RuleFactsView {
	if f == nil {
		return nil
	}
	view := &RuleFactsView{PlannedCost: f.PlannedCost, BusinessUnits: make([]OptionRef, 0, len(f.BusinessUnits))}
	if f.ActivityType != nil {
		view.ActivityType = &errs.RecordRef{Type: "activityType", ID: f.ActivityType.ID, Name: f.ActivityType.Name}
	}
	for _, o := range f.BusinessUnits {
		choice, known := rc.names.businessUnits[o.ID]
		view.BusinessUnits = append(view.BusinessUnits, OptionRef{ID: o.ID, Label: o.Label, Active: !known || choice.Active})
	}
	return view
}
