package plans

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Plans reads Settings (feature 002) and people and teams (feature 003) through the two small interfaces
// below (D-073, D-196): plans declares what it needs, internal/app wires what implements it. The methods have
// the names and signatures of the readers those features publish (002 plan "How later features read
// Settings", 003 plan "Lookup functions for other features"), so their services can satisfy them directly.
// Every method joins the caller's transaction, which ctx carries, and takes no actor: the plans service
// already holds it.

// Settings is what plans reads from Settings.
type Settings interface {
	// Model returns the model with id, active or not, and false when there is none.
	Model(ctx context.Context, id string) (settings.Model, bool, error)
	// Models returns every model, active or not, sorted by name.
	Models(ctx context.Context) ([]settings.Model, error)
	// FinanceBudget returns the finance budget with id, and false when there is none.
	FinanceBudget(ctx context.Context, id string) (settings.FinanceBudget, bool, error)
	// Attributes returns the Shell attributes of level in display order, active and inactive.
	Attributes(ctx context.Context, level settings.Level) ([]settings.Attribute, error)
	// Option returns the Shell attribute option with id, and false when there is none.
	Option(ctx context.Context, id string) (settings.Option, bool, error)
	// CheckAttributes checks the Shell attribute values of a form of level: their problems (fields
	// "attributes.<id>"), what is still missing and the values to store (002 data model §9).
	CheckAttributes(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error)
	// MissingAttributes lists the needed attributes of level that are active, shown and empty, in display
	// order. At plan level the values hold the business unit, and businessUnit is ignored.
	MissingAttributes(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error)
	// BusinessUnit returns the option ids of the plan's Business unit value; empty when none is chosen.
	BusinessUnit(ctx context.Context, planValues settings.AttributeValues) ([]string, error)
	// DescribeAttribute gives the label of a field "attributes.<id>" and the text of its value, for the
	// activity log; "", "" for other fields.
	DescribeAttribute(ctx context.Context, field string, value any) (label, text string, err error)
	// Currency returns the currency with the ISO code, active or not, and false when there is none (016).
	Currency(ctx context.Context, code string) (settings.Currency, bool, error)
	// Rates returns the current rate of every currency, active or not (016 FR-002).
	Rates(ctx context.Context) (money.Rates, error)
}

// Directory is what plans reads from people and teams.
type Directory interface {
	// PersonRefs returns the people with the ids, as records show them; unknown ids are left out.
	PersonRefs(ctx context.Context, ids []string) (map[string]people.PersonRef, error)
	// TeamRefs returns the teams with the ids, as records show them; unknown ids are left out.
	TeamRefs(ctx context.Context, ids []string) (map[string]people.TeamRef, error)
	// MemberIDs returns the ids of each team's members, active and inactive people.
	MemberIDs(ctx context.Context, teamIDs []string) (map[string][]string, error)
	// TeamIDsOf returns the ids of the teams the person belongs to, active and inactive.
	TeamIDsOf(ctx context.Context, personID string) ([]string, error)
	// DisplayRateOf returns the person's display currency and its rate in units per 1 USD (USD and "1"
	// when the person chose none).
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}
