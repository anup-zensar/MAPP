package approvals

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
)

// Approver resolution at submission (017 FR-011, FR-012, data-model §3 steps 3–6): who approves each level
// that needs a decision, who was left out and why, and the problems of a level whose source names nobody.

// resolution is what submitting an activity would do with its workflow's levels: the round's levels and the
// problems that block the submission (one per level whose source names nobody, FR-012).
type resolution struct {
	levels   []RoundLevel
	problems []SubmitProblem
}

// noApproverAdvice is the advice of a level with no approver (data-model §3): 014's when the person
// submitting was the only one left out as owner or submitter, otherwise the general one. "" when the level
// has an approver.
func noApproverAdvice(l RoundLevel, submitterID string) string {
	if !l.NoApprover {
		return ""
	}
	var self []Excluded
	for _, e := range l.Excluded {
		if e.Reason == ReasonOwner || e.Reason == ReasonSubmitter {
			self = append(self, e)
		}
	}
	if len(self) == 1 && self[0].Person.ID == submitterID {
		return fmt.Sprintf(onlyApproverAdvice, l.Number)
	}
	return fmt.Sprintf(nobodyElseAdvice, l.Number)
}

// accessRule reads whether 014's access rule "Nobody approves an activity they own or submitted" is on.
func (s *Service) accessRule(ctx context.Context) (bool, error) {
	return s.access.NoSelfApproval(ctx)
}

// roles returns 014's roles by id.
func (s *Service) roles(ctx context.Context) (map[string]RoleInfo, error) {
	list, err := s.access.Roles(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]RoleInfo, len(list))
	for _, r := range list {
		byID[r.ID] = r
	}
	return byID, nil
}

// resolve finds the approvers of each level of plans that needs a decision, for the activity facts that
// submitter submits (data-model §3 steps 3–6): the named people, the owner's line manager or the holders of
// the level's role; then leaves out, in this order of reasons, inactive people, people who cannot approve
// and — while 014's access rule is on — the owner and the submitter. When nobody is left the level's backup
// approver, if usable, approves it; otherwise it waits with no approver. A level whose source names nobody
// adds a problem instead (FR-012).
func (s *Service) resolve(ctx context.Context, plans []levelPlan, facts ActivityFacts, submitter actor.Actor, d directory) (resolution, error) {
	rule, err := s.accessRule(ctx)
	if err != nil {
		return resolution{}, err
	}
	roles, err := s.roles(ctx)
	if err != nil {
		return resolution{}, err
	}
	result := resolution{levels: make([]RoundLevel, 0, len(plans)), problems: []SubmitProblem{}}
	ownerName := d.person(facts.OwnerID).Name
	for _, plan := range plans {
		l := plan.level
		level := RoundLevel{
			Number: plan.number, Name: l.Name, Approvers: []actor.Actor{}, ApproverType: l.ApproverType, Mode: l.Mode,
			Resolution: plan.resolution, RuleText: plan.ruleText, Excluded: []Excluded{},
		}
		if level.ApproverType == SourceRole {
			level.RoleID, level.RoleName = l.RoleID, roles[l.RoleID].Name
		}
		if plan.resolution != ResolutionDecide {
			result.levels = append(result.levels, level)
			continue
		}

		var candidates []string
		problem := ""
		switch l.ApproverType {
		case SourceManager:
			level.Source = "line manager of " + ownerName
			if manager := d.byID[facts.OwnerID].ManagerID; manager != "" {
				candidates = []string{manager}
			} else if facts.OwnerID != "" {
				problem = fmt.Sprintf(noLineManagerProblem, plan.number, ownerName)
			}
		case SourceRole:
			role, known := roles[l.RoleID]
			if !known {
				problem = fmt.Sprintf(roleGoneProblem, plan.number)
				break
			}
			level.Source = "role " + role.Name
			holders, err := s.access.RoleHolders(ctx, l.RoleID, facts.Plan)
			if err != nil {
				return resolution{}, err
			}
			candidates = holders
			if len(holders) == 0 {
				problem = fmt.Sprintf(nobodyHoldsRoleProblem, plan.number, role.Name)
			}
		default:
			candidates = l.ApproverIDs
		}
		if problem != "" {
			result.problems = append(result.problems, SubmitProblem{Field: workflowField, Group: GroupActivity, Message: problem})
			result.levels = append(result.levels, level)
			continue
		}

		candidates = unique(candidates)
		backupID := l.BackupApproverID
		can, err := s.access.CanApprove(ctx, append(slices.Clone(candidates), backupID))
		if err != nil {
			return resolution{}, err
		}
		reasonOf := func(id string) string {
			switch {
			case !d.byID[id].Active:
				return ReasonInactive
			case !can[id]:
				return ReasonCannotApprove
			case rule && id == facts.OwnerID:
				return ReasonOwner
			case rule && id == submitter.ID:
				return ReasonSubmitter
			}
			return ""
		}
		var kept []string
		for _, id := range candidates {
			if reason := reasonOf(id); reason != "" {
				level.Excluded = append(level.Excluded, Excluded{Person: d.actors([]string{id})[0], Reason: reason})
			} else {
				kept = append(kept, id)
			}
		}
		slices.SortStableFunc(level.Excluded, func(x, y Excluded) int { return cmp.Compare(reasonRank(x.Reason), reasonRank(y.Reason)) })

		usableBackup := backupID != "" && reasonOf(backupID) == ""
		switch {
		case len(kept) > 0:
			level.Approvers = d.actors(kept)
		case usableBackup:
			level.Approvers, level.BackupUsed = d.actors([]string{backupID}), true
		default:
			level.NoApprover = true
		}
		if usableBackup && !slices.Contains(kept, backupID) && !level.BackupUsed {
			backup := d.actors([]string{backupID})[0]
			level.Backup = &backup
		}
		result.levels = append(result.levels, level)
	}
	return result, nil
}

// reasonRank orders the reasons a person is left out (data-model §3 step 4).
func reasonRank(reason string) int {
	return slices.Index([]string{ReasonInactive, ReasonCannotApprove, ReasonOwner, ReasonSubmitter}, reason)
}

// unique returns ids without empty ids and repeats, in order.
func unique(ids []string) []string {
	var list []string
	for _, id := range ids {
		if id != "" && !slices.Contains(list, id) {
			list = append(list, id)
		}
	}
	return list
}

// firstDecideLevel returns the number of the first level of levels whose approvers decide, after the level
// after (0: from the start), and 0 when there is none.
func firstDecideLevel(levels []RoundLevel, after int) int {
	for _, l := range levels {
		if l.Number > after && l.Resolution == ResolutionDecide {
			return l.Number
		}
	}
	return 0
}
