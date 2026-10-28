package approval

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

type decider struct {
	person      actor.Actor
	canApprove  bool
	rule        bool
	delegations []Delegation
}

func (s *Service) deciderOf(ctx context.Context, a actor.Actor, dl []Delegation) (decider, error) {
	can, err := s.access.CanApprove(ctx, []string{a.ID})
	if err != nil {
		return decider{}, err
	}
	rule, err := s.accessRule(ctx)
	if err != nil {
		return decider{}, err
	}
	return decider{person: a, canApprove: can[a.ID], rule: rule, delegations: dl}, nil
}

func (s *Service) currentDelegations(ctx context.Context, d directory) ([]Delegation, error) {
	all, err := s.repo.ListDelegations(ctx)
	if err != nil {
		return nil, err
	}
	today := date.Today(s.now())
	var current []Delegation
	var delegates []string
	for _, dl := range all {
		if dl.state(today) == DelegationCurrent && d.byID[dl.DelegateID].Active {
			current = append(current, dl)
			delegates = append(delegates, dl.DelegateID)
		}
	}
	if len(current) == 0 {
		return []Delegation{}, nil
	}
	can, err := s.access.CanApprove(ctx, unique(delegates))
	if err != nil {
		return nil, err
	}
	return where(current, func(dl Delegation) bool { return can[dl.DelegateID] }), nil
}

func delegatesOf(dl []Delegation, personID string) []string {
	var ids []string
	for _, x := range dl {
		if x.PersonID == personID {
			ids = append(ids, x.DelegateID)
		}
	}
	return ids
}

func countedFor(level RoundLevel, entries []Action) map[string]bool {
	counted := map[string]bool{}
	for _, e := range entries {
		if e.Level != level.Number || e.Action != ActionApproved {
			continue
		}
		switch e.Capacity {
		case CapacityApprover, "":
			counted[e.Actor.ID] = true
		case CapacityDelegate:
			if e.AsDelegateOf != nil {
				counted[e.AsDelegateOf.ID] = true
			}
		}
	}
	return counted
}

func waitingOn(level RoundLevel, entries []Action) []actor.Actor {
	counted := countedFor(level, entries)
	waiting := []actor.Actor{}
	for _, a := range level.Approvers {
		if !counted[a.ID] {
			waiting = append(waiting, a)
		}
	}
	return waiting
}

func levelDone(level RoundLevel, entries []Action) bool {
	approved := false
	for _, e := range entries {
		if e.Level != level.Number || e.Action != ActionApproved {
			continue
		}
		approved = true
		if e.Capacity == CapacityBackup || e.Capacity == CapacityOnBehalf || len(e.OnBehalfOf) > 0 {
			return true
		}
	}
	if !approved {
		return false
	}
	if level.Mode != ModeAll {
		return true
	}
	return len(waitingOn(level, entries)) == 0
}

type authority struct {
	allowed      bool
	capacity     Capacity
	asDelegateOf []actor.Actor
	onBehalfOf   []actor.Actor
	refusal      *errs.Error
	shown        string
	waiting      []actor.Actor
}

func (au authority) waits() bool {
	return au.allowed && au.capacity != CapacityOnBehalf
}

// decisionPoint is the current level of a record's open round as authority looks at it; onBehalf says whether
// to ask 014 about deciding on behalf (Waiting for me never does).
type decisionPoint struct {
	sub      subject
	round    Round
	entries  []Action
	onBehalf bool
}

// authority runs the check order of 017 data-model §6 for the decider on the current level: approver,
// delegate, backup approver after escalation, on behalf; then segregation of duties for activities.
func (s *Service) authority(ctx context.Context, dc decider, at decisionPoint) (authority, error) {
	level := at.round.Levels[at.round.CurrentLevel-1]
	au := namedCapacity(dc, level, at.entries)
	if !au.allowed && at.onBehalf {
		grant, err := s.access.Check(ctx, dc.person.ID, PermissionDecideOnBehalf, &at.sub.Plan)
		if err != nil {
			return authority{}, err
		}
		if grant.Allowed {
			au.allowed, au.capacity, au.onBehalfOf = true, CapacityOnBehalf, slices.Clone(au.waiting)
		}
	}
	if !au.allowed {
		return refused(au, dc.person.ID, level, at), nil
	}
	return segregated(au, dc, at), nil
}

func namedCapacity(dc decider, level RoundLevel, entries []Action) authority {
	p := dc.person.ID
	au := authority{waiting: waitingOn(level, entries)}
	if !dc.canApprove {
		return au
	}
	delegators := delegatorsOf(dc, au.waiting)
	escalatedToMe := level.EscalatedAt != nil && level.Backup != nil && level.Backup.ID == p
	switch {
	case includes(au.waiting, p):
		au.allowed, au.capacity = true, CapacityApprover
	case len(delegators) > 0:
		au.allowed, au.capacity, au.asDelegateOf = true, CapacityDelegate, delegators
	case escalatedToMe:
		au.allowed, au.capacity = true, CapacityBackup
	}
	return au
}

// delegatorsOf returns the waiting approvers who delegated to the decider today.
func delegatorsOf(dc decider, waiting []actor.Actor) []actor.Actor {
	var delegators []actor.Actor
	for _, w := range waiting {
		if slices.ContainsFunc(dc.delegations, func(dl Delegation) bool { return dl.PersonID == w.ID && dl.DelegateID == dc.person.ID }) {
			delegators = append(delegators, w)
		}
	}
	return delegators
}

func refused(au authority, p string, level RoundLevel, at decisionPoint) authority {
	if includes(level.Approvers, p) && countedFor(level, at.entries)[p] {
		au.shown = fmt.Sprintf(alreadyApprovedLevelMessage, level.Number, names(au.waiting))
		au.refusal = errs.Forbidden(au.shown)
		return au
	}
	who := names(level.Approvers)
	if len(level.Approvers) == 0 {
		who = nobodyText
	}
	au.refusal = errs.Forbidden(fmt.Sprintf(onlyApproversMessage, level.Number, who, at.sub.Kind.word()))
	return au
}

// segregated refuses the owner and the round's submitter while 014's access rule is on — for activities only.
func segregated(au authority, dc decider, at decisionPoint) authority {
	if !dc.rule || !at.sub.Kind.barsSelfApproval() {
		return au
	}
	p := dc.person.ID
	switch {
	case at.sub.OwnerID != "" && p == at.sub.OwnerID:
		au.shown = ownActivityMessage
	case p == at.round.SubmittedBy.ID:
		au.shown = submittedActivityMessage
	}
	if au.shown != "" {
		au.allowed, au.refusal = false, errs.Forbidden(au.shown)
	}
	return au
}

func capacitySuffix(au authority) string {
	switch au.capacity {
	case CapacityDelegate:
		return "; as delegate of " + au.asDelegateOf[0].Name
	case CapacityBackup:
		return "; as backup approver"
	case CapacityOnBehalf:
		if len(au.onBehalfOf) > 0 {
			return "; on behalf of " + names(au.onBehalfOf)
		}
		return "; on behalf of the approvers"
	}
	return ""
}

func (s *Service) NamedIn(ctx context.Context, personID, activityID string) (bool, error) {
	rounds, err := s.repo.RoundsOf(ctx, activityID)
	if err != nil {
		return false, err
	}
	for _, r := range rounds {
		for _, l := range r.Levels {
			if includes(l.Approvers, personID) {
				return true, nil
			}
		}
	}
	open, found := openRound(rounds)
	if !found {
		return false, nil
	}
	level := open.Levels[open.CurrentLevel-1]
	if level.EscalatedAt != nil && level.Backup != nil && level.Backup.ID == personID {
		return true, nil
	}
	d, err := s.directory(ctx)
	if err != nil {
		return false, err
	}
	dl, err := s.currentDelegations(ctx, d)
	if err != nil {
		return false, err
	}
	entries, err := s.repo.ActionsOf(ctx, activityID)
	if err != nil {
		return false, err
	}
	for _, w := range waitingOn(level, entriesOf(entries, open.ID)) {
		if slices.Contains(delegatesOf(dl, w.ID), personID) {
			return true, nil
		}
	}
	return false, nil
}

func approversText(level RoundLevel) string {
	if len(level.Approvers) == 0 {
		return "no approver"
	}
	return names(level.Approvers)
}

func levelNotes(levels []RoundLevel) string {
	var notes []string
	for _, l := range levels {
		switch l.Resolution {
		case ResolutionAutoApproved:
			notes = append(notes, fmt.Sprintf("level %d approved automatically (%s)", l.Number, l.RuleText))
		case ResolutionSkipped:
			notes = append(notes, fmt.Sprintf("level %d not needed (%s)", l.Number, l.RuleText))
		}
	}
	return strings.Join(notes, "; ")
}
