package approval

import (
	"context"

	"mapp/internal/platform/actor"
)

// Approval returns the activity's Approval section as a sees it.
func (s *Service) Approval(ctx context.Context, a actor.Actor, activityID string) (ActivityApproval, error) {
	var approval ActivityApproval
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		sec, err := s.section(ctx, a, activitySubject(facts))
		approval = activityApproval(sec)
		return err
	})
	if err != nil {
		return ActivityApproval{}, err
	}
	return approval, nil
}

// section builds a record's approval section inside the caller's transaction: every round of the record with
// its history, newest first, a's relation to it and the actions available now.
func (s *Service) section(ctx context.Context, a actor.Actor, sub subject) (Section, error) {
	rc, err := s.ruleContext(ctx)
	if err != nil {
		return Section{}, err
	}
	rounds, entries, err := s.recordRounds(ctx, sub.ID)
	if err != nil {
		return Section{}, err
	}
	open := openRoundPointer(rounds)
	view := Section{
		Subject: sub.view(), OwnerID: sub.OwnerID, Owner: rc.people.owner(sub.OwnerID), UpcomingLevels: []LevelView{},
		Rounds: make([]RoundView, 0, len(rounds)), Actions: availableActions(sub, open),
		Viewer: Viewer{IsOwner: sub.OwnerID != "" && sub.OwnerID == a.ID, OnBehalfOf: []PersonView{}, AsDelegateOf: []PersonView{}},
	}
	for i := len(rounds) - 1; i >= 0; i-- {
		view.Rounds = append(view.Rounds, roundView(rounds[i], entriesOf(entries, rounds[i].ID), rc))
	}
	if err = s.sectionWorkflow(ctx, &view, sub, rc); err != nil {
		return Section{}, err
	}
	if open != nil {
		at := decisionPoint{sub: sub.as(open.kind()), round: *open, entries: entriesOf(entries, open.ID), onBehalf: true}
		if err = s.viewerOn(ctx, &view, at, viewing{a: a, d: rc.people}); err != nil {
			return Section{}, err
		}
		if err = s.overagesOn(ctx, &view, at.sub); err != nil {
			return Section{}, err
		}
	}
	limitSubmit(ctx, &view.Actions, sub)
	if sub.Status == StatusDeclined {
		view.LatestDecline = latestDecline(entries)
	}
	return view, nil
}

func (s *Service) recordRounds(ctx context.Context, id string) ([]Round, []Action, error) {
	rounds, err := s.repo.RoundsOf(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.repo.ActionsOf(ctx, id)
	return rounds, entries, err
}

// sectionWorkflow shows the record's workflow now and, before its first round, the levels it would copy.
func (s *Service) sectionWorkflow(ctx context.Context, view *Section, sub subject, rc ruleContext) error {
	workflow, err := s.workflowOf(ctx, sub)
	if err != nil || workflow == nil {
		return err
	}
	view.Workflow = workflowRef(*workflow)
	if len(view.Rounds) == 0 {
		view.UpcomingLevels = levelViews(workflow.Levels, rc, workflow.Kind())
	}
	return nil
}

// viewing is the person reading a section, with the people read once.
type viewing struct {
	a actor.Actor
	d directory
}

func (s *Service) viewerOn(ctx context.Context, view *Section, at decisionPoint, v viewing) error {
	a, d := v.a, v.d
	level := at.round.Levels[at.round.CurrentLevel-1]
	view.Viewer.IsCurrentApprover = includes(level.Approvers, a.ID)
	view.Viewer.IsSubmitter = at.round.SubmittedBy.ID == a.ID
	dl, err := s.currentDelegations(ctx, d)
	if err != nil {
		return err
	}
	dc, err := s.deciderOf(ctx, a, dl)
	if err != nil {
		return err
	}
	au, err := s.authority(ctx, dc, at)
	if err != nil {
		return err
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
	if at.sub.waiting() {
		decide := ActionAvailability{Available: au.allowed, Reason: au.shown}
		view.Actions.Approve, view.Actions.Decline = decide, decide
	}
	limitDecisions(ctx, &view.Actions, at.sub, at.round)
	return nil
}

// availableActions says which approval actions the record offers (data-model §9); Approve and Decline follow
// who may decide, and an Approved activity explains why Recall is not offered.
func availableActions(sub subject, open *Round) ActionSet {
	var set ActionSet
	switch submissionKind(sub, open) {
	case kindSubmit:
		set.Submit.Available = true
	case kindResubmit:
		set.Resubmit.Available = true
	}
	set.Recall.Available = open != nil || (sub.Kind == KindActivity && sub.Status == StatusSubmitted)
	if sub.Kind == KindActivity && sub.Status == StatusApproved {
		set.Recall.Reason = approvedNoRecallMessage
	}
	return set
}
