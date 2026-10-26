// Package settings holds MAPP's Settings (feature 002): models, activity types, Shell attributes and
// their options, currencies with exchange rates, and finance budgets with their items. Later features read
// Settings only through this package's readers and attribute functions (002 plan, "How later features read
// Settings"), never with store.Get on its tables.
//
// The shared foundation wrote the data layer: the stored types (this file), the Shell attribute values
// kept on planning records (values.go), the API types (views.go), the usage hook (usage.go) and the
// repository (repository.go, store_json.go). Feature 002 adds the service, the readers, the history kinds
// and the handler.
package settings

import (
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// Level is the planning level a Shell attribute belongs to (data model §4).
type Level string

// The levels, top-down.
const (
	LevelPlan       Level = "plan"
	LevelCampaign   Level = "campaign"
	LevelInitiative Level = "initiative"
	LevelActivity   Level = "activity"
	LevelLineItem   Level = "lineItem"
)

// Levels lists the levels top-down.
var Levels = []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity, LevelLineItem}

// AttributeType is the kind of value a Shell attribute holds (data model §4, §9).
type AttributeType string

// The attribute types. The first three are the choice types: they have options.
const (
	TypeSingleChoice        AttributeType = "singleChoice"
	TypeMultipleChoice      AttributeType = "multipleChoice"
	TypeMultipleChoiceSplit AttributeType = "multipleChoiceSplit" // only at activity and line-item level (D-029)
	TypeYesNo               AttributeType = "yesNo"               // always optional; missing reads as No
	TypeText                AttributeType = "text"                // at most 120 characters
	TypeLongText            AttributeType = "longText"            // at most 2,000 characters
	TypeNumber              AttributeType = "number"              // a whole number, ±999,999,999,999
)

// Marking says whether an empty attribute blocks publishing (plan, campaign, initiative) or submitting
// (activity, line item).
type Marking string

// The markings.
const (
	MarkingOptional        Marking = "optional"
	MarkingNeededToPublish Marking = "neededToPublish" // plan, campaign and initiative only
	MarkingNeededToSubmit  Marking = "neededToSubmit"  // activity and line item only
)

// RoleBusinessUnit marks the plan-level Business unit attribute (Attribute.Role, D-055): conditions of other
// attributes refer to its options. The seed creates it; exactly one may exist.
const RoleBusinessUnit = "businessUnit"

// DefaultSection is the section of an attribute whose section was left empty (FR-019).
const DefaultSection = "Shell attributes"

// Model is the rulebook a plan uses (table "models", data model §2): its conversion rates, its average deal
// size and the activity types it allows.
type Model struct {
	ID              string        `json:"id"`              // "MOD-1001"
	Name            string        `json:"name"`            // 1–80 characters, unique among models
	InqToMqlRate    money.Percent `json:"inqToMqlRate"`    // above 0 and at most 100.00
	MqlToDealRate   money.Percent `json:"mqlToDealRate"`   // above 0 and at most 100.00
	AverageDealSize money.Money   `json:"averageDealSize"` // USD, above 0 and at most 100,000,000.00
	ActivityTypeIDs []string      `json:"activityTypeIds"` // at least one; in the order of the activity-type list (by name)
	Active          bool          `json:"active"`
	record.Meta
}

// Allows reports whether the model allows the activity type activityTypeID (R7).
func (m Model) Allows(activityTypeID string) bool {
	return slices.Contains(m.ActivityTypeIDs, activityTypeID)
}

// Goals are the goals the waterfall works out from a number of inquiries with a model (data model §2.1):
// MQL = INQ→MQL rate of INQ, deals = MQL→deal rate of MQL (both half-up to whole numbers), revenue = deals ×
// average deal size, in USD.
type Goals struct {
	INQ, MQL, Deals int64
	Revenue         money.Amount // USD
}

// ActivityType is a kind of marketing work (table "activity_types", data model §3).
type ActivityType struct {
	ID                 string       `json:"id"`                 // "AT-1001"
	Name               string       `json:"name"`               // 1–120 characters, unique among activity types
	DefaultPlannedCost *money.Money `json:"defaultPlannedCost"` // USD, 0.00 or more; nil when none
	Active             bool         `json:"active"`
	record.Meta
}

// Attribute is a Shell attribute: a configurable field of one level's form (table "shell_attributes",
// data model §4).
type Attribute struct {
	ID                string        `json:"id"`                // "SA-1001"
	Label             string        `json:"label"`             // 1–80 characters, unique within its level
	HelpText          string        `json:"helpText"`          // at most 300 characters; "" when none
	Level             Level         `json:"level"`             // locked once a record has a value
	Type              AttributeType `json:"type"`              // locked once a record has a value
	Marking           Marking       `json:"marking"`           // by level; yes/no is always optional
	Section           string        `json:"section"`           // 1–60 characters; DefaultSection when left empty
	Position          int           `json:"position"`          // ≥ 1; display order inside the level (data model §4.2)
	ParentAttributeID string        `json:"parentAttributeId"` // the attribute it depends on; "" when none
	Condition         []string      `json:"condition"`         // Business unit option ids; empty = always shown
	Role              string        `json:"role"`              // RoleBusinessUnit or ""
	Active            bool          `json:"active"`
	record.Meta
}

// IsChoice reports whether the attribute's type has options: single choice, multiple choice or multiple
// choice with % split (FR-018).
func (a Attribute) IsChoice() bool {
	return a.Type.IsChoice()
}

// IsChoice reports whether attributes of type t have options.
func (t AttributeType) IsChoice() bool {
	return t == TypeSingleChoice || t == TypeMultipleChoice || t == TypeMultipleChoiceSplit
}

// Option is one choice of a Shell attribute (table "shell_attribute_options", data model §5).
type Option struct {
	ID              string   `json:"id"`              // "SO-1001"
	AttributeID     string   `json:"attributeId"`     // the attribute it belongs to; never changes
	Label           string   `json:"label"`           // 1–120 characters, unique within its attribute
	Position        int      `json:"position"`        // ≥ 1; display order inside the attribute
	ParentOptionIDs []string `json:"parentOptionIds"` // options of the parent attribute it is linked to; empty when the attribute depends on nothing
	Active          bool     `json:"active"`
	record.Meta
}

// Currency is a currency (table "currencies", data model §6). Its id is its ISO code. USD always exists, is
// active and has rate 1.
//
// Since 016 a currency's rates are dated records of their own (ExchangeRate, D-258): the current rate is
// the latest one dated today or earlier. The stored fields Rate, RateChangedAt and RateChangedBy are legacy:
// MigrateRates turns them into the first dated rate once and clears them. The readers (Currency,
// Currencies) and the API fill them on read with the current rate and who recorded it, so every caller
// keeps reading c.Rate.
type Currency struct {
	Code          string      `json:"code"`                   // "EUR": three capital letters, unique, locked once saved
	Name          string      `json:"name"`                   // 1–80 characters, unique among currencies
	Symbol        string      `json:"symbol"`                 // at most 5 characters; "" when none
	Rate          money.Rate  `json:"rate,omitzero"`          // legacy when stored; the current rate when read
	RateChangedAt time.Time   `json:"rateChangedAt,omitzero"` // legacy when stored; when the current rate was recorded when read
	RateChangedBy actor.Actor `json:"rateChangedBy,omitzero"` // legacy when stored; who recorded it when read
	Active        bool        `json:"active"`
	record.Meta
}

// RateSource says where a dated exchange rate came from (contract RateSource).
type RateSource string

// The rate sources.
const (
	RateManual     RateSource = "manual"     // entered by hand (Add rate, Edit, the currency form)
	RateGSAPImport RateSource = "gsapImport" // written by a GSAP rate import
)

// ExchangeRate is one dated rate of a currency (table "exchange_rates", 016 data-model §2.1): units of the
// currency per 1 USD from its date on. At most one per currency and date; USD has none (always 1).
type ExchangeRate struct {
	ID       string     `json:"id"`       // "XR-1001"
	Currency string     `json:"currency"` // an existing currency other than USD; never changes
	Date     date.Date  `json:"date"`     // the day the rate applies from; never changes
	Rate     money.Rate `json:"rate"`     // above 0, at most six decimals, at most 1,000,000
	Source   RateSource `json:"source"`   // manual for Add rate and every Edit
	ImportID string     `json:"importId"` // the RI-… import that wrote it; "" for manual
	record.Meta
}

// RateImport is one GSAP rate file read into MAPP (table "rate_imports", 016 data-model §2.3). Never
// changed or deleted.
type RateImport struct {
	ID        string     `json:"id"`        // "RI-1001"
	FileName  string     `json:"fileName"`  // 1–200 characters, as uploaded
	Feed      string     `json:"feed"`      // the rate feed's name, "gsapDailyRatesFile"
	FirstDate *date.Date `json:"firstDate"` // the earliest date among the rates added or replaced; nil when none
	LastDate  *date.Date `json:"lastDate"`  // the latest such date
	Added     int        `json:"added"`
	Replaced  int        `json:"replaced"`
	Unchanged int        `json:"unchanged"`
	NotUsed   int        `json:"notUsed"`
	Problems  int        `json:"problems"`
	record.Meta
}

// FieldHelpText is the edited help text of one built-in field (table "field_help", 016 data-model §2.5): its
// id is the catalogue key, such as "plan.budget". Only edited texts are stored; Reset to default deletes the
// row.
type FieldHelpText struct {
	Key  string `json:"key"`
	Text string `json:"text"` // 0–300 characters, plain text; "" hides the ⓘ
	record.Meta
}

// FinanceBudget is the group of finance budget items with the same year and budget name (table
// "finance_budgets", data model §7, D-047). It is created with its first item and never deleted on its own.
type FinanceBudget struct {
	ID   string `json:"id"`   // "FB-1001"
	Year int    `json:"year"` // never changes
	Name string `json:"name"` // 1–120 characters, unique within its year
	record.Meta
}

// FinanceBudgetItem is one item of a finance budget, which line items link to (table
// "finance_budget_items", data model §8).
type FinanceBudgetItem struct {
	ID       string `json:"id"`       // "FBI-1001"
	BudgetID string `json:"budgetId"` // its finance budget; the item's year is the budget's
	Name     string `json:"name"`     // 1–120 characters, unique within its budget
	Code     string `json:"code"`     // at most 40 characters, unique within the year ignoring case; "" when none
	Active   bool   `json:"active"`
	record.Meta
}
