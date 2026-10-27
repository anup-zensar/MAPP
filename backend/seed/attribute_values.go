package seed

import "mapp/internal/settings"

// values builds the Shell attribute values of one record from attribute labels at its level, and keeps
// the first label that does not resolve.
type values struct {
	d      *demo
	level  settings.Level
	values settings.AttributeValues
	r      resolver
}

// newValues starts the values of a record of level.
func (d *demo) newValues(level settings.Level) *values {
	return &values{d: d, level: level, values: settings.AttributeValues{}}
}

// attribute returns the ID of the attribute label at the record's level.
func (v *values) attribute(label string) string {
	return v.r.id(v.d.settings.Attributes, "Shell attribute", attributeKey(v.level, label))
}

// option returns the ID of the option of the attribute label at the record's level.
func (v *values) option(label, option string) string {
	return v.r.id(v.d.settings.Options, "option", optionKey(v.level, label, option))
}

// choice sets a single-choice value.
func (v *values) choice(label, option string) *values {
	id := v.option(label, option)
	v.values[v.attribute(label)] = value(id)
	return v
}

// choices sets a multiple-choice value.
func (v *values) choices(label string, options ...string) *values {
	list := make([]string, 0, len(options))
	for _, o := range options {
		list = append(list, v.option(label, o))
	}
	v.values[v.attribute(label)] = value(list)
	return v
}

// split sets a multiple-choice-with-%-split value from option labels and shares: "Germany", "50.00", ….
func (v *values) split(label string, optionsAndShares ...string) *values {
	var shares []share
	for i := 0; i+1 < len(optionsAndShares); i += 2 {
		shares = append(shares, share{optionID: v.option(label, optionsAndShares[i]), percent: optionsAndShares[i+1]})
	}
	encoded, err := splitValue(shares...)
	if err != nil && v.r.err == nil {
		v.r.err = err
	}
	v.values[v.attribute(label)] = encoded
	return v
}

// yesNo sets a yes/no value.
func (v *values) yesNo(label string, yes bool) *values {
	v.values[v.attribute(label)] = value(yes)
	return v
}

// text sets a text value.
func (v *values) text(label, text string) *values { //nolint:unparam // every setter takes the label, so a record's values read alike
	v.values[v.attribute(label)] = value(text)
	return v
}

// done returns the values, or the first label that did not resolve.
func (v *values) done() (settings.AttributeValues, error) {
	return v.values, v.r.err
}
