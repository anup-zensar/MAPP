package app

import (
	"context"
	"encoding/json"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// LineItems returns the activity's line items with their planned cost in their own currency and in USD at
// the current rate, and the readable labels of their finance coding (FR-032).
func (p *approvalsActivities) LineItems(ctx context.Context, _ actor.Actor, activityID string) ([]approval.PreviewLineItem, error) {
	items, err := p.lineItems.ListByActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	rates, err := p.currentRates(ctx)
	if err != nil {
		return nil, err
	}
	labels, err := p.codingLabels(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]approval.PreviewLineItem, 0, len(items))
	for _, li := range items {
		view := approval.PreviewLineItem{
			ID: li.ID, Name: li.Name, PlannedCost: money.Money{Amount: li.PlannedCost, Currency: li.Currency},
			SpendType: labels.of(li.Attributes, "Spend type"), GLCode: labels.of(li.Attributes, "GL code"),
		}
		if usd, err := rates.ToUSD(view.PlannedCost); err == nil {
			converted := money.USD(usd)
			view.PlannedCostDisplay = &converted // approvals converts it to the reader's display currency
		}
		if li.FinanceBudgetItemID != "" {
			item, found, err := p.settings.GetFinanceBudgetItem(ctx, li.FinanceBudgetItemID)
			if err != nil {
				return nil, err
			}
			if found {
				view.FinanceBudgetItem = item.Name
			}
		}
		list = append(list, view)
	}
	return list, nil
}

// coding finds the labels of the single-choice values of the line-item attributes, by attribute label.
type coding struct {
	attributeIDs map[string]string // attribute label → id
	options      map[string]string // option id → label
}

// codingLabels reads the line-item attributes and their options (002).
func (p *approvalsActivities) codingLabels(ctx context.Context) (coding, error) {
	labels := coding{attributeIDs: map[string]string{}, options: map[string]string{}}
	attributes, err := p.settings.ListAttributes(ctx)
	if err != nil {
		return coding{}, err
	}
	for _, attribute := range attributes {
		if attribute.Level == settings.LevelLineItem {
			labels.attributeIDs[attribute.Label] = attribute.ID
		}
	}
	options, err := p.settings.ListOptions(ctx)
	if err != nil {
		return coding{}, err
	}
	for _, option := range options {
		labels.options[option.ID] = option.Label
	}
	return labels, nil
}

// of returns the label of the option chosen in values for the attribute labelled attributeLabel, "" when
// none is chosen.
func (c coding) of(values settings.AttributeValues, attributeLabel string) string {
	var optionID string
	if raw, found := values[c.attributeIDs[attributeLabel]]; !found || json.Unmarshal(raw, &optionID) != nil {
		return ""
	}
	return c.options[optionID]
}
