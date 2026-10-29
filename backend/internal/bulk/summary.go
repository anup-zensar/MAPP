package bulk

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"mapp/internal/platform/errs"
)

// consequenceSums adds up what changes below the campaigns that land in another plan (FR-018).
type consequenceSums struct {
	fromPlan, toPlan, model string
	typesKept               int
	hidden, shown           map[string]int
}

// summarise counts the records a delete takes, before it deletes them (FR-019).
func (r *runner) summarise(ctx context.Context, records []Record) error {
	summary := DeleteSummary{}
	for _, rec := range records {
		below, err := r.s.planning.Below(ctx, rec)
		if err != nil {
			return err
		}
		switch rec.Level {
		case LevelCampaign:
			summary.Selected.Campaigns++
		case LevelInitiative:
			summary.Selected.Initiatives++
		case LevelActivity:
			summary.Selected.Activities++
		}
		summary.Below.Campaigns += below.Counts.Campaigns
		summary.Below.Initiatives += below.Counts.Initiatives
		summary.Below.Activities += below.Counts.Activities
		summary.Below.LineItems += below.Counts.LineItems
		summary.SubmittedOrApproved += below.InApproval
	}
	r.result.DeleteSummary = &summary
	return nil
}

// consequences notes, before a campaign is copied or moved to another plan, what changes below it there.
func (r *runner) consequences(ctx context.Context, rec Record) error {
	if rec.Level != LevelCampaign || rec.PlanID == r.req.TargetID {
		return nil
	}
	c, err := r.s.planning.Consequences(ctx, r.a, rec, r.req.TargetID)
	if err != nil {
		return err
	}
	sums := &r.effects
	if sums.hidden == nil {
		sums.hidden, sums.shown = map[string]int{}, map[string]int{}
	}
	sums.fromPlan, sums.toPlan, sums.model = rec.PlanName, c.OtherPlanName, c.ModelName
	sums.typesKept += c.TypesNotAllowed
	for _, group := range c.GroupsHidden {
		sums.hidden[group]++
	}
	for _, group := range c.GroupsShown {
		sums.shown[group]++
	}
	return nil
}

func (c consequenceSums) effects(action Kind) []Effect {
	effects := []Effect{}
	if c.toPlan == "" || (action != ActionCopy && action != ActionMove) {
		return effects
	}
	if c.typesKept > 0 {
		effects = append(effects, Effect{Kind: "typeNotAllowedKept", Count: c.typesKept,
			Text: fmt.Sprintf(effectTypesKept, c.typesKept, c.model)})
	}
	for _, group := range slices.Sorted(maps.Keys(c.hidden)) {
		effects = append(effects, Effect{Kind: "groupsHidden", Count: c.hidden[group],
			Text: fmt.Sprintf(effectHidden, group, c.hidden[group])})
	}
	for _, group := range slices.Sorted(maps.Keys(c.shown)) {
		effects = append(effects, Effect{Kind: "groupsAdded", Count: c.shown[group],
			Text: fmt.Sprintf(effectShown, group, c.shown[group])})
	}
	return append(effects, Effect{Kind: "totalsUpdate", Count: 2, Text: fmt.Sprintf(effectTotals, c.fromPlan, c.toPlan)})
}

// reasonsOf turns a refusal of an owning operation into plain reasons: the missing fields of a submit check
// grouped per record, then each field's message, or else the message.
func reasonsOf(rec Record, e *errs.Error) []string {
	reasons := []string{}
	if text := missingText(rec, e.Missing); text != "" {
		reasons = append(reasons, text)
	}
	for _, f := range e.Fields {
		if !slices.Contains(reasons, f.Message) {
			reasons = append(reasons, f.Message)
		}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, e.Message)
	}
	return reasons
}

// missingText reads "3 fields needed: Owner, Company code; line item L-1004: GL code".
func missingText(rec Record, groups []errs.ReadinessGroup) string {
	parts := []string{}
	for _, g := range groups {
		labels := make([]string, 0, len(g.Fields))
		for _, f := range g.Fields {
			labels = append(labels, f.Label)
		}
		if len(labels) == 0 {
			continue
		}
		if g.Record.ID == rec.ID {
			format := reasonFieldsNeeded
			if len(labels) == 1 {
				parts = append(parts, fmt.Sprintf(reasonOneFieldNeeded, labels[0]))
				continue
			}
			parts = append(parts, fmt.Sprintf(format, len(labels), strings.Join(labels, ", ")))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s: %s", g.Record.Type, g.Record.ID, strings.Join(labels, ", ")))
	}
	return strings.Join(parts, "; ")
}
