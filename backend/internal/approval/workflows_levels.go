package approval

import (
	"fmt"
	"slices"
	"strings"

	"mapp/internal/people"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// levelsOf returns the levels of a request, normalised (017 plan, Service operations): names and ids without
// surrounding spaces; a level without an approver source has named people, one without a mode "any one
// approves"; approvers are kept only for named people, the role only for role levels, and line-manager
// levels are always "any". Amounts that cannot be read are returned as problems by condition field
// ("levels.<n>.appliesWhen"), so that checkWorkflow reports them with the others. Levels are never nil.
func levelsOf(inputs []LevelInput) ([]Level, map[string]string) {
	levels := make([]Level, 0, len(inputs))
	problems := map[string]string{}
	for i, in := range inputs {
		n := i + 1
		l := levelOf(in)
		var bad bool
		if l.AppliesWhen, bad = conditionOf(in.AppliesWhen); bad {
			problems[fmt.Sprintf("levels.%d.appliesWhen", n)] = fmt.Sprintf(amountsMessage, n)
		}
		if l.AutoApproveWhen, bad = conditionOf(in.AutoApproveWhen); bad {
			problems[fmt.Sprintf("levels.%d.autoApproveWhen", n)] = fmt.Sprintf(amountsMessage, n)
		}
		levels = append(levels, l)
	}
	return levels, problems
}

// levelOf returns the level in sends, normalised as levelsOf says, without its conditions.
func levelOf(in LevelInput) Level {
	l := Level{
		Name: strings.TrimSpace(in.Name), ApproverType: strings.TrimSpace(in.ApproverType), ApproverIDs: []string{},
		RoleID: strings.TrimSpace(in.RoleID), Mode: strings.TrimSpace(in.Mode), BackupApproverID: strings.TrimSpace(in.BackupApproverID),
	}
	if l.ApproverType == "" {
		l.ApproverType = SourcePeople
	}
	if l.Mode == "" || l.ApproverType == SourceManager {
		l.Mode = ModeAny
	}
	if l.ApproverType == SourcePeople && in.ApproverIDs != nil {
		l.ApproverIDs = in.ApproverIDs
	}
	if l.ApproverType != SourceRole {
		l.RoleID = ""
	}
	return l
}

// conditionOf reads a condition as sent: nil stays nil; empty amount texts are no part. bad is true when an
// amount is not a decimal number with at most two decimals.
func conditionOf(in *ConditionInput) (*Condition, bool) {
	if in == nil {
		return nil, false
	}
	c := Condition{BusinessUnitIDs: nonNilIDs(in.BusinessUnitIDs), ActivityTypeIDs: nonNilIDs(in.ActivityTypeIDs)}
	bad := false
	for _, part := range []struct {
		text *string
		into **money.Amount
	}{{in.PlannedCostAtLeast, &c.PlannedCostAtLeast}, {in.PlannedCostBelow, &c.PlannedCostBelow}} {
		if part.text == nil || strings.TrimSpace(*part.text) == "" {
			continue
		}
		amount, err := money.ParseAmount(strings.TrimSpace(*part.text))
		if err != nil {
			bad = true
			continue
		}
		*part.into = &amount
	}
	return &c, bad
}

// nonNilIDs returns ids, or an empty list for nil.
func nonNilIDs(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}

// levelCheck checks the level l at position n; before is the level at that position before an update.
type levelCheck struct {
	problems  *errs.Problems
	rc        ruleContext
	n         int
	l, before Level
}

func (c levelCheck) field(name string) string { return fmt.Sprintf("levels.%d.%s", c.n, name) }

// checkLevel adds every problem of c's level to c.problems (data-model §2), in the order of its fields.
func checkLevel(c levelCheck, amountProblems map[string]string, escalates bool) {
	if tooLong(c.l.Name, maxLevelNameLength) {
		c.problems.Add(c.field("name"), fmt.Sprintf(levelNameTooLongMessage, c.n))
	}
	switch c.l.ApproverType {
	case SourcePeople, SourceManager, SourceRole:
	default:
		c.problems.Add(c.field("approverType"), fmt.Sprintf(chooseSourceMessage, c.n))
	}
	if c.l.ApproverType == SourcePeople {
		c.checkApprovers()
	}
	if c.l.ApproverType == SourceRole {
		c.checkRole()
	}
	if c.l.Mode != ModeAny && c.l.Mode != ModeAll {
		c.problems.Add(c.field("mode"), fmt.Sprintf(chooseModeMessage, c.n))
	}
	c.checkConditions(amountProblems)
	c.checkBackup(escalates)
}

func (c levelCheck) checkApprovers() {
	field := c.field("approverIds")
	if len(c.l.ApproverIDs) == 0 {
		c.problems.Add(field, fmt.Sprintf(chooseApproverMessage, c.n))
	}
	seen, unknown := map[string]bool{}, false
	for _, id := range c.l.ApproverIDs {
		p, found := c.rc.people.byID[id]
		if !found {
			unknown = true
		} else if message := c.approverProblem(p, seen[id]); message != "" {
			c.problems.Add(field, message)
		}
		seen[id] = true
	}
	if unknown {
		c.problems.Add(field, fmt.Sprintf(unknownApproverMessage, c.n))
	}
}

// approverProblem returns the problem of the named approver p, or "": someone kept from before the update may
// stay when inactive or no longer allowed to approve.
func (c levelCheck) approverProblem(p people.Person, listedBefore bool) string {
	added := !slices.Contains(c.before.ApproverIDs, p.ID)
	switch {
	case listedBefore:
		return fmt.Sprintf(listedTwiceMessage, c.n, p.Name)
	case !p.Active && added:
		return fmt.Sprintf(inactiveApproverMessage, c.n, p.Name)
	case !c.rc.canApprove[p.ID] && added:
		return fmt.Sprintf(approverCannotApproveMessage, p.Name)
	}
	return ""
}

func (c levelCheck) checkRole() {
	role, known := c.rc.roles[c.l.RoleID]
	switch {
	case c.l.RoleID == "":
		c.problems.Add(c.field("roleId"), fmt.Sprintf(chooseRoleMessage, c.n))
	case !known:
		c.problems.Add(c.field("roleId"), fmt.Sprintf(roleFromListMessage, c.n))
	case !role.Approves:
		c.problems.Add(c.field("roleId"), fmt.Sprintf(roleCannotApproveMessage, c.n, role.Name))
	}
}

func (c levelCheck) checkConditions(amountProblems map[string]string) {
	for _, cond := range []struct {
		name, emptyMessage string
		condition          *Condition
	}{{"appliesWhen", emptyAppliesWhenMessage, c.l.AppliesWhen}, {"autoApproveWhen", emptyAutoApproveMessage, c.l.AutoApproveWhen}} {
		field := c.field(cond.name)
		if message, bad := amountProblems[field]; bad {
			c.problems.Add(field, message)
			continue
		}
		if cond.condition == nil {
			continue
		}
		if message := conditionProblem(c.rc, c.n, *cond.condition, cond.emptyMessage); message != "" {
			c.problems.Add(field, message)
		}
	}
}

func (c levelCheck) checkBackup(escalates bool) {
	field := c.field("backupApproverId")
	if c.l.BackupApproverID == "" {
		if escalates {
			c.problems.Add(field, fmt.Sprintf(backupRequiredMessage, c.n))
		}
		return
	}
	if message := c.backupProblem(); message != "" {
		c.problems.Add(field, message)
	}
}

// backupProblem returns the problem of the backup approver, or "": one kept from before the update may stay
// when inactive or no longer allowed to approve.
func (c levelCheck) backupProblem() string {
	p, found := c.rc.people.byID[c.l.BackupApproverID]
	changed := c.l.BackupApproverID != c.before.BackupApproverID
	switch {
	case !found:
		return fmt.Sprintf(backupFromListMessage, c.n)
	case !p.Active && changed:
		return fmt.Sprintf(backupInactiveMessage, c.n, p.Name)
	case !c.rc.canApprove[p.ID] && changed:
		return fmt.Sprintf(backupCannotApproveMessage, c.n, p.Name)
	case c.l.ApproverType == SourcePeople && slices.Contains(c.l.ApproverIDs, p.ID):
		return fmt.Sprintf(backupIsApproverMessage, c.n, p.Name)
	}
	return ""
}

// conditionProblem returns the first problem of the condition c of the level n (data-model §2: one message
// per condition field), or "".
func conditionProblem(rc ruleContext, n int, c Condition, emptyMessage string) string {
	switch {
	case c.empty():
		return fmt.Sprintf(emptyMessage, n)
	case c.hasNegativeAmount():
		return fmt.Sprintf(amountsMessage, n)
	case c.hasReversedAmounts():
		return fmt.Sprintf(amountOrderMessage, n)
	case !knownOnce(c.BusinessUnitIDs, rc.names.businessUnits):
		return fmt.Sprintf(businessUnitsMessage, n)
	case !knownOnce(c.ActivityTypeIDs, rc.names.activityTypes):
		return fmt.Sprintf(activityTypesMessage, n)
	}
	return ""
}

func (c Condition) hasNegativeAmount() bool {
	return (c.PlannedCostAtLeast != nil && c.PlannedCostAtLeast.Sign() < 0) || (c.PlannedCostBelow != nil && c.PlannedCostBelow.Sign() < 0)
}

// hasReversedAmounts reports whether c's "at least" amount is not below its "below" amount.
func (c Condition) hasReversedAmounts() bool {
	return c.PlannedCostAtLeast != nil && c.PlannedCostBelow != nil && c.PlannedCostAtLeast.Cmp(*c.PlannedCostBelow) >= 0
}

// knownOnce reports whether every id is one of choices, each listed once. Inactive choices may be listed.
func knownOnce(ids []string, choices map[string]Choice) bool {
	seen := map[string]bool{}
	for _, id := range ids {
		if _, known := choices[id]; !known || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
