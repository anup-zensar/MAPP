package app

import (
	"bytes"
	"context"
	"encoding/json"

	"mapp/internal/allwork"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// loadAttributes reads every Shell attribute of every level and the options of the choice attributes once.
func (y *yearReader) loadAttributes(ctx context.Context) error {
	y.attributes, y.options = map[settings.Level][]settings.Attribute{}, map[string]settings.Option{}
	for _, level := range settings.Levels {
		list, err := y.s.settings.Attributes(ctx, level)
		if err != nil {
			return err
		}
		y.attributes[level] = list
		for _, a := range list {
			if !a.IsChoice() {
				continue
			}
			options, err := y.s.settings.Options(ctx, a.ID)
			if err != nil {
				return err
			}
			for _, o := range options {
				y.options[o.ID] = o
			}
		}
	}
	return nil
}

// addAttributes adds to fields the column "attr.<id>" of every Shell attribute of level that values has (002
// data model §9): a choice by its options' labels, a % split with the shares, Yes or No (an empty yes/no reads
// as No), a text or a whole number.
func (y *yearReader) addAttributes(fields map[string]allwork.Value, level settings.Level, values settings.AttributeValues) {
	for _, a := range y.attributes[level] {
		key := "attr." + a.ID
		raw := values[a.ID]
		if emptyValue(raw) {
			if a.Type == settings.TypeYesNo {
				no := false
				fields[key] = allwork.Value{Yes: &no}
			}
			continue
		}
		fields[key] = y.attributeValue(a.Type, raw)
	}
}

func (y *yearReader) attributeValue(t settings.AttributeType, raw json.RawMessage) allwork.Value {
	var v allwork.Value
	switch t {
	case settings.TypeSingleChoice:
		v.Choices = y.singleChoice(raw)
	case settings.TypeMultipleChoice:
		v.Choices = y.multipleChoices(raw)
	case settings.TypeMultipleChoiceSplit:
		v.Choices = y.splitChoices(raw)
	case settings.TypeYesNo:
		var yes bool
		if json.Unmarshal(raw, &yes) == nil {
			v.Yes = &yes
		}
	case settings.TypeNumber:
		var n int64
		if json.Unmarshal(raw, &n) == nil {
			v.Number = &n
		}
	case settings.TypeText, settings.TypeLongText:
		_ = json.Unmarshal(raw, &v.Text)
	}
	return v
}

func (y *yearReader) singleChoice(raw json.RawMessage) []allwork.Choice {
	var id string
	if json.Unmarshal(raw, &id) != nil {
		return nil
	}
	return []allwork.Choice{y.choice(id)}
}

func (y *yearReader) multipleChoices(raw json.RawMessage) []allwork.Choice {
	var ids []string
	if json.Unmarshal(raw, &ids) != nil {
		return nil
	}
	var choices []allwork.Choice
	for _, id := range ids {
		choices = append(choices, y.choice(id))
	}
	return choices
}

func (y *yearReader) splitChoices(raw json.RawMessage) []allwork.Choice {
	var shares []settings.SplitShare
	if json.Unmarshal(raw, &shares) != nil {
		return nil
	}
	var choices []allwork.Choice
	for _, s := range shares {
		c := y.choice(s.OptionID)
		c.Share = &s.Share
		choices = append(choices, c)
	}
	return choices
}

// attrValues returns the values of the choice and yes/no Shell attributes of level that values has, by
// attribute id, for the attribute filters and the Breakdown of 016 (FR-039 – FR-041): the options in their
// order (with the shares of a % split), or the yes/no answer — an empty yes/no reads as No.
func (y *yearReader) attrValues(level settings.Level, values settings.AttributeValues) map[string]allwork.AttrValue {
	attrs := map[string]allwork.AttrValue{}
	for _, a := range y.attributes[level] {
		raw := values[a.ID]
		if emptyValue(raw) {
			if a.Type == settings.TypeYesNo {
				no := false
				attrs[a.ID] = allwork.AttrValue{Yes: &no}
			}
			continue
		}
		if v := attrValue(a.Type, raw); len(v.OptionIDs) > 0 || v.Yes != nil {
			attrs[a.ID] = v
		}
	}
	return attrs
}

// attrValue returns the options or the yes/no answer of a stored value that is not empty.
func attrValue(t settings.AttributeType, raw json.RawMessage) allwork.AttrValue {
	var v allwork.AttrValue
	switch t {
	case settings.TypeSingleChoice:
		var id string
		if json.Unmarshal(raw, &id) == nil && id != "" {
			v.OptionIDs = []string{id}
		}
	case settings.TypeMultipleChoice:
		_ = json.Unmarshal(raw, &v.OptionIDs)
	case settings.TypeMultipleChoiceSplit:
		v = splitValue(raw)
	case settings.TypeYesNo:
		yes := false
		_ = json.Unmarshal(raw, &yes)
		v.Yes = &yes
	case settings.TypeText, settings.TypeLongText, settings.TypeNumber: // neither filtered nor broken down by
	}
	return v
}

func splitValue(raw json.RawMessage) allwork.AttrValue {
	var shares []settings.SplitShare
	if json.Unmarshal(raw, &shares) != nil {
		return allwork.AttrValue{}
	}
	v := allwork.AttrValue{Shares: map[string]money.Percent{}}
	for _, share := range shares {
		v.OptionIDs = append(v.OptionIDs, share.OptionID)
		v.Shares[share.OptionID] = share.Share
	}
	return v
}

// choice returns the option id as a choice; an option that no longer resolves shows its id.
func (y *yearReader) choice(id string) allwork.Choice {
	o, found := y.options[id]
	if !found {
		return allwork.Choice{Label: id}
	}
	return allwork.Choice{Label: o.Label, Inactive: !o.Active}
}

// emptyValue reports whether a stored attribute value is no value: missing, null, "", [] or {} (002 data
// model §9).
func emptyValue(raw json.RawMessage) bool {
	switch string(bytes.TrimSpace(raw)) {
	case "", "null", `""`, "[]", "{}":
		return true
	}
	return false
}
