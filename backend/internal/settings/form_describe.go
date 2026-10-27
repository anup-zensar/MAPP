package settings

import (
	"context"
	"encoding/json"
	"strings"
)

// DescribeAttribute gives the label and text of a Shell attribute value in the activity log (data model
// §12): for the field "attributes.<id>", the attribute's label and its value's text — option labels joined
// with ", ", shares as "China 60.00 %, India 40.00 %", "Yes" or "No", the text or the number. Other fields
// give "" and "", meaning "use the default". The Describe functions of the planning kinds of 004–007 call
// it; value is the stored value decoded, as history.Kind.Describe receives it.
func (s *Service) DescribeAttribute(ctx context.Context, field string, value any) (label, text string, err error) {
	id, isAttribute := strings.CutPrefix(field, "attributes.")
	if !isAttribute {
		return "", "", nil
	}
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		a, found, readErr := s.repo.GetAttribute(ctx, id)
		if readErr != nil || !found {
			return readErr
		}
		label = a.Label
		text, readErr = s.valueText(ctx, a, value)
		return readErr
	})
	return label, text, err
}

// valueText writes a decoded value of the attribute a as people read it; "" for an empty value.
func (s *Service) valueText(ctx context.Context, a Attribute, value any) (string, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return "Yes", nil
		}
		return "No", nil
	case string:
		if a.IsChoice() {
			return s.namesOf(ctx, v, s.optionLabel)
		}
		return v, nil
	case []any:
		if a.Type != TypeMultipleChoiceSplit {
			return s.namesOf(ctx, v, s.optionLabel)
		}
		return s.sharesText(ctx, v)
	case json.Number:
		return v.String(), nil
	}
	return "", nil
}

func (s *Service) sharesText(ctx context.Context, shares []any) (string, error) {
	texts := make([]string, 0, len(shares))
	for _, item := range shares {
		share, _ := item.(map[string]any)
		id, _ := share["optionId"].(string)
		label, err := s.namesOf(ctx, id, s.optionLabel)
		if err != nil {
			return "", err
		}
		texts = append(texts, strings.TrimSpace(label+" "+percentText(share["share"])))
	}
	return strings.Join(texts, ", "), nil
}
