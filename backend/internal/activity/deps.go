package activity

import (
	"context"

	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// What activities reads from earlier features: small consumer-side interfaces (D-196). The repositories
// of 001, 004 and 005 satisfy PlanReader, HierarchyReader and PeopleReader directly; *settings.Service of
// feature 002 satisfies SettingsReader (its reader API, "How later features read Settings"). Every method
// runs inside the caller's transaction, which ctx carries.

// PlanReader finds plans (plan.Store, feature 004).
type PlanReader interface {
	Get(ctx context.Context, id string) (plan.Plan, bool, error)
}

// HierarchyReader finds campaigns and initiatives (campaign.Store, feature 005).
type HierarchyReader interface {
	GetCampaign(ctx context.Context, id string) (campaign.Campaign, bool, error)
	CampaignsOfPlan(ctx context.Context, planID string) ([]campaign.Campaign, error)
	GetInitiative(ctx context.Context, id string) (campaign.Initiative, bool, error)
	InitiativesOfCampaigns(ctx context.Context, campaignIDs []string) ([]campaign.Initiative, error)
}

// PeopleReader finds people (people.Store, features 001 and 003).
type PeopleReader interface {
	Get(ctx context.Context, id string) (people.Person, bool, error)
}

// SettingsReader is what activities reads from Settings (feature 002): the method names and signatures of
// 002's reader API, so that *settings.Service satisfies it.
type SettingsReader interface {
	// Model returns the model with id, and false when there is none.
	Model(ctx context.Context, id string) (settings.Model, bool, error)
	// ActivityType returns the activity type with id, and false when there is none.
	ActivityType(ctx context.Context, id string) (settings.ActivityType, bool, error)
	// ActivityTypes returns every activity type, active or not, sorted by name.
	ActivityTypes(ctx context.Context) ([]settings.ActivityType, error)
	// Attributes returns the Shell attributes of level, active and inactive, in display order.
	Attributes(ctx context.Context, level settings.Level) ([]settings.Attribute, error)
	// ListAttributes returns the Shell attributes of level as the API shows them, in display order.
	ListAttributes(ctx context.Context, a actor.Actor, level *settings.Level) ([]settings.ShellAttributeView, error)
	// Option returns the option with id, and false when there is none.
	Option(ctx context.Context, id string) (settings.Option, bool, error)
	// BusinessUnit returns the Business unit option ids among a plan's attribute values (empty when none).
	BusinessUnit(ctx context.Context, planValues settings.AttributeValues) ([]string, error)
	// CheckAttributes checks the attribute values of a save (problems, cleared values, missing ones and the
	// values to store).
	CheckAttributes(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error)
	// MissingAttributes lists the needed attributes that are still empty, or splits not totalling 100.00 %.
	MissingAttributes(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error)
	// DescribeAttribute gives the label and the text of an "attributes.<id>" field for the activity log.
	DescribeAttribute(ctx context.Context, field string, value any) (label, text string, err error)
	// Rates returns the current rate of every currency (016 FR-002): planned costs in a plan currency are shown
	// in the display currency, and the types' USD defaults converted into the plan currency, with them.
	Rates(ctx context.Context) (money.Rates, error)
}

// Deps are the readers of earlier features the activities service works with.
type Deps struct {
	Plans     PlanReader
	Hierarchy HierarchyReader
	Settings  SettingsReader
	People    PeopleReader
	Workflows Workflows // NewSeededWorkflows until feature 008 is wired
}

// noLineItems is the LineItems of a build without feature 007: no line items, nothing to copy or re-spread.
type noLineItems struct{}

func (noLineItems) Facts(context.Context, []string) (map[string]LineItemFacts, error) {
	return map[string]LineItemFacts{}, nil
}

func (noLineItems) CopyAll(context.Context, actor.Actor, string, string, int) (int, error) {
	return 0, nil
}

func (noLineItems) CopyMany(context.Context, actor.Actor, []CopyPair, int) (int, error) {
	return 0, nil
}

func (noLineItems) DatesChanged(context.Context, actor.Actor, Activity, Activity) error {
	return nil
}

func (noLineItems) Items(context.Context, string) ([]LineItemBrief, error) {
	return []LineItemBrief{}, nil
}

func (noLineItems) MoveToActivity(context.Context, actor.Actor, []string, string) error {
	return nil
}

// noApprovalHooks is the ApprovalHooks of a build without feature 008: nothing reacts.
type noApprovalHooks struct{}

func (noApprovalHooks) ApprovalRestarted(context.Context, actor.Actor, Activity, Activity, []history.Change) error {
	return nil
}

func (noApprovalHooks) ActivityDeleting(context.Context, actor.Actor, Activity) error {
	return nil
}

func (noApprovalHooks) ActivityRestored(context.Context, history.RestoreContext, string) ([]string, error) {
	return nil, nil
}

// The compiler checks here that the defaults implement the seams.
var (
	_ LineItems     = noLineItems{}
	_ ApprovalHooks = noApprovalHooks{}
)
