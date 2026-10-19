package app

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/plans"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// The adapters below give feature 005's campaigns package what it needs from features 002 and 004
// (campaigns.Deps, D-196); people.Repository is its PersonSource as it is. Three of them stand in for parts
// that features 002 and 004 build at the same time, and are marked INTERIM: when those features are merged,
// the integration replaces them (interimPlanKind by 004's plan kind, interimSettings by 002's
// *settings.Service, which has the same methods) and registers campaignLevels with 004's RegisterLevel.

// campaignPlans is the campaigns.PlanSource over the plans repository (feature 004).
type campaignPlans struct {
	repo plans.Repository
}

// Plan returns the plan id as campaigns needs it.
func (p campaignPlans) Plan(ctx context.Context, id string) (campaigns.PlanFacts, bool, error) {
	plan, found, err := p.repo.Get(ctx, id)
	if err != nil || !found {
		return campaigns.PlanFacts{}, false, err
	}
	return planFacts(plan), true, nil
}

// PlansOfYear returns every plan of year as campaigns needs them.
func (p campaignPlans) PlansOfYear(ctx context.Context, year int) ([]campaigns.PlanFacts, error) {
	list, err := p.repo.PlansOfYear(ctx, year)
	if err != nil {
		return nil, err
	}
	facts := make([]campaigns.PlanFacts, 0, len(list))
	for _, plan := range list {
		facts = append(facts, planFacts(plan))
	}
	return facts, nil
}

// planFacts turns a stored plan into campaigns' PlanFacts: no owner and no budget become nil, the budget
// USD money.
func planFacts(p plans.Plan) campaigns.PlanFacts {
	facts := campaigns.PlanFacts{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: string(p.Status), ModelID: p.ModelID, Attributes: p.Attributes,
	}
	if p.OwnerID != "" {
		owner := p.OwnerID
		facts.OwnerID = &owner
	}
	if p.Budget != nil {
		budget := money.USD(*p.Budget)
		facts.Budget = &budget
	}
	return facts
}

// campaignLevels are the campaign and initiative levels for feature 004's level registry (D-195, 005 plan
// "Alignment with feature 004 as planned"): they add a plan's campaigns, initiatives, campaign budgets and
// the campaigns and initiatives that block publishing to its facts. INTERIM: 004's
// planService.RegisterLevel registers them once 004 is merged; plan copies (Copy, through
// CopyPlanContents) arrive with US6.
func campaignLevels(svc *campaigns.Service) []plans.Level {
	addFacts := func(level string) func(ctx context.Context, planIDs []string, facts map[string]*plans.Facts) error {
		return func(ctx context.Context, planIDs []string, facts map[string]*plans.Facts) error {
			rollups, err := svc.PlanRollups(ctx, planIDs)
			if err != nil {
				return err
			}
			for _, planID := range planIDs {
				r, f := rollups[planID], facts[planID]
				if level == "campaign" {
					f.Campaigns += r.Campaigns
					f.CampaignBudgets = f.CampaignBudgets.Add(r.AllocatedToCampaigns.Amount)
				} else {
					f.Initiatives += r.Initiatives
				}
				for _, p := range r.PublishProblems {
					if p.Record.Type == level {
						f.NeedsAttention = append(f.NeedsAttention, plans.AttentionItem{Record: p.Record, Missing: p.Missing, BlocksPublish: true})
					}
				}
			}
			return nil
		}
	}
	return []plans.Level{
		{Name: "campaign", AddFacts: addFacts("campaign")},
		// The campaign level copies the initiatives with their campaigns, so this level's Copy does nothing.
		{Name: "initiative", AddFacts: addFacts("initiative"), Copy: func(context.Context, *plans.CopyJob) error { return nil }},
	}
}

// interimPlanKind is a minimal history kind "plan", the parent of the campaign kind, over the plans
// repository: history needs it to log campaigns with their path and to delete them. INTERIM: feature 004
// registers the real plan kind; this one is then removed from app.go.
func interimPlanKind(repo plans.Repository) history.Kind {
	return history.Kind{
		Name:     "plan",
		Singular: "plan",
		Plural:   "plans",
		Snapshot: func(ctx context.Context, id string) (history.Snapshot, bool, error) {
			plan, found, err := repo.Get(ctx, id)
			if err != nil || !found {
				return history.Snapshot{}, false, err
			}
			data, err := json.Marshal(plan)
			if err != nil {
				return history.Snapshot{}, false, err
			}
			return history.Snapshot{ID: plan.ID, Name: plan.Name, Data: data}, true, nil
		},
		Remove: repo.Delete,
		Put: func(ctx context.Context, s history.Snapshot) error {
			var plan plans.Plan
			if err := json.Unmarshal(s.Data, &plan); err != nil {
				return err
			}
			return repo.Put(ctx, plan)
		},
	}
}

// Messages of feature 002's attribute checks used by interimSettings (002 plan, "Check results").
const (
	attributeGoneMessage   = "This Shell attribute no longer exists — reload the page."
	notLinkedOptionMessage = "'%s' doesn't belong to the chosen %s — choose another %s."
)

// interimSettings is campaigns.Settings over the Settings repository, with the part of feature 002's
// attribute rules that campaigns relies on: unknown attributes and dependent options that belong to no
// chosen parent option are refused (R8), needed-to-publish attributes that are empty are missing (with
// D-034 and the business-unit condition, R9), and values are described for the log. INTERIM: feature 002's
// *settings.Service has the same methods with every rule, and replaces it once 002 is merged.
type interimSettings struct {
	repo settings.Repository
}

// CheckAttributes refuses unknown attributes, attributes of another level and newly chosen dependent
// options that belong to no chosen parent option; it stores the values as sent.
func (s interimSettings) CheckAttributes(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error) {
	attributes, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return settings.AttributeResult{}, err
	}
	byID := map[string]settings.Attribute{}
	for _, a := range attributes {
		byID[a.ID] = a
	}
	problems := []errs.FieldError{}
	for _, id := range slices.Sorted(maps.Keys(in.Values)) {
		attribute, known := byID[id]
		if !known || attribute.Level != level {
			problems = append(problems, errs.FieldError{Field: "attributes." + id, Message: attributeGoneMessage})
			continue
		}
		if attribute.ParentAttributeID == "" || !attribute.IsChoice() {
			continue
		}
		parentChosen := optionIDs(in.Values[attribute.ParentAttributeID])
		for _, optionID := range optionIDs(in.Values[id]) {
			if slices.Contains(optionIDs(in.Previous[id]), optionID) {
				continue // held before: it may stay
			}
			option, found, err := s.repo.GetOption(ctx, optionID)
			if err != nil {
				return settings.AttributeResult{}, err
			}
			if found && !slices.ContainsFunc(option.ParentOptionIDs, func(p string) bool { return slices.Contains(parentChosen, p) }) {
				parent := byID[attribute.ParentAttributeID]
				message := fmt.Sprintf(notLinkedOptionMessage, option.Label, midSentence(parent.Label), midSentence(attribute.Label))
				problems = append(problems, errs.FieldError{Field: "attributes." + id, Message: message})
				break
			}
		}
	}
	missing, err := s.MissingAttributes(ctx, level, in.Values, in.BusinessUnit)
	if err != nil {
		return settings.AttributeResult{}, err
	}
	values := maps.Clone(in.Values)
	if values == nil {
		values = settings.AttributeValues{}
	}
	return settings.AttributeResult{Problems: problems, Cleared: []settings.ClearedValue{}, Missing: missing, Values: values}, nil
}

// MissingAttributes lists the active, shown, needed-to-publish attributes of level that are empty, in
// display order. A dependent attribute whose chosen parent options have no options of their own is not
// missing (D-034); with no parent option chosen it is.
func (s interimSettings) MissingAttributes(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error) {
	attributes, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	attributes = slices.DeleteFunc(attributes, func(a settings.Attribute) bool {
		visible := len(a.Condition) == 0 || slices.ContainsFunc(a.Condition, func(o string) bool { return slices.Contains(businessUnit, o) })
		return a.Level != level || !a.Active || a.Marking != settings.MarkingNeededToPublish || !visible
	})
	slices.SortStableFunc(attributes, func(x, y settings.Attribute) int { return x.Position - y.Position })

	missing := []settings.MissingField{}
	for _, a := range attributes {
		if !emptyValue(values[a.ID]) {
			continue
		}
		if a.ParentAttributeID != "" {
			nothingToChoose, err := s.noOptionsFor(ctx, a, optionIDs(values[a.ParentAttributeID]))
			if err != nil {
				return nil, err
			}
			if nothingToChoose {
				continue
			}
		}
		missing = append(missing, settings.MissingField{Field: "attributes." + a.ID, Label: a.Label})
	}
	return missing, nil
}

// noOptionsFor reports whether parent options are chosen and none of the active options of the dependent
// attribute a belongs to them: a field nobody can fill never blocks publishing (D-034).
func (s interimSettings) noOptionsFor(ctx context.Context, a settings.Attribute, parentChosen []string) (bool, error) {
	if len(parentChosen) == 0 {
		return false, nil
	}
	options, err := s.repo.OptionsOf(ctx, a.ID)
	if err != nil {
		return false, err
	}
	linked := slices.ContainsFunc(options, func(o settings.Option) bool {
		return o.Active && slices.ContainsFunc(o.ParentOptionIDs, func(p string) bool { return slices.Contains(parentChosen, p) })
	})
	return !linked, nil
}

// BusinessUnit returns the option ids of the plan's Business unit attribute.
func (s interimSettings) BusinessUnit(ctx context.Context, planValues settings.AttributeValues) ([]string, error) {
	attributes, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range attributes {
		if a.Role == settings.RoleBusinessUnit {
			return optionIDs(planValues[a.ID]), nil
		}
	}
	return []string{}, nil
}

// DescribeAttribute gives the label of the attribute of a field "attributes.<id>" and its value as people
// read it: option labels joined with ", ", Yes or No, or the text.
func (s interimSettings) DescribeAttribute(ctx context.Context, field string, value any) (string, string, error) {
	id, isAttribute := strings.CutPrefix(field, "attributes.")
	if !isAttribute {
		return "", "", nil
	}
	attribute, found, err := s.repo.GetAttribute(ctx, id)
	if err != nil || !found {
		return "", "", err
	}
	switch v := value.(type) {
	case bool:
		if v {
			return attribute.Label, "Yes", nil
		}
		return attribute.Label, "No", nil
	case string:
		if !attribute.IsChoice() {
			return attribute.Label, "", nil // the text as it is
		}
		label, err := s.optionLabel(ctx, v)
		return attribute.Label, label, err
	case []any:
		labels := make([]string, 0, len(v))
		for _, item := range v {
			id, _ := item.(string)
			label, err := s.optionLabel(ctx, id)
			if err != nil {
				return "", "", err
			}
			labels = append(labels, label)
		}
		return attribute.Label, strings.Join(labels, ", "), nil
	}
	return attribute.Label, "", nil
}

// optionLabel returns the label of the option id, or the id when it is gone.
func (s interimSettings) optionLabel(ctx context.Context, id string) (string, error) {
	option, found, err := s.repo.GetOption(ctx, id)
	if err != nil || !found {
		return id, err
	}
	return option.Label, nil
}

// optionIDs reads a choice value: one option id or a list of them; nothing for anything else.
func optionIDs(value json.RawMessage) []string {
	var list []string
	if json.Unmarshal(value, &list) == nil {
		return list
	}
	var one string
	if json.Unmarshal(value, &one) == nil && one != "" {
		return []string{one}
	}
	return []string{}
}

// emptyValue reports whether an attribute value is empty: missing, null, "", [] or {}.
func emptyValue(value json.RawMessage) bool {
	switch strings.TrimSpace(string(value)) {
	case "", "null", `""`, "[]", "{}":
		return true
	}
	return false
}

// midSentence writes a label inside a sentence: the first letter in lower case when the second is lower
// case, so "Region" becomes "region" and "GL code" stays.
func midSentence(label string) string {
	first, size := utf8.DecodeRuneInString(label)
	second, _ := utf8.DecodeRuneInString(label[size:])
	if unicode.IsLower(second) {
		return string(unicode.ToLower(first)) + label[size:]
	}
	return label
}
