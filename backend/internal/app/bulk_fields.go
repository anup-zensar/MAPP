package app

import (
	"context"
	"strings"

	"mapp/internal/bulk"
	"mapp/internal/settings"
)

const (
	bulkKindDate   = "date"
	bulkKindSingle = "singleChoice"
	bulkKindMany   = "multipleChoice"
	bulkKindYesNo  = "yesNo"
	attrKeyPrefix  = "attr."

	keyStartDate        = "startDate"
	keyEndDate          = "endDate"
	keyActivityType     = "activityType"
	keyApprovalWorkflow = "approvalWorkflow"
)

// Fields lists what Change field offers for a level (012 FR-009): the dates, for activities the type and the
// approval workflow, then the level's active yes/no, single-choice and multiple-choice Shell attributes.
func (p bulkPlanning) Fields(ctx context.Context, level bulk.Level) ([]bulk.Field, error) {
	fields := []bulk.Field{
		{Key: keyStartDate, Label: "Start date", Kind: bulkKindDate, Options: []bulk.FieldOption{}},
		{Key: keyEndDate, Label: "End date", Kind: bulkKindDate, Options: []bulk.FieldOption{}},
	}
	if level == bulk.LevelActivity {
		more, err := p.activityFields(ctx)
		if err != nil {
			return nil, err
		}
		fields = append(fields, more...)
	}
	attributes, err := p.settings.Attributes(ctx, settings.Level(level))
	if err != nil {
		return nil, err
	}
	for _, attr := range attributes {
		kind, offered := bulkKindOf(attr)
		if !attr.Active || !offered {
			continue
		}
		field, err := p.attributeField(ctx, attr, kind)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func bulkKindOf(attr settings.Attribute) (string, bool) {
	switch attr.Type {
	case settings.TypeSingleChoice:
		return bulkKindSingle, true
	case settings.TypeMultipleChoice:
		return bulkKindMany, true
	case settings.TypeYesNo:
		return bulkKindYesNo, true
	case settings.TypeMultipleChoiceSplit, settings.TypeText, settings.TypeLongText, settings.TypeNumber:
	}
	return "", false
}

func (p bulkPlanning) activityFields(ctx context.Context) ([]bulk.Field, error) {
	types, err := p.settings.ActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	typeOptions := []bulk.FieldOption{}
	for _, t := range types {
		if t.Active {
			typeOptions = append(typeOptions, bulk.FieldOption{ID: t.ID, Label: t.Name, ParentOptionIDs: []string{}})
		}
	}
	workflows, err := p.workflows.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	workflowOptions := []bulk.FieldOption{}
	for _, w := range workflows {
		if w.Active {
			workflowOptions = append(workflowOptions, bulk.FieldOption{ID: w.ID, Label: w.Name, ParentOptionIDs: []string{}})
		}
	}
	return []bulk.Field{
		{Key: keyActivityType, Label: "Activity type", Kind: bulkKindSingle, Required: true, Options: typeOptions},
		{Key: keyApprovalWorkflow, Label: "Approval workflow", Kind: bulkKindSingle, Options: workflowOptions},
	}, nil
}

func (p bulkPlanning) attributeField(ctx context.Context, attr settings.Attribute, kind string) (bulk.Field, error) {
	field := bulk.Field{Key: attrKeyPrefix + attr.ID, Label: attr.Label, Kind: kind, Options: []bulk.FieldOption{}}
	if attr.ParentAttributeID != "" {
		parent := attrKeyPrefix + attr.ParentAttributeID
		field.DependsOn = &parent
	}
	options, err := p.settings.Options(ctx, attr.ID)
	if err != nil {
		return bulk.Field{}, err
	}
	for _, o := range options {
		if o.Active {
			field.Options = append(field.Options, bulk.FieldOption{ID: o.ID, Label: o.Label, ParentOptionIDs: append([]string{}, o.ParentOptionIDs...)})
		}
	}
	if len(attr.Condition) == 0 {
		return field, nil
	}
	labels := []string{}
	for _, id := range attr.Condition {
		option, found, err := p.settings.Option(ctx, id)
		if err != nil {
			return bulk.Field{}, err
		}
		if found {
			labels = append(labels, option.Label)
		}
	}
	condition := "Shown when the plan's business unit includes " + strings.Join(labels, " or ")
	field.Condition = &condition
	return field, nil
}
