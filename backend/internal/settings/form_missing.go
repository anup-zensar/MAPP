package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"mapp/internal/platform/money"
)

// MissingAttributes lists the needed Shell attributes of a record of level that are still empty, for its
// completeness and for the publish and submit checks of features 004–008 (FR-026, FR-027): active, shown
// for the plan's business unit businessUnit, not yes/no, and empty — or a % split not adding up to
// 100.00 %. It joins the caller's transaction.
func (s *Service) MissingAttributes(ctx context.Context, level Level, values AttributeValues, businessUnit []string) ([]MissingField, error) {
	var missing []MissingField
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.formOf(ctx, level)
		if err != nil {
			return err
		}
		normalized := map[string]json.RawMessage{}
		for _, a := range f.attributes {
			if raw, problem := f.normalize(a, values[a.ID]); problem == "" && raw != nil {
				normalized[a.ID] = raw
			}
		}
		if level == LevelPlan {
			businessUnit = f.businessUnitOf(normalized)
		}
		missing = f.missing(normalized, businessUnit)
		return nil
	})
	return missing, err
}

// BusinessUnit returns the plan's business unit: the option ids of the Business unit attribute in the
// plan's values planValues, or an empty list when none is chosen (FR-023, FR-026).
func (s *Service) BusinessUnit(ctx context.Context, planValues AttributeValues) ([]string, error) {
	var businessUnit []string
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.formOf(ctx, LevelPlan)
		if err != nil {
			return err
		}
		businessUnit = f.businessUnitOf(planValues)
		return nil
	})
	return businessUnit, err
}

// businessUnitOf returns the Business unit option ids in plan values.
func (f formData) businessUnitOf(values map[string]json.RawMessage) []string {
	if f.businessUnit == nil {
		return []string{}
	}
	return optionIDsOf(*f.businessUnit, values[f.businessUnit.ID])
}

// missing lists the needed attributes still empty in values on a record whose plan's business unit is
// businessUnit (data model §9 "Missing").
func (f formData) missing(values map[string]json.RawMessage, businessUnit []string) []MissingField {
	missing := []MissingField{}
	for _, a := range f.attributes {
		if !needsValue(a, businessUnit) {
			continue
		}
		value := values[a.ID]
		field := MissingField{Field: "attributes." + a.ID, Label: a.Label}
		switch {
		case value == nil && !f.noOptionsForParent(a, values):
			missing = append(missing, field)
		case value != nil && a.Type == TypeMultipleChoiceSplit:
			if total := sharesTotal(value); total != money.Hundred {
				field.Reason = fmt.Sprintf(sharesTotalMessage, total.String())
				missing = append(missing, field)
			}
		}
	}
	return missing
}

func needsValue(a Attribute, businessUnit []string) bool {
	return a.Active && a.Marking != MarkingOptional && a.Type != TypeYesNo && Visible(a, businessUnit)
}

func sharesTotal(value json.RawMessage) money.Percent {
	var shares []SplitShare
	_ = json.Unmarshal(value, &shares) // normalized already
	total := money.PercentFromHundredths(0)
	for _, s := range shares {
		total = money.SumPercent(total, s.Share)
	}
	return total
}

// noOptionsForParent reports whether the dependent attribute a has no active option linked to any of the
// parent options chosen in values (D-034): it then cannot be filled in, so it is never missing.
func (f formData) noOptionsForParent(a Attribute, values map[string]json.RawMessage) bool {
	parent, dependent := f.byID[a.ParentAttributeID]
	if !dependent {
		return false
	}
	selected := optionIDsOf(parent, values[parent.ID])
	if len(selected) == 0 {
		return false
	}
	for _, o := range f.options[a.ID] {
		if o.Active && f.linked(o.ID, selected) {
			return false
		}
	}
	return true
}
