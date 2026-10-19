package seed

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// SettingsSeed holds the IDs of the Settings records the seed created, by name (002 plan "Demo data"), for
// the planning steps that refer to them.
type SettingsSeed struct {
	Models        map[string]string // model name → ID
	ActivityTypes map[string]string // activity type name → ID
	Attributes    map[string]string // "<level>/<label>" → ID, such as "campaign/Cluster"
	Options       map[string]string // "<level>/<attribute>/<option>" → ID, such as "campaign/Region/LUBES – DACH"
	Budgets       map[string]string // "<year>/<budget name>" → ID
	Items         map[string]string // "<year>/<budget name>/<item name>" → ID
}

// The Settings seed of 002 data model §13. Option lists are fictitious samples (Q-003).

var demoCurrencies = []struct{ code, name, symbol, rate string }{
	{"USD", "US dollar", "$", "1"},
	{"EUR", "Euro", "€", "0.92"},
	{"GBP", "Pound sterling", "£", "0.79"},
	{"CNY", "Chinese yuan", "¥", "7.10"},
	{"INR", "Indian rupee", "₹", "83.50"},
}

var demoActivityTypes = []struct{ name, defaultPlannedCost string }{
	{"Agency fees – Creative", ""},
	{"Event", "25000.00"},
	{"Digital media", "10000.00"},
	{"Content production", ""},
	{"Sponsorship", "50000.00"},
	{"MOBILITY – Long-term contract", ""},
}

var demoModels = []struct {
	name, inqToMqlRate, mqlToDealRate, averageDealSize string
	activityTypes                                      []string
}{
	{"Lubricants 1.0", "2.00", "5.00", "10000.00",
		[]string{"Agency fees – Creative", "Event", "Digital media", "Content production", "Sponsorship"}},
	{"Mobility 1.0", "3.00", "4.00", "5000.00",
		[]string{"Agency fees – Creative", "Event", "Digital media", "Content production", "Sponsorship", "MOBILITY – Long-term contract"}},
}

// demoBudgetYear is the year of the seeded finance budgets: the year of the FY2026 plans.
const demoBudgetYear = 2026

var demoBudgets = []struct {
	name  string
	items [][2]string // name, code
}{
	{"2026 Lubricants Budget", [][2]string{{"Europe – Digital", "LUB-EU-DIG"}, {"Europe – Events", "LUB-EU-EVT"}, {"Europe – Brand", "LUB-EU-BRD"}}},
	{"2026 Mobility Budget", [][2]string{{"China – Retail", "MOB-CN-RET"}, {"China – Fleet", "MOB-CN-FLT"}, {"Delhi", "MOB-IN-DEL"}}},
}

// demoAttribute is one seeded Shell attribute. Parent is the label of the attribute it depends on (same
// level, earlier); Condition lists Business unit option labels.
type demoAttribute struct {
	level     settings.Level
	label     string
	typ       settings.AttributeType
	marking   settings.Marking
	section   string // "" means settings.DefaultSection
	help      string
	parent    string
	condition []string
	role      string
	options   []demoOption
}

// demoOption is one option of a seeded attribute, with the labels of the parent options it is linked to.
type demoOption struct {
	label   string
	parents []string
}

// opts lists options without links.
func opts(labels ...string) []demoOption {
	list := make([]demoOption, 0, len(labels))
	for _, l := range labels {
		list = append(list, demoOption{label: l})
	}
	return list
}

// The sections and conditions of the business-unit groups.
const (
	lubricantsSection = "Lubricants attributes"
	mobilitySection   = "Mobility attributes"
	financeSection    = "Finance coding"
)

var (
	lubricants = []string{"Lubricants"}
	mobility   = []string{"Mobility"}
)

// demoAttributes are the 35 Shell attributes, SA-1001 … in this order; positions 1…n per level.
var demoAttributes = []demoAttribute{
	{level: settings.LevelPlan, label: "Business unit", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToPublish,
		help: "Lubricants or Mobility. Decides which extra Shell attributes the plan's activities show.",
		role: settings.RoleBusinessUnit, options: opts("Mobility", "Lubricants")},
	{level: settings.LevelPlan, label: "Strategic Marketing Plan (SMP)", typ: settings.TypeYesNo, marking: settings.MarkingOptional, condition: mobility},

	{level: settings.LevelCampaign, label: "Cluster", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToPublish,
		options: opts("Lubricants – Europe", "Lubricants – Asia Pacific", "Retail – China", "Fleet – China", "Retail – India")},
	{level: settings.LevelCampaign, label: "Region", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToPublish,
		help: "Choose a cluster first — each region belongs to a cluster.", parent: "Cluster", options: []demoOption{
			{"LUBES – UK & IRELAND", []string{"Lubricants – Europe"}},
			{"LUBES – BENELUX", []string{"Lubricants – Europe"}},
			{"LUBES – DACH", []string{"Lubricants – Europe"}},
			{"LUBES – SOUTH EAST ASIA", []string{"Lubricants – Asia Pacific"}},
			{"RETAIL – CHINA", []string{"Retail – China"}},
			{"FLEET – NORTH CHINA", []string{"Fleet – China"}},
			{"FLEET – SOUTH CHINA", []string{"Fleet – China"}},
		}},
	{level: settings.LevelCampaign, label: "Insight session completed", typ: settings.TypeYesNo, marking: settings.MarkingOptional},
	{level: settings.LevelCampaign, label: "Workshop completed", typ: settings.TypeYesNo, marking: settings.MarkingOptional},
	{level: settings.LevelCampaign, label: "Global GM sign-off completed", typ: settings.TypeYesNo, marking: settings.MarkingOptional},

	{level: settings.LevelInitiative, label: "Growth ambition", typ: settings.TypeMultipleChoice, marking: settings.MarkingOptional,
		options: opts("LUBRICANTS – Premium growth", "LUBRICANTS – B2B expansion", "MOBILITY – Integrated offer", "MOBILITY – Convenience retail")},
	{level: settings.LevelInitiative, label: "Function", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		options: opts("Marketing", "Sales", "Customer service", "Digital")},
	{level: settings.LevelInitiative, label: "Focus area", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		options: opts("Brand", "Customer acquisition", "Loyalty", "Product launch")},
	{level: settings.LevelInitiative, label: "FIT4 initiative", typ: settings.TypeYesNo, marking: settings.MarkingOptional},

	{level: settings.LevelActivity, label: "Channel / audience", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("External", "Internal", "Partners")},
	{level: settings.LevelActivity, label: "Country / OU", typ: settings.TypeMultipleChoiceSplit, marking: settings.MarkingNeededToSubmit,
		help:    "Split the activity's cost over countries; the shares must add up to 100 %.",
		options: opts("China", "India", "United Kingdom", "Netherlands", "Germany", "Singapore")},
	{level: settings.LevelActivity, label: "Company code", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("GB01", "NL02", "CN04", "IN05", "DE03", "SG06")},
	{level: settings.LevelActivity, label: "Cost centre", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		help:    "Type part of the code to find it.",
		options: opts("CN04000220", "CN04000231", "GB01000105", "GB01000118", "NL02000340", "IN05000412", "DE03000150", "SG06000077")},
	{level: settings.LevelActivity, label: "Global campaign", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("Go Well integrated campaign", "Helix Ultra global platform", "Shell Recharge launch")},
	{level: settings.LevelActivity, label: "Growth ambition", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("LUBRICANTS – Premium growth", "LUBRICANTS – B2B expansion", "MOBILITY – Integrated offer", "MOBILITY – Convenience retail")},
	{level: settings.LevelActivity, label: "PIR", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("Yes", "No")},
	{level: settings.LevelActivity, label: "FIT4 initiative", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("FIT4 – Cost", "FIT4 – Growth", "Not FIT4")},
	{level: settings.LevelActivity, label: "FIT4 ID", typ: settings.TypeText, marking: settings.MarkingNeededToSubmit},
	{level: settings.LevelActivity, label: "Zero-based cost hierarchy", typ: settings.TypeMultipleChoice, marking: settings.MarkingOptional,
		options: opts("Media", "Agency", "Events", "Production")},
	{level: settings.LevelActivity, label: "Brands", typ: settings.TypeMultipleChoiceSplit, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants, options: opts("Helix", "Rimula", "Spirax", "Tellus")},
	{level: settings.LevelActivity, label: "Global key accounts", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants,
		options: opts("Account A – Automotive OEM", "Account B – Mining", "Account C – Logistics")},
	{level: settings.LevelActivity, label: "Sector", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants, options: opts("Automotive", "Mining", "Construction", "Agriculture")},
	{level: settings.LevelActivity, label: "Sub-sector", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants, options: opts("Passenger cars", "Heavy duty", "Off-highway")},
	{level: settings.LevelActivity, label: "Product launch", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants, options: opts("Yes – new product", "Yes – relaunch", "No")},
	{level: settings.LevelActivity, label: "Function (Retail)", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility, options: opts("Marketing", "Operations", "Category management")},
	{level: settings.LevelActivity, label: "Focus area (Retail)", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility, options: opts("Fuel", "Convenience", "Car wash")},
	{level: settings.LevelActivity, label: "Function (Mobility)", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility, options: opts("Marketing", "Fleet sales", "E-mobility")},
	{level: settings.LevelActivity, label: "Focus area (Mobility)", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility, options: opts("Fleet solutions", "EV charging", "Loyalty")},
	{level: settings.LevelActivity, label: "Innovation", typ: settings.TypeLongText, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility},

	{level: settings.LevelLineItem, label: "Spend type", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		section: financeSection,
		options: opts("Digital – Platform development", "Digital – Paid media", "BTL – Event", "ATL – TV", "Agency – Creative")},
	{level: settings.LevelLineItem, label: "GL code", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		section: financeSection, help: "Choose a spend type first — each GL code belongs to one or more spend types.",
		parent: "Spend type", options: []demoOption{
			{"7250080 – Advertising digital", []string{"Digital – Platform development", "Digital – Paid media", "Agency – Creative"}},
			{"7250010 – Advertising TV", []string{"ATL – TV"}},
			{"7250040 – Events and sponsorship", []string{"BTL – Event"}},
			{"7250060 – Agency fees", []string{"Agency – Creative"}},
			{"7250090 – Digital platforms", []string{"Digital – Platform development"}},
		}},
	{level: settings.LevelLineItem, label: "Spend category", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: financeSection, options: opts("B2B customer specific", "B2C mass market", "Internal")},
	{level: settings.LevelLineItem, label: "Contracted spend", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: financeSection, options: opts("Yes – long term", "Yes – short term", "No")},
}

// addSettings creates the Settings of 002 data model §13 through the settings repository, in this order
// (so the IDs are stable): currencies, activity types, models, finance budgets with their items, Shell
// attributes with their options. It keeps their IDs by name in d.settings.
func (d *demo) addSettings(ctx context.Context) error {
	repo := settings.NewJSONRepository()
	d.settings = SettingsSeed{
		Models: map[string]string{}, ActivityTypes: map[string]string{}, Attributes: map[string]string{},
		Options: map[string]string{}, Budgets: map[string]string{}, Items: map[string]string{},
	}
	for _, step := range []func(context.Context, settings.Repository) error{
		d.addCurrencies, d.addActivityTypes, d.addModels, d.addFinanceBudgets, d.addAttributes,
	} {
		if err := step(ctx, repo); err != nil {
			return err
		}
	}
	return nil
}

// addCurrencies creates the five currencies; a currency's ID is its code.
func (d *demo) addCurrencies(ctx context.Context, repo settings.Repository) error {
	for _, c := range demoCurrencies {
		rate, err := money.ParseRate(c.rate)
		if err != nil {
			return fmt.Errorf("seed: rate of %s: %w", c.code, err)
		}
		currency := settings.Currency{
			Code: c.code, Name: c.name, Symbol: c.symbol, Rate: rate,
			RateChangedAt: d.at, RateChangedBy: actor.System, Active: true, Meta: d.meta(),
		}
		if err := repo.PutCurrency(ctx, currency); err != nil {
			return err
		}
	}
	return nil
}

// addActivityTypes creates the six activity types, AT-1001 … AT-1006.
func (d *demo) addActivityTypes(ctx context.Context, repo settings.Repository) error {
	for _, t := range demoActivityTypes {
		id, err := d.next(ctx, ids.PrefixActivityType)
		if err != nil {
			return err
		}
		activityType := settings.ActivityType{ID: id, Name: t.name, Active: true, Meta: d.meta()}
		if t.defaultPlannedCost != "" {
			cost, err := usd(t.defaultPlannedCost)
			if err != nil {
				return err
			}
			activityType.DefaultPlannedCost = &cost
		}
		if err := repo.PutActivityType(ctx, activityType); err != nil {
			return err
		}
		d.settings.ActivityTypes[t.name] = id
	}
	return nil
}

// addModels creates the two models, MOD-1001 and MOD-1002, with their allowed activity types in the order
// of the activity-type list (by name).
func (d *demo) addModels(ctx context.Context, repo settings.Repository) error {
	for _, m := range demoModels {
		var r resolver
		typeNames := slices.Clone(m.activityTypes)
		slices.SortFunc(typeNames, func(a, b string) int { return cmp.Compare(strings.ToLower(a), strings.ToLower(b)) })
		typeIDs := make([]string, 0, len(typeNames))
		for _, name := range typeNames {
			typeIDs = append(typeIDs, r.id(d.settings.ActivityTypes, "activity type", name))
		}
		if r.err != nil {
			return r.err
		}
		inqToMql, err1 := money.ParsePercent(m.inqToMqlRate)
		mqlToDeal, err2 := money.ParsePercent(m.mqlToDealRate)
		dealSize, err3 := usd(m.averageDealSize)
		id, err4 := d.next(ctx, ids.PrefixModel)
		if err := cmp.Or(err1, err2, err3, err4); err != nil {
			return err
		}
		model := settings.Model{
			ID: id, Name: m.name, InqToMqlRate: inqToMql, MqlToDealRate: mqlToDeal, AverageDealSize: dealSize,
			ActivityTypeIDs: typeIDs, Active: true, Meta: d.meta(),
		}
		if err := repo.PutModel(ctx, model); err != nil {
			return err
		}
		d.settings.Models[m.name] = id
	}
	return nil
}

// addFinanceBudgets creates the two 2026 finance budgets, FB-1001 and FB-1002, each with three items,
// FBI-1001 … FBI-1006.
func (d *demo) addFinanceBudgets(ctx context.Context, repo settings.Repository) error {
	for _, b := range demoBudgets {
		budgetID, err := d.next(ctx, ids.PrefixFinanceBudget)
		if err != nil {
			return err
		}
		budget := settings.FinanceBudget{ID: budgetID, Year: demoBudgetYear, Name: b.name, Meta: d.meta()}
		if err := repo.PutFinanceBudget(ctx, budget); err != nil {
			return err
		}
		d.settings.Budgets[fmt.Sprintf("%d/%s", demoBudgetYear, b.name)] = budgetID
		for _, item := range b.items {
			itemID, err := d.next(ctx, ids.PrefixFinanceBudgetItem)
			if err != nil {
				return err
			}
			stored := settings.FinanceBudgetItem{ID: itemID, BudgetID: budgetID, Name: item[0], Code: item[1], Active: true, Meta: d.meta()}
			if err := repo.PutFinanceBudgetItem(ctx, stored); err != nil {
				return err
			}
			d.settings.Items[fmt.Sprintf("%d/%s/%s", demoBudgetYear, b.name, item[0])] = itemID
		}
	}
	return nil
}

// addAttributes creates the 35 Shell attributes, SA-1001 … SA-1035, with their 110 options, SO-1001 …,
// the links of dependent options to their parent options, and the business-unit conditions.
func (d *demo) addAttributes(ctx context.Context, repo settings.Repository) error {
	positions := map[settings.Level]int{}
	for _, a := range demoAttributes {
		var r resolver
		positions[a.level]++
		section := a.section
		if section == "" {
			section = settings.DefaultSection
		}
		attribute := settings.Attribute{
			Label: a.label, HelpText: a.help, Level: a.level, Type: a.typ, Marking: a.marking, Section: section,
			Position: positions[a.level], Condition: []string{}, Role: a.role, Active: true, Meta: d.meta(),
		}
		if a.parent != "" {
			attribute.ParentAttributeID = r.id(d.settings.Attributes, "Shell attribute", attributeKey(a.level, a.parent))
		}
		for _, unit := range a.condition {
			attribute.Condition = append(attribute.Condition,
				r.id(d.settings.Options, "Business unit option", optionKey(settings.LevelPlan, "Business unit", unit)))
		}
		if r.err != nil {
			return r.err
		}
		id, err := d.next(ctx, ids.PrefixShellAttribute)
		if err != nil {
			return err
		}
		attribute.ID = id
		if err := repo.PutAttribute(ctx, attribute); err != nil {
			return err
		}
		d.settings.Attributes[attributeKey(a.level, a.label)] = id

		for position, o := range a.options {
			option := settings.Option{
				AttributeID: id, Label: o.label, Position: position + 1, ParentOptionIDs: []string{}, Active: true, Meta: d.meta(),
			}
			for _, parent := range o.parents {
				option.ParentOptionIDs = append(option.ParentOptionIDs,
					r.id(d.settings.Options, "option", optionKey(a.level, a.parent, parent)))
			}
			if r.err != nil {
				return r.err
			}
			if option.ID, err = d.next(ctx, ids.PrefixShellOption); err != nil {
				return err
			}
			if err := repo.PutOption(ctx, option); err != nil {
				return err
			}
			d.settings.Options[optionKey(a.level, a.label, o.label)] = option.ID
		}
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
func (v *values) text(label, text string) *values {
	v.values[v.attribute(label)] = value(text)
	return v
}

// done returns the values, or the first label that did not resolve.
func (v *values) done() (settings.AttributeValues, error) {
	return v.values, v.r.err
}
