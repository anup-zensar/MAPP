package approvals

import (
	"context"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Approval returns the activity's Approval section as a sees it (FR-024, FR-025): the chain of every round
// with its history, newest first, the latest decline while Declined (FR-016), a's relation to the activity
// and the actions available now (FR-009, FR-014, FR-021; 017 FR-017, FR-033, FR-034).
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
	rc, err := s.ruleContext(ctx)
	if err != nil {
		return ActivityApproval{}, err
	}
	d := rc.people
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
		Status: facts.Status, DisplayStatus: facts.DisplayStatus, OwnerID: facts.OwnerID, Owner: d.owner(facts.OwnerID),
		UpcomingLevels: []LevelView{}, Rounds: make([]RoundView, 0, len(rounds)),
		Viewer: Viewer{
			IsOwner: facts.OwnerID != "" && facts.OwnerID == a.ID, OnBehalfOf: []PersonView{}, AsDelegateOf: []PersonView{},
		},
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
				view.UpcomingLevels = levelViews(workflow.Levels, rc) // "Not submitted yet"
			}
		}
	}
	for i := len(rounds) - 1; i >= 0; i-- {
		view.Rounds = append(view.Rounds, roundView(rounds[i], entriesOf(entries, rounds[i].ID), rc))
	}
	if open, found := openRound(rounds); found {
		level := open.Levels[open.CurrentLevel-1]
		view.Viewer.IsCurrentApprover = includes(level.Approvers, a.ID)
		view.Viewer.IsSubmitter = open.SubmittedBy.ID == a.ID
		dl, err := s.currentDelegations(ctx, d)
		if err != nil {
			return ActivityApproval{}, err
		}
		dc, err := s.deciderOf(ctx, a, dl)
		if err != nil {
			return ActivityApproval{}, err
		}
		au, err := s.authority(ctx, dc, facts, open, entriesOf(entries, open.ID), true)
		if err != nil {
			return ActivityApproval{}, err
		}
		view.Viewer.CanDecide, view.Viewer.CannotDecideReason = au.allowed, au.shown
		if au.allowed {
			capacity := au.capacity
			view.Viewer.Capacity = &capacity
			view.Viewer.AsDelegateOf = d.copiedAll(au.asDelegateOf)
			if au.capacity == CapacityOnBehalf {
				view.Viewer.OnBehalfOf = d.copiedAll(au.onBehalfOf)
			}
		}
		if facts.Status == StatusSubmitted {
			decide := ActionAvailability{Available: au.allowed, Reason: au.shown}
			view.Actions.Approve, view.Actions.Decline = decide, decide
		}
		limitDecisions(ctx, &view.Actions, facts, open) // 014 FR-025 – FR-027
	}
	limitSubmit(ctx, &view.Actions, facts) // 014 FR-027
	if facts.Status == StatusDeclined {
		view.LatestDecline = latestDecline(entries)
	}
	return view, nil
}

// availableActions says which approval actions an activity with status offers (FR-009, FR-014, FR-021).
// Only an unavailable recall of an Approved activity explains itself: the screen must say why. Approve and
// decline of a Submitted activity follow who may decide (approval, 017 FR-017).
func availableActions(status string) ActionSet {
	var set ActionSet
	switch status {
	case StatusCreated:
		set.Submit.Available = true
	case StatusDeclined:
		set.Resubmit.Available = true
	case StatusSubmitted:
		set.Recall.Available = true
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

// waitingSince is when the round reached the level (017 FR-029): its stored time, or for rounds stored
// before 017 008's rule — the submission for the first level, else the approval of the level before. nil for
// a level not reached, not needed or approved automatically.
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

// roundView returns round with its chain and its history, newest first (FR-024, FR-025; 017 FR-033,
// FR-034). entries are the round's history entries, oldest first.
func roundView(round Round, entries []Action, rc ruleContext) RoundView {
	d := rc.people
	view := RoundView{
		ID: round.ID, Number: round.Number, Outcome: round.Outcome,
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

// chainLevel returns one level of a round as the chain shows it (017 FR-033): what the rules decided, who
// approves it and how they were found, who was left out, its decisions and progress, its timings and
// whether nobody waiting can act.
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
		decision := decisionView(decisions[len(decisions)-1]) // the approval that completed it
		level.Decision = &decision
	case l.Number == round.CurrentLevel && round.Outcome == OutcomeOpen:
		level.State = LevelWaiting
		if !level.NoApprover && len(l.Approvers) > 0 {
			level.NoApprover = !slices.ContainsFunc(waitingOn(l, entries), func(a actor.Actor) bool {
				return d.byID[a.ID].Active && rc.canApprove[a.ID]
			})
		}
	case l.Number == round.CurrentLevel: // recalled, restarted or cancelled while it waited
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

// excludedViews returns the people left out of a level as the API shows them; never nil.
func excludedViews(excluded []Excluded, d directory) []ExcludedView {
	views := make([]ExcludedView, 0, len(excluded))
	for _, e := range excluded {
		views = append(views, ExcludedView{Person: d.copied(e.Person), Reason: e.Reason})
	}
	return views
}

// factsView returns the values the rules used as the API shows them; nil for rounds stored before 017.
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
