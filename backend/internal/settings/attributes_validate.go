package settings

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

// newOptions checks the options of the new attribute a (FR-018, FR-022) and returns them in the order
// given, without ids: a choice type has 1 to 5,000, each with a label of at most 120 characters that no
// other one has, ignoring letter case, and — when a depends on another attribute — linked to at least one
// active option of it; other types have none.
func (f formData) newOptions(a Attribute, inputs []NewOption, problems *errs.Problems) []Option {
	if !takesOptions(a, inputs, problems) {
		return nil
	}
	parent, dependent := f.parentOf(a)
	options := make([]Option, 0, len(inputs))
	var labels, twice []string
	for i, in := range inputs {
		field := fmt.Sprintf("options[%d]", i)
		label := strings.TrimSpace(in.Label)
		switch first := firstNamed(labels, label); {
		case label == "":
			problems.Add(field+".label", optionLabelMessage)
		case utf8.RuneCountInString(label) > maxOptionLabel:
			problems.Add(field+".label", tooLongMessage(maxOptionLabel))
		case first == "":
			labels = append(labels, label)
		case firstNamed(twice, first) == "":
			problems.Add(field+".label", fmt.Sprintf(appearsTwiceMessage, first))
			twice = append(twice, first)
		}
		links := []string{}
		if dependent {
			links = f.checkLinks(problems, linkCheck{field: field + ".parentOptionIds", label: label, parent: parent, links: in.ParentOptionIDs})
		}
		options = append(options, Option{Label: label, ParentOptionIDs: links})
	}
	return options
}

// takesOptions reports whether the new attribute a has options to check, adding the problem when inputs
// can't be its options.
func takesOptions(a Attribute, inputs []NewOption, problems *errs.Problems) bool {
	switch {
	case a.Type == "":
		return false // no type chosen: its options cannot be checked yet
	case !a.IsChoice():
		if len(inputs) > 0 {
			problems.Add("options", onlyChoiceOptionsMessage)
		}
		return false
	case len(inputs) == 0:
		problems.Add("options", addOptionMessage)
		return false
	case len(inputs) > maxOptions:
		problems.Add("options", tooManyOptionsMessage)
		return false
	}
	return true
}

// linkCheck is the parent options that the option label of a dependent attribute is linked to, with those
// it was linked to before (previous); field is where a problem goes.
type linkCheck struct {
	field, label string
	parent       Attribute
	links        []string
	previous     []string
}

// checkLinks checks the links of c (FR-022): at least one, each an option of the parent, and each active
// unless it was linked already. It returns them once each, in the order given.
func (f formData) checkLinks(problems *errs.Problems, c linkCheck) []string {
	links := unique(c.links)
	parentName := midSentence(c.parent.Label)
	if len(links) == 0 {
		problems.Add(c.field, fmt.Sprintf(linkOptionMessage, c.label, parentName))
		return links
	}
	for _, id := range links {
		o, found := f.optionByID[id]
		switch {
		case !found || o.AttributeID != c.parent.ID:
			problems.Add(c.field, fmt.Sprintf(linkParentOnlyMessage, c.label, parentName))
			return links
		case !o.Active && !slices.Contains(c.previous, id):
			problems.Add(c.field, fmt.Sprintf(inactiveLinkMessage, o.Label, parentName))
		}
	}
	return links
}

// checkCondition adds the problem of a condition whose options are not all options of the Business unit
// attribute (FR-023).
func (f formData) checkCondition(condition []string, problems *errs.Problems) {
	if f.businessUnit == nil {
		problems.Add("conditionOptionIds", noBusinessUnitMessage)
		return
	}
	for _, id := range condition {
		if o, found := f.optionByID[id]; !found || o.AttributeID != f.businessUnit.ID {
			problems.Add("conditionOptionIds", conditionOptionsMessage)
			return
		}
	}
}
