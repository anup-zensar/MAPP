package approval

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
)

// resolution is what submitting would do with a workflow's levels: the round's levels and the problems that
// block the submission.
type resolution struct {
	levels   []RoundLevel
	problems []SubmitProblem
}

type resolving struct {
	sub       subject
	submitter actor.Actor
	d         directory
	selfBar   bool // 014's access rule applies: owner and submitter are left out (activities only, D-324)
	roles     map[string]RoleInfo
}

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

func (s *Service) accessRule(ctx context.Context) (bool, error) {
	return s.access.NoSelfApproval(ctx)
}

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

// resolve finds the approvers of each level that needs a decision:
// the level's source names candidates; inactive people, people who cannot approve and — for activities while
// 014's rule is on — the owner and the submitter are left out; then the backup approver, else no approver.
func (s *Service) resolve(ctx context.Context, plans []levelPlan, r resolving) (resolution, error) {
	rule, err := s.accessRule(ctx)
	if err != nil {
		return resolution{}, err
	}
	if r.roles, err = s.roles(ctx); err != nil {
		return resolution{}, err
	}
	r.selfBar = rule && r.sub.Kind.barsSelfApproval()
	result := resolution{levels: make([]RoundLevel, 0, len(plans)), problems: []SubmitProblem{}}
	for _, plan := range plans {
		level, problem, err := s.resolveLevel(ctx, plan, r)
		if err != nil {
			return resolution{}, err
		}
		if problem != "" {
			result.problems = append(result.problems, SubmitProblem{
				Field: workflowFieldOf(r.sub.Kind), Group: recordGroup(r.sub.Kind), Message: problem,
			})
		}
		result.levels = append(result.levels, level)
	}
	return result, nil
}

func (s *Service) resolveLevel(ctx context.Context, plan levelPlan, r resolving) (RoundLevel, string, error) {
	l := plan.level
	level := RoundLevel{
		Number: plan.number, Name: l.Name, Approvers: []actor.Actor{}, ApproverType: l.ApproverType, Mode: l.Mode,
		Resolution: plan.resolution, RuleText: plan.ruleText, Excluded: []Excluded{},
	}
	if level.ApproverType == SourceRole {
		level.RoleID, level.RoleName = l.RoleID, r.roles[l.RoleID].Name
	}
	if plan.resolution != ResolutionDecide {
		return level, "", nil
	}
	c, err := s.candidates(ctx, plan, r)
	level.Source = c.source
	if err != nil || c.problem != "" {
		return level, c.problem, err
	}
	c.ids, c.backupID = unique(c.ids), l.BackupApproverID
	err = s.keepApprovers(ctx, &level, c, r)
	return level, "", err
}

// found are the candidates a level's source names, how they were found, or why it names nobody; and the
// level's backup approver.
type found struct {
	ids      []string
	source   string
	problem  string
	backupID string
}

func (s *Service) candidates(ctx context.Context, plan levelPlan, r resolving) (found, error) {
	l, n, sub := plan.level, plan.number, r.sub
	ownerName := r.d.person(sub.OwnerID).Name
	switch l.ApproverType {
	case SourceManager:
		return managerOf(r, n, ownerName), nil
	case SourceRole:
		return s.roleHolders(ctx, plan, r)
	case SourceOwner:
		if sub.OwnerID == "" {
			return found{source: recordOwnerSource, problem: fmt.Sprintf(noOwnerProblem, n, sub.Kind.word())}, nil
		}
		return found{ids: []string{sub.OwnerID}, source: recordOwnerSource}, nil
	case SourceRecordField:
		field, _ := sub.Kind.recordField(l.RecordField)
		person := sub.personIn(l.RecordField)
		if person == "" {
			return found{source: field.Label, problem: fmt.Sprintf(emptyFieldProblem, n, field.Label, sub.Kind.word())}, nil
		}
		return found{ids: []string{person}, source: field.Label}, nil
	}
	return found{ids: l.ApproverIDs}, nil
}

func managerOf(r resolving, n int, ownerName string) found {
	f := found{source: "line manager of " + ownerName}
	if manager := r.d.byID[r.sub.OwnerID].ManagerID; manager != "" {
		f.ids = []string{manager}
	} else if r.sub.OwnerID != "" {
		f.problem = fmt.Sprintf(noLineManagerProblem, n, ownerName)
	}
	return f
}

func (s *Service) roleHolders(ctx context.Context, plan levelPlan, r resolving) (found, error) {
	role, known := r.roles[plan.level.RoleID]
	if !known {
		return found{problem: fmt.Sprintf(roleGoneProblem, plan.number)}, nil
	}
	holders, err := s.access.RoleHolders(ctx, plan.level.RoleID, r.sub.Plan)
	f := found{ids: holders, source: "role " + role.Name}
	if len(holders) == 0 {
		f.problem = fmt.Sprintf(nobodyHoldsRoleProblem, plan.number, role.Name)
	}
	return f, err
}

// keepApprovers leaves out the candidates who may not approve, with the reason, and falls back on the backup.
func (s *Service) keepApprovers(ctx context.Context, level *RoundLevel, c found, r resolving) error {
	can, err := s.access.CanApprove(ctx, append(slices.Clone(c.ids), c.backupID))
	if err != nil {
		return err
	}
	k := keeping{backupID: c.backupID, usable: c.backupID != "" && exclusion(r, can, c.backupID) == ""}
	for _, id := range c.ids {
		if reason := exclusion(r, can, id); reason != "" {
			level.Excluded = append(level.Excluded, Excluded{Person: r.d.actors([]string{id})[0], Reason: reason})
		} else {
			k.kept = append(k.kept, id)
		}
	}
	slices.SortStableFunc(level.Excluded, func(x, y Excluded) int { return cmp.Compare(reasonRank(x.Reason), reasonRank(y.Reason)) })
	k.assign(level, r.d)
	return nil
}

// keeping is what keepApprovers found: the candidates kept, and the backup approver and whether it is usable.
type keeping struct {
	kept     []string
	backupID string
	usable   bool
}

func (k keeping) assign(level *RoundLevel, d directory) {
	switch {
	case len(k.kept) > 0:
		level.Approvers = d.actors(k.kept)
	case k.usable:
		level.Approvers, level.BackupUsed = d.actors([]string{k.backupID}), true
	default:
		level.NoApprover = true
	}
	if k.usable && !slices.Contains(k.kept, k.backupID) && !level.BackupUsed {
		backup := d.actors([]string{k.backupID})[0]
		level.Backup = &backup
	}
}

func exclusion(r resolving, can map[string]bool, id string) string {
	switch {
	case !r.d.byID[id].Active:
		return ReasonInactive
	case !can[id]:
		return ReasonCannotApprove
	case r.selfBar && id == r.sub.OwnerID:
		return ReasonOwner
	case r.selfBar && id == r.submitter.ID:
		return ReasonSubmitter
	}
	return ""
}

func reasonRank(reason string) int {
	return slices.Index([]string{ReasonInactive, ReasonCannotApprove, ReasonOwner, ReasonSubmitter}, reason)
}

func unique(ids []string) []string {
	var list []string
	for _, id := range ids {
		if id != "" && !slices.Contains(list, id) {
			list = append(list, id)
		}
	}
	return list
}

// firstDecideLevel returns the first level after the level after (0: from the start) whose approvers decide;
// 0 when there is none.
func firstDecideLevel(levels []RoundLevel, after int) int {
	for _, l := range levels {
		if l.Number > after && l.Resolution == ResolutionDecide {
			return l.Number
		}
	}
	return 0
}
