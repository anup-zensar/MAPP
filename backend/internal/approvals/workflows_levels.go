package approvals

import (
	"fmt"
	"strings"

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
