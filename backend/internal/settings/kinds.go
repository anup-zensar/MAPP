package settings

import (
	"context"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/money"
)

// The history kinds of the Settings records (data model §11).
const (
	modelKind        = "model"
	activityTypeKind = "activityType"
	attributeKind    = "shellAttribute"
	optionKind       = "shellAttributeOption"
	currencyKind     = "currency"
	budgetKind       = "financeBudget"
	itemKind         = "financeBudgetItem"
)

// Kinds returns the history kinds of the Settings records for internal/app to register, parents first:
// model, activity type, Shell attribute, option (under its attribute), currency, finance budget and finance
// budget item (under its budget). With them, history logs their changes in words, deletes an attribute
// with its options, and brings records back with the restore rules of data model §11.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{
			Name: modelKind, Singular: modelNoun, Plural: "models",
			Snapshot: s.modelSnapshot, Remove: s.repo.DeleteModel, Put: putRecord(s.repo.PutModel),
			Describe: s.describeModel, PrepareRestore: s.prepareModelRestore,
		},
		{
			Name: activityTypeKind, Singular: activityTypeNoun, Plural: "activity types",
			Snapshot: s.activityTypeSnapshot, Remove: s.repo.DeleteActivityType, Put: putRecord(s.repo.PutActivityType),
			Describe: describeActivityType, PrepareRestore: s.prepareActivityTypeRestore,
		},
		{
			Name: attributeKind, Singular: attributeNoun, Plural: "Shell attributes",
			Snapshot: s.attributeSnapshot, Remove: s.repo.DeleteAttribute, Put: putRecord(s.repo.PutAttribute),
			Describe: s.describeAttribute, PrepareRestore: s.prepareAttributeRestore,
		},
		{
			Name: optionKind, Singular: optionNoun, Plural: "options", Parent: attributeKind,
			Snapshot: s.optionSnapshot, ChildIDs: s.optionIDs, Remove: s.repo.DeleteOption, Put: putRecord(s.repo.PutOption),
			Describe: s.describeOption, PrepareRestore: s.prepareOptionRestore,
		},
		{
			Name: currencyKind, Singular: currencyNoun, Plural: "currencies",
			Snapshot: s.currencySnapshot, Remove: s.repo.DeleteCurrency, Put: putRecord(s.repo.PutCurrency),
			Describe: describeCurrency, PrepareRestore: s.prepareCurrencyRestore,
		},
		{
			// 016: a currency's dated rates, deleted with it or one by one (D-272).
			Name: exchangeRateKind, Singular: exchangeRateNoun, Plural: "exchange rates", Parent: currencyKind,
			Snapshot: s.rateSnapshot, ChildIDs: s.rateIDs, Remove: s.repo.DeleteRate, Put: putRecord(s.repo.PutRate),
			Describe: describeRate, PrepareRestore: s.prepareRateRestore,
		},
		{
			// 016: GSAP rate imports, logged and never deleted.
			Name: rateImportKind, Singular: rateImportNoun, Plural: "rate imports",
			Snapshot: s.rateImportSnapshot, Remove: neverRemoved, Put: putRecord(s.repo.PutRateImport),
		},
		{
			// 016: edited Field help texts, logged and never in the recycle bin.
			Name: fieldHelpKind, Singular: fieldHelpNoun, Plural: "field help texts",
			Snapshot: s.fieldHelpSnapshot, Remove: neverRemoved, Put: putRecord(s.repo.PutFieldHelp),
			Describe: describeFieldHelp,
		},
		{
			Name: budgetKind, Singular: budgetNoun, Plural: "finance budgets",
			Snapshot: s.budgetSnapshot, Remove: s.repo.DeleteFinanceBudget, Put: putRecord(s.repo.PutFinanceBudget),
			Describe: describeBudget,
		},
		{
			Name: itemKind, Singular: itemNoun, Plural: "finance budget items", Parent: budgetKind,
			Snapshot: s.itemSnapshot, ChildIDs: s.itemIDs, Remove: s.repo.DeleteFinanceBudgetItem, Put: putRecord(s.repo.PutFinanceBudgetItem),
			Describe: s.describeItem, PrepareRestore: s.prepareItemRestore,
		},
	}
}

// The words of the attribute enums on screen and in the log (data model §4).
var (
	levelWords = map[Level]string{
		LevelPlan: "Plan", LevelCampaign: "Campaign", LevelInitiative: "Initiative", LevelActivity: "Activity", LevelLineItem: "Line item",
		LevelPurchaseRequest: "Purchase request", LevelPurchaseRequestLine: prLineWords,
	}
	typeWords = map[AttributeType]string{
		TypeSingleChoice: "Single choice", TypeMultipleChoice: "Multiple choice", TypeMultipleChoiceSplit: "Multiple choice with % split",
		TypeYesNo: "Yes/no", TypeText: "Text", TypeLongText: "Long text", TypeNumber: "Number",
	}
	markingWords = map[Marking]string{
		MarkingOptional: "Optional", MarkingNeededToPublish: "Needed to publish", MarkingNeededToSubmit: "Needed to submit",
	}
)

// The Describe functions give the label of each field and the text of each value in the activity log
// (data model §12). value is the stored JSON value decoded: nil, bool, json.Number, string, []any or
// map[string]any. An empty text means history's default ("Not filled in" for an empty value).

func (s *Service) describeModel(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Name", "", nil
	case "inqToMqlRate":
		return "INQ→MQL rate", percentText(value), nil
	case "mqlToDealRate":
		return "MQL→deal rate", percentText(value), nil
	case "averageDealSize":
		return "Average deal size", moneyText(value), nil
	case "activityTypeIds":
		text, err := s.namesOf(ctx, value, func(ctx context.Context, id string) (string, bool, error) {
			t, found, err := s.repo.GetActivityType(ctx, id)
			return t.Name, found, err
		})
		return "Allowed activity types", text, err
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func describeActivityType(_ context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Name", "", nil
	case "defaultPlannedCost":
		return "Default planned cost", moneyText(value), nil
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func (s *Service) describeAttribute(ctx context.Context, field string, value any) (string, string, error) {
	text, _ := value.(string)
	switch field {
	case "label":
		return "Label", "", nil
	case "helpText":
		return "Help text", "", nil
	case "level":
		return "Level", levelWords[Level(text)], nil
	case "type":
		return "Type", typeWords[AttributeType(text)], nil
	case "marking":
		return "Marking", markingWords[Marking(text)], nil
	case "section":
		return "Section", "", nil
	case "position":
		return "Place on the form", "", nil
	case "parentAttributeId":
		label, err := s.namesOf(ctx, value, s.attributeLabel)
		return "Depends on", label, err
	case "condition":
		if list, _ := value.([]any); len(list) == 0 {
			return "Shown when the business unit includes", "Always", nil
		}
		labels, err := s.namesOf(ctx, value, s.optionLabel)
		return "Shown when the business unit includes", labels, err
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func (s *Service) describeOption(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "label":
		return "Label", "", nil
	case "parentOptionIds":
		labels, err := s.namesOf(ctx, value, s.optionLabel)
		return "Linked to", labels, err
	case "position":
		return "Place in the list", "", nil
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func describeCurrency(_ context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Name", "", nil
	case "symbol":
		return "Symbol", "", nil
	case "rate":
		text, _ := value.(string)
		if rate, err := money.ParseRate(text); err == nil {
			return "Rate (units per 1 USD)", rate.Format(), nil
		}
		return "Rate (units per 1 USD)", "", nil
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func describeBudget(_ context.Context, field string, _ any) (string, string, error) {
	if field == "name" {
		return "Budget name", "", nil
	}
	return "", "", nil
}

func (s *Service) describeItem(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "name":
		return "Item name", "", nil
	case "code":
		return "Code", "", nil
	case "budgetId":
		name, err := s.namesOf(ctx, value, func(ctx context.Context, id string) (string, bool, error) {
			b, found, err := s.repo.GetFinanceBudget(ctx, id)
			return b.Name, found, err
		})
		return "Budget", name, err
	case "active":
		return "Status", statusText(value), nil
	}
	return "", "", nil
}

func (s *Service) attributeLabel(ctx context.Context, id string) (string, bool, error) {
	a, found, err := s.repo.GetAttribute(ctx, id)
	return a.Label, found, err
}

func (s *Service) optionLabel(ctx context.Context, id string) (string, bool, error) {
	o, found, err := s.repo.GetOption(ctx, id)
	return o.Label, found, err
}

// namesOf writes the names of the ids in value (one id or a list of ids) joined with ", ", looking each up
// with name; an id that is not stored is written as it is. No ids give "".
func (s *Service) namesOf(ctx context.Context, value any, name func(context.Context, string) (string, bool, error)) (string, error) {
	var ids []string
	switch v := value.(type) {
	case string:
		ids = []string{v}
	case []any:
		for _, item := range v {
			if id, ok := item.(string); ok {
				ids = append(ids, id)
			}
		}
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		n, found, err := name(ctx, id)
		if err != nil {
			return "", err
		}
		if !found {
			n = id
		}
		names = append(names, n)
	}
	return strings.Join(names, ", "), nil
}

func statusText(value any) string {
	if active, ok := value.(bool); ok && !active {
		return "Inactive"
	}
	if value == nil {
		return ""
	}
	return "Active"
}

func percentText(value any) string {
	text, _ := value.(string)
	p, err := money.ParsePercent(text)
	if err != nil {
		return ""
	}
	return p.Format()
}

func moneyText(value any) string {
	object, _ := value.(map[string]any)
	text, _ := object["amount"].(string)
	currency, _ := object["currency"].(string)
	amount, err := money.ParseAmount(text)
	if err != nil {
		return ""
	}
	return money.Money{Amount: amount, Currency: currency}.Format()
}
