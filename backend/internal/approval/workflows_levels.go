package approval

import (
	"fmt"
	"slices"
	"strings"

	"mapp/internal/people"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

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

func levelOf(in LevelInput) Level {
	l := Level{
		Name: strings.TrimSpace(in.Name), ApproverType: strings.TrimSpace(in.ApproverType), ApproverIDs: []string{},
		RoleID: strings.TrimSpace(in.RoleID), Mode: strings.TrimSpace(in.Mode), BackupApproverID: strings.TrimSpace(in.BackupApproverID),
		RecordField: strings.TrimSpace(in.RecordField),
	}
	if l.ApproverType == "" {
		l.ApproverType = SourcePeople
	}
	if l.Mode == "" || singlePerson(l.ApproverType) {
		l.Mode = ModeAny
	}
	if l.ApproverType == SourcePeople && in.ApproverIDs != nil {
		l.ApproverIDs = in.ApproverIDs
	}
	if l.ApproverType != SourceRole {
		l.RoleID = ""
	}
	if l.ApproverType != SourceRecordField {
		l.RecordField = ""
	}
	return l
}

// singlePerson reports whether a source names one person, so its level is always "any one approves".
func singlePerson(source string) bool {
	return source == SourceManager || source == SourceOwner || source == SourceRecordField
}

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

func nonNilIDs(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}

type levelCheck struct {
	problems  *errs.Problems
	rc        ruleContext
	n         int
	l, before Level
	kind      SubjectKind
}

func (c levelCheck) field(name string) string { return fmt.Sprintf("levels.%d.%s", c.n, name) }

func checkLevel(c levelCheck, amountProblems map[string]string, escalates bool) {
	if tooLong(c.l.Name, maxLevelNameLength) {
		c.problems.Add(c.field("name"), fmt.Sprintf(levelNameTooLongMessage, c.n))
	}
	switch c.l.ApproverType {
	case SourcePeople:
		c.checkApprovers()
	case SourceRole:
		c.checkRole()
	case SourceRecordField:
		c.checkRecordField()
	case SourceManager, SourceOwner:
	default:
		c.problems.Add(c.field("approverType"), fmt.Sprintf(chooseSourceMessage, c.n))
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
		if message := c.conditionProblem(*cond.condition, cond.emptyMessage); message != "" {
			c.problems.Add(field, message)
		}
	}
}

// conditionProblem adds to the rules of every condition that only activity workflows name activity facts.
func (c levelCheck) conditionProblem(condition Condition, emptyMessage string) string {
	message := conditionProblem(c.rc, c.n, condition, emptyMessage)
	if message == "" && c.kind != KindActivity && condition.namesActivityFacts() {
		return fmt.Sprintf(activityConditionsOnlyMessage, c.n)
	}
	return message
}

func (c levelCheck) checkRecordField() {
	field := c.field("recordField")
	if _, known := c.kind.recordField(c.l.RecordField); known {
		return
	}
	if c.kind == KindActivity {
		c.problems.Add(field, fmt.Sprintf(activityNoRecordFieldMessage, c.n))
		return
	}
	c.problems.Add(field, fmt.Sprintf(chooseRecordFieldMessage, c.n))
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

func (c Condition) hasReversedAmounts() bool {
	return c.PlannedCostAtLeast != nil && c.PlannedCostBelow != nil && c.PlannedCostAtLeast.Cmp(*c.PlannedCostBelow) >= 0
}

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
