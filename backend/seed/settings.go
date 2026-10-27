package seed

import (
	"context"

	"mapp/internal/settings"
	"mapp/internal/stores"
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
	{level: settings.LevelPlan, label: "Strategic Marketing Plan (SMP)", help: "Yes when the plan is part of Mobility's Strategic Marketing Plan.", typ: settings.TypeYesNo, marking: settings.MarkingOptional, condition: mobility},

	{level: settings.LevelCampaign, label: "Cluster", help: "The business clusters the campaign serves. Regions are chosen within them.", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToPublish,
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
	{level: settings.LevelCampaign, label: "Insight session completed", help: "Yes once the customer insight session for this campaign has taken place.", typ: settings.TypeYesNo, marking: settings.MarkingOptional},
	{level: settings.LevelCampaign, label: "Workshop completed", help: "Yes once the campaign planning workshop has taken place.", typ: settings.TypeYesNo, marking: settings.MarkingOptional},
	{level: settings.LevelCampaign, label: "Global GM sign-off completed", help: "Yes once the global general manager has signed the campaign off.", typ: settings.TypeYesNo, marking: settings.MarkingOptional},

	{level: settings.LevelInitiative, label: "Growth ambition", help: "The growth ambitions this initiative contributes to.", typ: settings.TypeMultipleChoice, marking: settings.MarkingOptional,
		options: opts("LUBRICANTS – Premium growth", "LUBRICANTS – B2B expansion", "MOBILITY – Integrated offer", "MOBILITY – Convenience retail")},
	{level: settings.LevelInitiative, label: "Function", help: "The function that leads the initiative.", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		options: opts("Marketing", "Sales", "Customer service", "Digital")},
	{level: settings.LevelInitiative, label: "Focus area", help: "What the initiative mainly works on.", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		options: opts("Brand", "Customer acquisition", "Loyalty", "Product launch")},
	{level: settings.LevelInitiative, label: "FIT4 initiative", help: "Yes when the initiative is part of the FIT4 programme.", typ: settings.TypeYesNo, marking: settings.MarkingOptional},

	{level: settings.LevelActivity, label: "Channel / audience", help: "Who the activity reaches: external customers, Shell staff or partners.", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("External", "Internal", "Partners")},
	{level: settings.LevelActivity, label: "Country / OU", typ: settings.TypeMultipleChoiceSplit, marking: settings.MarkingNeededToSubmit,
		help:    "Split the activity's cost over countries; the shares must add up to 100 %.",
		options: opts("China", "India", "United Kingdom", "Netherlands", "Germany", "Singapore")},
	{level: settings.LevelActivity, label: "Company code", help: "The Shell company that pays for the activity.", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("GB01", "NL02", "CN04", "IN05", "DE03", "SG06")},
	{level: settings.LevelActivity, label: "Cost centre", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		help:    "Type part of the code to find it.",
		options: opts("CN04000220", "CN04000231", "GB01000105", "GB01000118", "NL02000340", "IN05000412", "DE03000150", "SG06000077")},
	{level: settings.LevelActivity, label: "Global campaign", help: "The global campaigns the activity supports.", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("Go Well integrated campaign", "Helix Ultra global platform", "Shell Recharge launch")},
	{level: settings.LevelActivity, label: "Growth ambition", help: "The growth ambitions this activity contributes to.", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("LUBRICANTS – Premium growth", "LUBRICANTS – B2B expansion", "MOBILITY – Integrated offer", "MOBILITY – Convenience retail")},
	{level: settings.LevelActivity, label: "PIR", help: "Yes when a post-implementation review is planned.", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("Yes", "No")},
	{level: settings.LevelActivity, label: "FIT4 initiative", help: "Whether the activity belongs to FIT4, and to which part.", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		options: opts("FIT4 – Cost", "FIT4 – Growth", "Not FIT4")},
	{level: settings.LevelActivity, label: "FIT4 ID", help: "The FIT4 reference of the activity, when it has one.", typ: settings.TypeText, marking: settings.MarkingNeededToSubmit},
	{level: settings.LevelActivity, label: "Zero-based cost hierarchy", help: "The zero-based budgeting cost groups the activity's spend falls into.", typ: settings.TypeMultipleChoice, marking: settings.MarkingOptional,
		options: opts("Media", "Agency", "Events", "Production")},
	{level: settings.LevelActivity, label: "Brands", help: "Split the activity's cost over the brands it promotes; the shares must add up to 100 %.", typ: settings.TypeMultipleChoiceSplit, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants, options: opts("Helix", "Rimula", "Spirax", "Tellus")},
	{level: settings.LevelActivity, label: "Global key accounts", help: "The global key accounts the activity targets.", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants,
		options: opts("Account A – Automotive OEM", "Account B – Mining", "Account C – Logistics")},
	{level: settings.LevelActivity, label: "Sector", help: "The customer sectors the activity targets.", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants, options: opts("Automotive", "Mining", "Construction", "Agriculture")},
	{level: settings.LevelActivity, label: "Sub-sector", help: "The sub-sectors within the chosen sectors.", typ: settings.TypeMultipleChoice, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants, options: opts("Passenger cars", "Heavy duty", "Off-highway")},
	{level: settings.LevelActivity, label: "Product launch", help: "Whether the activity launches or relaunches a product.", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
		section: lubricantsSection, condition: lubricants, options: opts("Yes – new product", "Yes – relaunch", "No")},
	{level: settings.LevelActivity, label: "Function (Retail)", help: "The Retail function that leads the activity.", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility, options: opts("Marketing", "Operations", "Category management")},
	{level: settings.LevelActivity, label: "Focus area (Retail)", help: "What the Retail activity mainly works on.", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility, options: opts("Fuel", "Convenience", "Car wash")},
	{level: settings.LevelActivity, label: "Function (Mobility)", help: "The Mobility function that leads the activity.", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility, options: opts("Marketing", "Fleet sales", "E-mobility")},
	{level: settings.LevelActivity, label: "Focus area (Mobility)", help: "What the Mobility activity mainly works on.", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility, options: opts("Fleet solutions", "EV charging", "Loyalty")},
	{level: settings.LevelActivity, label: "Innovation", help: "Describe what is new about the activity, if anything.", typ: settings.TypeLongText, marking: settings.MarkingOptional,
		section: mobilitySection, condition: mobility},

	{level: settings.LevelLineItem, label: "Spend type", help: "What kind of spend the line item is. GL codes are chosen within it.", typ: settings.TypeSingleChoice, marking: settings.MarkingNeededToSubmit,
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
	{level: settings.LevelLineItem, label: "Spend category", help: "Who the spend is for: one B2B customer, the mass market or internal.", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: financeSection, options: opts("B2B customer specific", "B2C mass market", "Internal")},
	{level: settings.LevelLineItem, label: "Contracted spend", help: "Whether the spend is under a contract, and for how long.", typ: settings.TypeSingleChoice, marking: settings.MarkingOptional,
		section: financeSection, options: opts("Yes – long term", "Yes – short term", "No")},
}

// addSettings creates the Settings of 002 data model §13 through the settings repository, in this order
// (so the IDs are stable): currencies, activity types, models, finance budgets with their items, Shell
// attributes with their options. It keeps their IDs by name in d.settings.
func (d *demo) addSettings(ctx context.Context) error {
	repo := d.stores.Settings
	d.settings = SettingsSeed{
		Models: map[string]string{}, ActivityTypes: map[string]string{}, Attributes: map[string]string{},
		Options: map[string]string{}, Budgets: map[string]string{}, Items: map[string]string{},
	}
	for _, step := range []func(context.Context, stores.SettingsStore) error{
		d.addCurrencies, d.addActivityTypes, d.addModels, d.addFinanceBudgets, d.addAttributes,
	} {
		if err := step(ctx, repo); err != nil {
			return err
		}
	}
	return nil
}

// The simulated GSAP rate history of 016 data-model §8: one import, RI-1001, of 31 daily rates per currency.
const (
	seedRateFile = "gsap_rates_seed.csv"
	seedRateDays = 30 // the days of rates before the seed day
)

// seedRateSwing is how the rates of the days before the seed day differ from the seeded rate, in thousandths,
// cycling from the oldest day: −0.6 %, −0.3 %, 0, +0.3 %, +0.6 %.
var seedRateSwing = []int64{-6, -3, 0, 3, 6}
