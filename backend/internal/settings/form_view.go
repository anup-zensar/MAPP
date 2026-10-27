package settings

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/memo"
)

// formOf shares one read of the form per transaction with the rules run once per record, as on Home; its
// callers must not change it, so code that changes Settings uses loadForm.
func (s *Service) formOf(ctx context.Context, level Level) (formData, error) {
	return memo.Get(ctx, "settings:form:"+string(level), func(ctx context.Context) (formData, error) {
		return s.loadForm(ctx, level)
	})
}

func (s *Service) loadForm(ctx context.Context, level Level) (formData, error) {
	attributes, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return formData{}, err
	}
	options, err := s.repo.ListOptions(ctx)
	if err != nil {
		return formData{}, err
	}
	f := formData{
		level: level, attributes: ofLevel(attributes, level), byID: map[string]Attribute{},
		options: map[string][]Option{}, optionByID: map[string]Option{},
	}
	for _, a := range attributes {
		f.byID[a.ID] = a
		if a.Role == RoleBusinessUnit {
			businessUnit := a
			f.businessUnit = &businessUnit
		}
	}
	for _, o := range optionOrder(options) {
		f.options[o.AttributeID] = append(f.options[o.AttributeID], o)
		f.optionByID[o.ID] = o
	}
	return f, nil
}

// Form holds every attribute and option, active or not, and the Business unit options for condition notes.
func (s *Service) Form(ctx context.Context, _ actor.Actor, level Level) (FormView, error) {
	var view FormView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.loadForm(ctx, level)
		if err != nil {
			return err
		}
		view = f.view()
		return nil
	})
	return view, err
}

func (f formData) view() FormView {
	view := FormView{Level: f.level, Sections: []FormSection{}, BusinessUnitOptions: []FormOption{}}
	for _, a := range f.attributes {
		last := len(view.Sections) - 1
		if last < 0 || view.Sections[last].Name != a.Section {
			view.Sections = append(view.Sections, FormSection{Name: a.Section, Conditional: true, Attributes: []FormAttribute{}})
			last++
		}
		section := &view.Sections[last]
		section.Attributes = append(section.Attributes, f.formAttribute(a))
		section.Conditional = section.Conditional && len(a.Condition) > 0
	}
	if f.businessUnit != nil {
		id := f.businessUnit.ID
		view.BusinessUnitAttributeID = &id
		view.BusinessUnitOptions = f.formOptions(id)
	}
	return view
}

func (f formData) formAttribute(a Attribute) FormAttribute {
	var parent *string
	if a.ParentAttributeID != "" {
		id := a.ParentAttributeID
		parent = &id
	}
	return FormAttribute{
		ID: a.ID, Label: a.Label, HelpText: a.HelpText, Type: a.Type, Marking: a.Marking, Active: a.Active,
		ParentAttributeID: parent, ConditionOptionIDs: nonNil(a.Condition), Options: f.formOptions(a.ID),
	}
}

func (f formData) formOptions(attributeID string) []FormOption {
	options := []FormOption{}
	for _, o := range f.options[attributeID] {
		options = append(options, FormOption{ID: o.ID, Label: o.Label, Active: o.Active, ParentOptionIDs: nonNil(o.ParentOptionIDs)})
	}
	return options
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
