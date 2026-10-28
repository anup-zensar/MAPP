package seed

import (
	"context"
	"slices"

	"mapp/internal/platform/ids"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// addAttributes creates the 56 Shell attributes, SA-1001 … SA-1056, with their 185 options, SO-1001 …: the
// planning levels' first, then the purchase request's (032).
func (d *demo) addAttributes(ctx context.Context, repo stores.SettingsStore) error {
	positions := map[settings.Level]int{}
	for _, a := range append(slices.Clone(demoAttributes), purchaseRequestAttributes()...) {
		positions[a.level]++
		attribute, err := d.attributeOf(a, positions[a.level])
		if err != nil {
			return err
		}
		if attribute.ID, err = d.next(ctx, ids.PrefixShellAttribute); err != nil {
			return err
		}
		if err := repo.PutAttribute(ctx, attribute); err != nil {
			return err
		}
		d.settings.Attributes[attributeKey(a.level, a.label)] = attribute.ID
		if err := d.addOptions(ctx, repo, a, attribute.ID); err != nil {
			return err
		}
	}
	return nil
}

// attributeOf resolves the parent attribute and business-unit condition of a but leaves its ID empty.
func (d *demo) attributeOf(a demoAttribute, position int) (settings.Attribute, error) {
	var r resolver
	section := a.section
	if section == "" {
		section = settings.DefaultSection
	}
	attribute := settings.Attribute{
		Label: a.label, HelpText: a.help, Level: a.level, Type: a.typ, Marking: a.marking, Section: section,
		Position: position, Condition: []string{}, Role: a.role, Active: true, Meta: d.meta(),
	}
	if a.parent != "" {
		attribute.ParentAttributeID = r.id(d.settings.Attributes, "Shell attribute", attributeKey(a.level, a.parent))
	}
	for _, unit := range a.condition {
		attribute.Condition = append(attribute.Condition,
			r.id(d.settings.Options, "Business unit option", optionKey(settings.LevelPlan, "Business unit", unit)))
	}
	return attribute, r.err
}

func (d *demo) addOptions(ctx context.Context, repo stores.SettingsStore, a demoAttribute, attributeID string) error {
	var r resolver
	for position, o := range a.options {
		option := settings.Option{
			AttributeID: attributeID, Label: o.label, Position: position + 1, ParentOptionIDs: []string{}, Active: true, Meta: d.meta(),
		}
		for _, parent := range o.parents {
			option.ParentOptionIDs = append(option.ParentOptionIDs,
				r.id(d.settings.Options, "option", optionKey(a.level, a.parent, parent)))
		}
		if r.err != nil {
			return r.err
		}
		var err error
		if option.ID, err = d.next(ctx, ids.PrefixShellOption); err != nil {
			return err
		}
		if err := repo.PutOption(ctx, option); err != nil {
			return err
		}
		d.settings.Options[optionKey(a.level, a.label, o.label)] = option.ID
	}
	return nil
}

// attributeKey is the key of an attribute in SettingsSeed.Attributes: "campaign/Cluster".
func attributeKey(level settings.Level, label string) string {
	return string(level) + "/" + label
}

// optionKey is the key of an option in SettingsSeed.Options: "campaign/Region/LUBES – DACH".
func optionKey(level settings.Level, attribute, option string) string {
	return string(level) + "/" + attribute + "/" + option
}
