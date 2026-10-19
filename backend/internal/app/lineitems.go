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

	"northstar/internal/activities"
	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/lineitems"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
	"northstar/internal/platform/web"
	"northstar/internal/settings"
)

// The wiring of feature 007, line items: the adapters that implement the dependency interfaces of package
// lineitems over the other features (D-196). They read the other features through their repositories, inside
// the caller's transaction. Where a feature's service offers the same rule (002's attribute checks, 006's
// RestartApproval and history kind "activity"), the adapter is a stand-in until that service is wired here.

// addLineItems builds the line-items service, registers its history kind and serves its routes.
func addLineItems(router *web.Router, signedIn web.Wrap, db *store.DB, generator *ids.Generator, hist *history.Service, now clock.Clock) error {
	settingsRepo := settings.NewJSONRepository()
	activityRepo := activities.NewJSONRepository()
	activityFacts := lineItemActivities{
		activities: activityRepo, campaigns: campaigns.NewJSONRepository(), plans: plans.NewJSONRepository(),
		settings: settingsRepo, history: hist, now: now,
	}
	deps := lineitems.Deps{
		Activities:     activityFacts,
		Currencies:     lineItemSettings{repo: settingsRepo},
		FinanceBudgets: lineItemSettings{repo: settingsRepo},
		Attributes:     lineItemAttributes{repo: settingsRepo},
		People:         lineItemPeople{repo: people.NewJSONRepository()},
	}
	service := lineitems.NewService(db, lineitems.NewJSONRepository(), generator, hist, deps, now)

	// STAND-IN until feature 006 registers its history kind "activity" here: the line-item kind needs its
	// parent kind registered first. Remove this registration when 006's is wired.
	if err := hist.Register(standInActivityKind(activityRepo)); err != nil {
		return err
	}
	for _, kind := range service.Kinds() {
		if err := hist.Register(kind); err != nil {
			return err
		}
	}
	lineitems.NewHandler(service).Routes(router, signedIn)
	return nil
}

// lineItemActivities implements lineitems.Activities over the activities (006), campaigns (005), plans (004)
// and Settings (002) repositories.
type lineItemActivities struct {
	activities activities.Repository
	campaigns  campaigns.Repository
	plans      plans.Repository
	settings   settings.Repository
	history    *history.Service
	now        clock.Clock
}

// Facts returns the activity id with its initiative, campaign and plan, and false when it is not stored.
func (x lineItemActivities) Facts(ctx context.Context, id string) (lineitems.ActivityFacts, bool, error) {
	act, found, err := x.activities.Get(ctx, id)
	if err != nil || !found {
		return lineitems.ActivityFacts{}, false, err
	}
	initiative, found, err := x.campaigns.GetInitiative(ctx, act.InitiativeID)
	if err != nil || !found {
		return lineitems.ActivityFacts{}, false, missing(err, "initiative", act.InitiativeID)
	}
	campaign, found, err := x.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return lineitems.ActivityFacts{}, false, missing(err, "campaign", initiative.CampaignID)
	}
	plan, found, err := x.plans.Get(ctx, campaign.PlanID)
	if err != nil || !found {
		return lineitems.ActivityFacts{}, false, missing(err, "plan", campaign.PlanID)
	}
	f := lineitems.ActivityFacts{
		ID: act.ID, Name: act.Name, OwnerID: act.OwnerID, Status: string(act.Status),
		StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
		Path: []errs.RecordRef{
			{Type: "plan", ID: plan.ID, Name: plan.Name},
			{Type: "campaign", ID: campaign.ID, Name: campaign.Name},
			{Type: "initiative", ID: initiative.ID, Name: initiative.Name},
		},
		PlanYear:      plan.Year,
		DefaultSpread: lineitems.SpreadMethod(plan.DefaultSpread),
	}
	if act.PlannedCost != nil {
		planned := act.PlannedCost.Amount
		f.PlannedCost = &planned
	}
	if plan.DefaultFinanceBudgetID != "" {
		budget, found, err := x.settings.GetFinanceBudget(ctx, plan.DefaultFinanceBudgetID)
		if err != nil {
			return lineitems.ActivityFacts{}, false, err
		}
		if found {
			f.DefaultFinanceBudget = budget.Name
		}
	}
	if f.BusinessUnit, err = businessUnit(ctx, x.settings, plan.Attributes); err != nil {
		return lineitems.ActivityFacts{}, false, err
	}
	return f, true, nil
}

// RestartApproval sends a Submitted or Approved activity back to Created and logs the reasons.
//
// STAND-IN until feature 006 is wired here: then call (*activities.Service).RestartApproval, which also
// tells feature 008 to end the open approval round.
func (x lineItemActivities) RestartApproval(ctx context.Context, a actor.Actor, id string, reasons []string) error {
	act, found, err := x.activities.Get(ctx, id)
	if err != nil || !found {
		return missing(err, "activity", id)
	}
	if act.Status != activities.StatusSubmitted && act.Status != activities.StatusApproved {
		return nil
	}
	before := act
	act.Status = activities.StatusCreated
	act.Touch(a, x.now())
	if err := x.activities.Put(ctx, act); err != nil {
		return err
	}
	changes, err := history.Diff(before, act)
	if err != nil {
		return err
	}
	_, err = x.history.Log(ctx, a, history.LogInput{
		Kind: "activity", RecordID: act.ID, RecordName: act.Name, Action: history.ActionStatusChanged,
		Verb: "restarted approval of", Changes: changes, Details: strings.Join(reasons, " "),
	})
	return err
}

// missing returns err, or, when there is none, an error saying that a record the data needs is not stored.
func missing(err error, kind, id string) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("app: the %s %s is not stored", kind, id)
}

// businessUnit returns the plan's Business unit option ids, read from its values of the Business unit
// attribute (D-055).
func businessUnit(ctx context.Context, repo settings.Repository, planValues settings.AttributeValues) ([]string, error) {
	all, err := repo.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(all, func(attr settings.Attribute) bool { return attr.Role == settings.RoleBusinessUnit })
	if i < 0 {
		return []string{}, nil
	}
	ids, _ := choiceIDs(planValues[all[i].ID])
	return ids, nil
}

// standInActivityKind is a minimal history kind "activity", top level, over the activities repository.
//
// STAND-IN until feature 006 registers its own kind "activity" (with its path, rules and child kinds).
func standInActivityKind(repo activities.Repository) history.Kind {
	return history.Kind{
		Name: "activity", Singular: "activity", Plural: "activities",
		Snapshot: func(ctx context.Context, id string) (history.Snapshot, bool, error) {
			act, found, err := repo.Get(ctx, id)
			if err != nil || !found {
				return history.Snapshot{}, false, err
			}
			data, err := json.Marshal(act)
			if err != nil {
				return history.Snapshot{}, false, err
			}
			return history.Snapshot{ID: act.ID, Name: act.Name, Data: data}, true, nil
		},
		Remove: repo.Delete,
		Put: func(ctx context.Context, s history.Snapshot) error {
			var act activities.Activity
			if err := json.Unmarshal(s.Data, &act); err != nil {
				return err
			}
			return repo.Put(ctx, act)
		},
	}
}

// lineItemSettings implements lineitems.Currencies and lineitems.FinanceBudgets over the Settings
// repository (002). Feature 002's readers Currency and Currencies have the same signatures.
type lineItemSettings struct {
	repo settings.Repository
}

// Currency returns the currency with code, active or not.
func (x lineItemSettings) Currency(ctx context.Context, code string) (settings.Currency, bool, error) {
	return x.repo.GetCurrency(ctx, code)
}

// Currencies returns every currency, active or not.
func (x lineItemSettings) Currencies(ctx context.Context) ([]settings.Currency, error) {
	return x.repo.ListCurrencies(ctx)
}

// Item returns the finance budget item id with its budget's year and name.
func (x lineItemSettings) Item(ctx context.Context, id string) (lineitems.BudgetItem, bool, error) {
	item, found, err := x.repo.GetFinanceBudgetItem(ctx, id)
	if err != nil || !found {
		return lineitems.BudgetItem{}, false, err
	}
	budget, found, err := x.repo.GetFinanceBudget(ctx, item.BudgetID)
	if err != nil || !found {
		return lineitems.BudgetItem{}, false, missing(err, "finance budget", item.BudgetID)
	}
	return budgetItem(budget, item), true, nil
}

// Items returns every finance budget item with its budget's year and name.
func (x lineItemSettings) Items(ctx context.Context) ([]lineitems.BudgetItem, error) {
	budgets, err := x.repo.ListFinanceBudgets(ctx)
	if err != nil {
		return nil, err
	}
	items, err := x.repo.ListFinanceBudgetItems(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]lineitems.BudgetItem, 0, len(items))
	for _, item := range items {
		if i := slices.IndexFunc(budgets, func(b settings.FinanceBudget) bool { return b.ID == item.BudgetID }); i >= 0 {
			list = append(list, budgetItem(budgets[i], item))
		}
	}
	return list, nil
}

// budgetItem returns item of budget as line items need it.
func budgetItem(budget settings.FinanceBudget, item settings.FinanceBudgetItem) lineitems.BudgetItem {
	return lineitems.BudgetItem{
		ID: item.ID, Year: budget.Year, BudgetName: budget.Name, Name: item.Name, Code: item.Code, Active: item.Active,
	}
}

// lineItemPeople implements lineitems.People over the people repository (003).
type lineItemPeople struct {
	repo people.Repository
}

// Person returns the person id as records show them, active or not.
func (x lineItemPeople) Person(ctx context.Context, id string) (people.PersonRef, bool, error) {
	person, found, err := x.repo.Get(ctx, id)
	if err != nil || !found {
		return people.PersonRef{}, false, err
	}
	return person.Ref(), true, nil
}

// lineItemAttributes implements lineitems.Attributes over the Settings repository: the rules of 002's
// CheckAttributes, MissingAttributes and DescribeAttribute (002 data model §9) for the choice attributes of
// level lineItem, which are the ones line items have.
//
// STAND-IN until feature 002's service is wired here: then call its CheckAttributes(ctx,
// settings.LevelLineItem, in), MissingAttributes(ctx, settings.LevelLineItem, values, businessUnit) and
// DescribeAttribute(ctx, field, value).
type lineItemAttributes struct {
	repo settings.Repository
}

// Check checks the values of a save: a known line-item attribute; for a choice attribute, options of that
// attribute, a new choice active, and a dependent value (GL code) fitting its parent's value (Spend type,
// R11). A dependent value that was already stored and no longer fits its changed parent is cleared (002
// FR-025). Empty values are left out of the values to store.
func (x lineItemAttributes) Check(ctx context.Context, in settings.AttributeInput) (settings.AttributeResult, error) {
	result := settings.AttributeResult{
		Problems: []errs.FieldError{}, Cleared: []settings.ClearedValue{}, Missing: []settings.MissingField{},
		Values: settings.AttributeValues{},
	}
	attributes, err := x.attributes(ctx)
	if err != nil {
		return settings.AttributeResult{}, err
	}
	for _, id := range slices.Sorted(maps.Keys(in.Values)) {
		raw := in.Values[id]
		if isEmptyValue(raw) {
			continue
		}
		field := "attributes." + id
		i := slices.IndexFunc(attributes, func(attr settings.Attribute) bool { return attr.ID == id })
		if i < 0 {
			result.Problems = append(result.Problems, errs.FieldError{Field: field, Message: "This Shell attribute no longer exists — reload the page."})
			continue
		}
		attr := attributes[i]
		if !attr.IsChoice() {
			result.Values[id] = raw
			continue
		}
		message, cleared, err := x.checkChoice(ctx, attr, in, attributes)
		switch {
		case err != nil:
			return settings.AttributeResult{}, err
		case message != "":
			result.Problems = append(result.Problems, errs.FieldError{Field: field, Message: message})
		case cleared:
			result.Cleared = append(result.Cleared, settings.ClearedValue{AttributeID: id, Label: attr.Label})
		default:
			result.Values[id] = raw
		}
	}
	return result, nil
}

// checkChoice checks the value of the choice attribute attr in in.Values and returns the message of its
// problem ("" when fine), or cleared when it was stored before and no longer fits its changed parent.
func (x lineItemAttributes) checkChoice(ctx context.Context, attr settings.Attribute, in settings.AttributeInput, attributes []settings.Attribute) (string, bool, error) {
	optionIDs, ok := choiceIDs(in.Values[attr.ID])
	previous, _ := choiceIDs(in.Previous[attr.ID])
	label := midSentence(attr.Label)
	if !ok {
		return fmt.Sprintf("Choose one of the options of %s.", label), false, nil
	}
	parentIDs, _ := choiceIDs(in.Values[attr.ParentAttributeID])
	for _, optionID := range optionIDs {
		option, found, err := x.repo.GetOption(ctx, optionID)
		if err != nil {
			return "", false, err
		}
		if !found || option.AttributeID != attr.ID {
			return fmt.Sprintf("Choose one of the options of %s.", label), false, nil
		}
		if !option.Active && !slices.Contains(previous, optionID) {
			return fmt.Sprintf("'%s' is inactive — choose another option.", option.Label), false, nil
		}
		fits := attr.ParentAttributeID == "" || len(parentIDs) == 0 ||
			slices.ContainsFunc(option.ParentOptionIDs, func(id string) bool { return slices.Contains(parentIDs, id) })
		if fits {
			continue
		}
		if slices.Equal(optionIDs, previous) { // kept from before, but its parent changed: cleared on save
			return "", true, nil
		}
		parent := "parent"
		if i := slices.IndexFunc(attributes, func(a settings.Attribute) bool { return a.ID == attr.ParentAttributeID }); i >= 0 {
			parent = midSentence(attributes[i].Label)
		}
		return fmt.Sprintf("'%s' doesn't belong to the chosen %s — choose another %s.", option.Label, parent, label), false, nil
	}
	return "", false, nil
}

// Missing lists the visible, active, needed-to-submit line-item attributes without a value, in display
// order. A dependent attribute whose parent value has no options set up is not counted (D-034).
func (x lineItemAttributes) Missing(ctx context.Context, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error) {
	attributes, err := x.attributes(ctx)
	if err != nil {
		return nil, err
	}
	missingFields := []settings.MissingField{}
	for _, attr := range attributes {
		visible := len(attr.Condition) == 0 ||
			slices.ContainsFunc(attr.Condition, func(id string) bool { return slices.Contains(businessUnit, id) })
		if !attr.Active || attr.Marking != settings.MarkingNeededToSubmit || !visible || !isEmptyValue(values[attr.ID]) {
			continue
		}
		if parentIDs, _ := choiceIDs(values[attr.ParentAttributeID]); attr.ParentAttributeID != "" && len(parentIDs) > 0 {
			options, err := x.repo.OptionsOf(ctx, attr.ID)
			if err != nil {
				return nil, err
			}
			linked := slices.ContainsFunc(options, func(o settings.Option) bool {
				return o.Active && slices.ContainsFunc(o.ParentOptionIDs, func(id string) bool { return slices.Contains(parentIDs, id) })
			})
			if !linked {
				continue
			}
		}
		missingFields = append(missingFields, settings.MissingField{Field: "attributes." + attr.ID, Label: attr.Label})
	}
	return missingFields, nil
}

// Describe gives an attribute's label and the labels of the chosen options; other values keep the default
// text.
func (x lineItemAttributes) Describe(ctx context.Context, field string, value any) (string, string, error) {
	attr, found, err := x.repo.GetAttribute(ctx, strings.TrimPrefix(field, "attributes."))
	if err != nil || !found {
		return "", "", err
	}
	var optionIDs []string
	switch v := value.(type) {
	case string:
		optionIDs = []string{v}
	case []any:
		for _, item := range v {
			if id, ok := item.(string); ok {
				optionIDs = append(optionIDs, id)
			}
		}
	}
	if !attr.IsChoice() || len(optionIDs) == 0 {
		return attr.Label, "", nil
	}
	labels := make([]string, 0, len(optionIDs))
	for _, id := range optionIDs {
		option, found, err := x.repo.GetOption(ctx, id)
		if err != nil {
			return "", "", err
		}
		if found {
			labels = append(labels, option.Label)
		}
	}
	return attr.Label, strings.Join(labels, ", "), nil
}

// attributes returns the Shell attributes of level lineItem, in display order.
func (x lineItemAttributes) attributes(ctx context.Context) ([]settings.Attribute, error) {
	all, err := x.repo.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	list := []settings.Attribute{}
	for _, attr := range all {
		if attr.Level == settings.LevelLineItem {
			list = append(list, attr)
		}
	}
	slices.SortStableFunc(list, func(a, b settings.Attribute) int { return a.Position - b.Position })
	return list, nil
}

// choiceIDs reads the option ids of a choice value: one id ("SO-1040") or a list of them; false when the
// value is something else. An empty value has no ids.
func choiceIDs(raw json.RawMessage) ([]string, bool) {
	if isEmptyValue(raw) {
		return []string{}, true
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return []string{one}, true
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		return many, true
	}
	return nil, false
}

// isEmptyValue reports whether an attribute value is empty: missing, null, "" or [].
func isEmptyValue(raw json.RawMessage) bool {
	switch strings.TrimSpace(string(raw)) {
	case "", "null", `""`, "[]":
		return true
	}
	return false
}

// midSentence writes a label for the middle of a sentence: the first letter in lower case when the second
// is lower case too ("Spend type" → "spend type"; "GL code" stays).
func midSentence(label string) string {
	first, size := utf8.DecodeRuneInString(label)
	second, _ := utf8.DecodeRuneInString(label[size:])
	if unicode.IsLower(second) {
		return string(unicode.ToLower(first)) + label[size:]
	}
	return label
}
