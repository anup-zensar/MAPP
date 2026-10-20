package campaigns

import (
	"context"

	"northstar/internal/people"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// Deps holds what campaigns needs from other features (005 plan, "Dependencies through small
// interfaces"; D-196). Each is a small interface declared here, the consumer side; internal/app fills it
// with the real packages, and service tests with fakes. Every method runs inside the caller's transaction,
// which its context carries.
type Deps struct {
	Plans      PlanSource   // feature 004
	People     PersonSource // features 001/003
	Settings   Settings     // feature 002
	Activities Activities   // feature 006; NoActivities{} until then (hooks.go)
}

// PlanFacts is a plan as campaigns needs it: what its campaigns show and are checked against.
type PlanFacts struct {
	ID, Name   string
	Year       int
	Status     string                   // "draft" or "published"
	OwnerID    *string                  // the default owner of a new campaign (R2); nil when none
	Budget     *money.Money             // USD; nil when none
	ModelID    string                   // the plan's model
	Attributes settings.AttributeValues // the plan's Shell attribute values, Business unit included (R9)
	Access     authz.PlanFacts          // what access decisions need: teams, owner, creator (feature 014)
}

// PlanSource reads plans (feature 004).
type PlanSource interface {
	// Plan returns the plan id, and false when there is none (deleted plans included).
	Plan(ctx context.Context, id string) (PlanFacts, bool, error)
	// PlansOfYear returns every plan of year, any status (moves and copies stay inside a year).
	PlansOfYear(ctx context.Context, year int) ([]PlanFacts, error)
}

// PersonSource reads people (features 001 and 003): the owners. people.Repository implements it as it is.
type PersonSource interface {
	// Get returns the person with id, and false when there is none.
	Get(ctx context.Context, id string) (people.Person, bool, error)
}

// Settings is what campaigns needs from Settings (feature 002): the Shell attribute rules. The method set
// is that of feature 002's *settings.Service (002 plan, "Forms and values"), so the service implements it as
// it is.
type Settings interface {
	// CheckAttributes checks the attribute values a form sends for level: its Problems block saving (fields
	// "attributes.<id>", such as a region that belongs to no chosen cluster, R8) and its Values are what to
	// store. Previous are the values the record had (inactive options already held stay allowed);
	// BusinessUnit is the plan's (R9).
	CheckAttributes(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error)
	// MissingAttributes lists the shown, active, needed-to-publish attributes of level that are empty, in
	// form order; a dependent attribute whose chosen parent options have no options set up is not missing
	// (D-034).
	MissingAttributes(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error)
	// BusinessUnit returns the plan's Business unit option ids from its attribute values (empty when none).
	BusinessUnit(ctx context.Context, planValues settings.AttributeValues) ([]string, error)
	// DescribeAttribute gives the label and readable text of a field "attributes.<id>" for the activity log;
	// "" and "" for other fields.
	DescribeAttribute(ctx context.Context, field string, value any) (label, text string, err error)
	// Model returns the model with id, and false when there is none (what a move to a plan with another
	// model changes, FR-024).
	Model(ctx context.Context, id string) (settings.Model, bool, error)
	// ActivityType returns the activity type with id, active or not, and false when there is none.
	ActivityType(ctx context.Context, id string) (settings.ActivityType, bool, error)
	// Attributes returns the Shell attributes of level, active and inactive, in display order (the attribute
	// groups a move hides or shows, R9).
	Attributes(ctx context.Context, level settings.Level) ([]settings.Attribute, error)
	// Option returns the Shell attribute option with id, active or not, and false when there is none.
	Option(ctx context.Context, id string) (settings.Option, bool, error)
}
