package app

import (
	"context"

	"mapp/internal/plan"
	"mapp/internal/settings"
	"mapp/internal/workbook"
)

// choices are the lists the plan's cells choose from: people, the types its model allows, workflows,
// currencies and the active Shell attributes its business unit uses (R7, R9).
func (p workbookPlanning) choices(ctx context.Context, pl plan.Plan) (workbook.Choices, error) {
	var ch workbook.Choices
	people, err := p.people.List(ctx)
	if err != nil {
		return ch, err
	}
	for _, person := range people {
		ch.People = append(ch.People, workbook.Person{ID: person.ID, Email: person.Email, Active: person.Active})
	}
	if err = p.typesOf(ctx, pl.ModelID, &ch); err != nil {
		return ch, err
	}
	workflows, err := p.workflows.ListWorkflows(ctx)
	if err != nil {
		return ch, err
	}
	for _, w := range workflows {
		ch.Workflows = append(ch.Workflows, workbook.Option{ID: w.ID, Label: w.Name, Active: w.Active})
	}
	currencies, err := p.settings.Currencies(ctx)
	if err != nil {
		return ch, err
	}
	for _, c := range currencies {
		ch.Currencies = append(ch.Currencies, workbook.Option{ID: c.Code, Label: c.Code, Active: c.Active})
	}
	ch.Attributes, err = p.attributesFor(ctx, pl.Attributes)
	return ch, err
}

func (p workbookPlanning) typesOf(ctx context.Context, modelID string, ch *workbook.Choices) error {
	model, _, err := p.settings.Model(ctx, modelID)
	if err != nil {
		return err
	}
	types, err := p.settings.ActivityTypes(ctx)
	ch.ModelName = model.Name
	for _, t := range types {
		if model.Allows(t.ID) {
			ch.ActivityTypes = append(ch.ActivityTypes, workbook.Option{ID: t.ID, Label: t.Name, Active: t.Active})
		}
	}
	return err
}

func (p workbookPlanning) attributesFor(ctx context.Context, planValues settings.AttributeValues) ([]workbook.Attribute, error) {
	businessUnit, err := p.settings.BusinessUnit(ctx, planValues)
	if err != nil {
		return nil, err
	}
	var attributes []workbook.Attribute
	for _, level := range []workbook.Level{workbook.LevelCampaign, workbook.LevelInitiative, workbook.LevelActivity, workbook.LevelLineItem} {
		list, err := p.settings.Attributes(ctx, settings.Level(level))
		if err != nil {
			return nil, err
		}
		for _, a := range list {
			if !a.Active || !settings.Visible(a, businessUnit) {
				continue
			}
			options, err := p.settings.Options(ctx, a.ID)
			if err != nil {
				return nil, err
			}
			attr := workbook.Attribute{ID: a.ID, Label: a.Label, Level: level, Type: string(a.Type), Marking: string(a.Marking)}
			for _, o := range options {
				attr.Options = append(attr.Options, workbook.Option{ID: o.ID, Label: o.Label, Active: o.Active})
			}
			attributes = append(attributes, attr)
		}
	}
	return attributes, nil
}
