package app

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"mapp/internal/allwork"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// spreadWords and goalStageWords return the stored values as the plan and line-item forms show them.
func spreadWords() map[string]string {
	return map[string]string{
		"evenly": "Evenly", "firstMonth": "All in first month", "lastMonth": "All in last month", "custom": "Custom",
	}
}

func goalStageWords() map[plan.GoalStage]string {
	return map[plan.GoalStage]string{plan.StageINQ: "INQ", plan.StageMQL: "MQL", plan.StageRevenue: "Revenue"}
}

// planFields returns the plan-level columns of p (allwork data model §6.2): its stored fields with the names
// of its model, teams (by name, as 004 shows them) and default finance budget, and — only when the view shows
// them — the plan goals 004 works out from the model.
func (y *yearReader) planFields(ctx context.Context, p plan.Plan) (map[string]allwork.Value, error) {
	year := int64(p.Year)
	fields := map[string]allwork.Value{
		"plan.year":          {Number: &year},
		"plan.strategic":     {Yes: &p.AOP},
		"plan.description":   {Text: p.Description},
		"plan.defaultSpread": {Text: spreadWords()[string(p.DefaultSpread)]},
		"plan.bcpir":         {Yes: &p.BCPIR},
		"plan.goalStage":     {Text: goalStageWords()[p.GoalStage]},
	}
	if err := y.addPlanReferences(ctx, fields, p); err != nil {
		return nil, err
	}
	if err := y.addPlanGoals(ctx, fields, p.ID); err != nil {
		return nil, err
	}
	addGoalAmount(fields, p)
	y.addAttributes(fields, settings.LevelPlan, p.Attributes)
	return fields, nil
}

func (y *yearReader) addPlanReferences(ctx context.Context, fields map[string]allwork.Value, p plan.Plan) error {
	if m, found, err := y.s.settings.Model(ctx, p.ModelID); err != nil {
		return err
	} else if found {
		fields["plan.model"] = allwork.Value{Text: m.Name, Inactive: !m.Active}
	}
	teams, err := y.planTeams(ctx, p.TeamIDs)
	if err != nil {
		return err
	}
	fields["plan.teams"] = allwork.Value{Choices: teams}
	if p.DefaultFinanceBudgetID == "" {
		return nil
	}
	b, found, err := y.s.settings.FinanceBudget(ctx, p.DefaultFinanceBudgetID)
	if err != nil {
		return err
	}
	if found {
		fields["plan.defaultFinanceBudget"] = allwork.Value{Text: b.Name}
	}
	return nil
}

// planTeams leaves out a team that no longer resolves.
func (y *yearReader) planTeams(ctx context.Context, teamIDs []string) ([]allwork.Choice, error) {
	refs, err := y.s.people.TeamRefs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	teams := []allwork.Choice{}
	for _, id := range teamIDs {
		if t, found := refs[id]; found {
			teams = append(teams, allwork.Choice{Label: t.Name, Inactive: !t.Active})
		}
	}
	slices.SortFunc(teams, func(a, b allwork.Choice) int {
		return strings.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})
	return teams, nil
}

// addPlanGoals reads the plan only when the view shows its goals: 004 works them out plan by plan.
func (y *yearReader) addPlanGoals(ctx context.Context, fields map[string]allwork.Value, planID string) error {
	if !y.needs("plan.goalMql", "plan.goalRevenue") {
		return nil
	}
	view, err := y.s.plans.Get(ctx, actor.System, planID) // actor.System sees amounts in USD
	if err != nil {
		return err
	}
	fields["plan.goalMql"] = allwork.Value{Number: view.Goals.Plan.MQL}
	if view.Goals.Plan.Revenue != nil {
		fields["plan.goalRevenue"] = allwork.Value{USD: &view.Goals.Plan.Revenue.Amount}
	}
	return nil
}

// addGoalAmount leaves out a goal it cannot read: an amount for a revenue goal, a whole number otherwise.
func addGoalAmount(fields map[string]allwork.Value, p plan.Plan) {
	if p.GoalAmount == "" {
		return
	}
	if p.GoalStage == plan.StageRevenue {
		if amount, err := money.ParseAmount(p.GoalAmount); err == nil {
			fields["plan.goalAmount"] = allwork.Value{USD: &amount}
		}
	} else if n, err := strconv.ParseInt(p.GoalAmount, 10, 64); err == nil {
		fields["plan.goalAmount"] = allwork.Value{Number: &n}
	}
}
