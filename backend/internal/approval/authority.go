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

// Who may decide (017 FR-017 – FR-020, data-model §6): 014's approve/decline check order, extended with the
// delegates of waiting approvers, the backup approver of an escalated level and the round's submitter in
// segregation of duties. Every check runs on the server; the answers' actions follow the same steps.

// decider is the signed-in person as the checks need them, read once per operation: whether they hold
// approvals.decide, whether 014's access rule is on, and the delegations current today.
type decider struct {
	person      actor.Actor
	canApprove  bool
	rule        bool
	delegations []Delegation // current today, with an active delegate who can approve (FR-023)
}

// deciderOf reads the decider a, with the current delegations dl.
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

// currentDelegations returns the delegations that count today (FR-023, data-model §7 Effect): not
// cancelled, today within their dates, and their delegate active and holding approvals.decide.
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

// delegatesOf returns the ids of the current delegates of the person personID among dl.
func delegatesOf(dl []Delegation, personID string) []string {
	var ids []string
	for _, x := range dl {
		if x.PersonID == personID {
			ids = append(ids, x.DelegateID)
		}
	}
	return ids
}

// countedFor returns the approvers of the level who have an approval counted for them among entries (data
// model §5): their own (capacity approver, or none for entries written before 017), or their delegate's.
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

// waitingOn returns the resolved approvers of the level with no approval counted for them, in order.
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

// levelDone reports whether the level is approved by entries (017 FR-015, data-model §4): by its first
// approval when any one approves; when every resolved approver has an approval counted for them when all
// must approve; and in both modes by an approval of the backup approver or on behalf.
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

// authority is whether the decider may approve or decline the current level of a round, and in which
// capacity (data-model §6).
type authority struct {
	allowed      bool
	capacity     Capacity
	asDelegateOf []actor.Actor // delegate: every waiting approver who delegated to the person today
	onBehalfOf   []actor.Actor // onBehalf: the waiting approvers
	refusal      *errs.Error   // 403 forbidden; nil when allowed
	shown        string        // the reason of actions.approve and decline; "" hides the buttons (014)
	waiting      []actor.Actor // the level's approvers still waiting
}

// waits reports whether the round belongs in the decider's Waiting for me: they may decide it as approver,
// delegate or backup approver — never on behalf (FR-031).
func (au authority) waits() bool {
	return au.allowed && au.capacity != CapacityOnBehalf
}

// authority runs the check order of data-model §6 for the decider dc on the current level of the open
// round, whose history entries are entries: steps 3a–3f (approver, delegate, backup approver after
// escalation, on behalf, already approved, anyone else), then step 4 (segregation of duties while 014's
// rule is on: not the owner, not the round's submitter, in any capacity). onBehalf says whether to ask 014
// about deciding on behalf (step 3d); Waiting for me never does.
func (s *Service) authority(ctx context.Context, dc decider, facts ActivityFacts, round Round, entries []Action, onBehalf bool) (authority, error) {
	level := round.Levels[round.CurrentLevel-1]
	p := dc.person.ID
	au := authority{waiting: waitingOn(level, entries)}

	var delegators []actor.Actor
	for _, w := range au.waiting {
		if slices.ContainsFunc(dc.delegations, func(dl Delegation) bool { return dl.PersonID == w.ID && dl.DelegateID == p }) {
			delegators = append(delegators, w)
		}
	}
	switch {
	case includes(au.waiting, p) && dc.canApprove:
		au.allowed, au.capacity = true, CapacityApprover
	case len(delegators) > 0 && dc.canApprove:
		au.allowed, au.capacity, au.asDelegateOf = true, CapacityDelegate, delegators
	case level.EscalatedAt != nil && level.Backup != nil && level.Backup.ID == p && dc.canApprove:
		au.allowed, au.capacity = true, CapacityBackup
	}
	if !au.allowed && onBehalf {
		grant, err := s.access.Check(ctx, p, PermissionDecideOnBehalf, &facts.Plan)
		if err != nil {
			return authority{}, err
		}
		if grant.Allowed {
			au.allowed, au.capacity, au.onBehalfOf = true, CapacityOnBehalf, slices.Clone(au.waiting)
		}
	}
	if !au.allowed {
		if includes(level.Approvers, p) && countedFor(level, entries)[p] {
			au.shown = fmt.Sprintf(alreadyApprovedLevelMessage, level.Number, names(au.waiting))
			au.refusal = errs.Forbidden(au.shown)
			return au, nil
		}
		who := names(level.Approvers)
		if len(level.Approvers) == 0 {
			who = nobodyText
		}
		au.refusal = errs.Forbidden(fmt.Sprintf(onlyApproversMessage, level.Number, who))
		return au, nil
	}

	switch {
	case dc.rule && facts.OwnerID != "" && p == facts.OwnerID:
		au.shown = ownActivityMessage
	case dc.rule && p == round.SubmittedBy.ID:
		au.shown = submittedActivityMessage
	}
	if au.shown != "" {
		au.allowed, au.refusal = false, errs.Forbidden(au.shown)
	}
	return au, nil
}

// capacitySuffix is the end of a decision's log details that says its capacity (plan, Activity-log
// entries): "; as delegate of <name>", "; as backup approver", "; on behalf of <names>".
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

// NamedIn reports whether the person personID may see the activity activityID because of its approval
// (014 FR-006, extended by 017 FR-021): they are named as an approver in any of its rounds, are a current
// delegate of a waiting approver of its open round, or are the backup approver of its escalated current
// level. Feature 014's activities.Approvers.NamedIn adapter calls it, inside the caller's transaction.
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

// approversText names the approvers of a level for the log: their names, or "no approver".
func approversText(level RoundLevel) string {
	if len(level.Approvers) == 0 {
		return "no approver"
	}
	return names(level.Approvers)
}

// levelNotes are the notes of the levels that did not wait at submission (plan, Activity-log entries):
// "level 2 not needed (only when …); level 3 approved automatically (…)".
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
