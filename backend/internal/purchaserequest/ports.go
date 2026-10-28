package purchaserequest

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

type Deps struct {
	LineItems  LineItems
	People     People
	Currencies Currencies
	Attributes Attributes
	Workflows  Workflows
	Invoices   Invoices
}

type LineItems interface {
	Facts(ctx context.Context, ids []string) (map[string]LineItemFacts, error)
	// Search returns at most limit live line items of the plans the scope covers whose name, id or HX key
	// contains text (any case), of planID when it is not "", ordered by id.
	Search(ctx context.Context, scope authz.Scope, q LineItemSearch) ([]LineItemFacts, error)
	Activity(ctx context.Context, activityID string) (ActivityLineItems, bool, error)
}

type LineItemSearch struct {
	Text, PlanID string
	Limit        int
}

type LineItemFacts struct {
	ID, Name, OwnerID, Currency     string
	PlannedCost                     money.Amount
	ActivityID, ActivityName        string
	ActivityStatus                  string
	Plan                            PlanFacts
	Access                          authz.ActivityFacts
	Path                            []errs.RecordRef
	SwbsCode, SwbsStatus            string
	CompanyCode, CostCentre, GLCode string
	BusinessUnit                    []string
	ActivityOverBy                  *money.Money // the activity's line items above its planned cost; nil when within
}

type PlanFacts struct {
	Access         authz.PlanFacts
	Name, Currency string
}

type ActivityLineItems struct {
	ID, Name  string
	Plan      PlanFacts
	Access    authz.ActivityFacts
	LineItems []LineItemFacts
}

type Person struct {
	people.PersonRef
	Email string
}

type People interface {
	People(ctx context.Context, ids []string) (map[string]Person, error)
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}

type Currencies interface {
	Currency(ctx context.Context, code string) (settings.Currency, bool, error)
	Rates(ctx context.Context) (money.Rates, error)
}

type Attributes interface {
	Check(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error)
	Missing(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error)
	// Roles returns, per role among roles, the texts of the values: a text as typed, the labels of chosen options.
	Roles(ctx context.Context, values settings.AttributeValues, roles []string) (map[string][]string, error)
	AttributeID(ctx context.Context, role string) (string, error)
	Describe(ctx context.Context, field string, value any) (label, text string, err error)
}

type Workflows interface {
	Workflow(ctx context.Context, id string) (WorkflowFacts, bool, error)
}

type WorkflowFacts struct {
	ID, Name, ObjectType string
	Active               bool
}

type Invoices interface {
	// InvoicedOn returns, per PO line, its invoices minus credit notes in USD at today's rates.
	InvoicedOn(ctx context.Context, lines []POLineRef) (map[POLineRef]money.Amount, error)
}

type POLineRef struct {
	PONumber, Line string
}

// Rounds is the approval engine as purchase requests use it (031); it joins the caller's Write.
type Rounds interface {
	Submit(ctx context.Context, a actor.Actor, s Submission) error
}

type Submission struct {
	Kind        RoundKind
	ID, Comment string
}

// GSAP answers a routed purchase request with GSAP's PR number and a PO line per line sent (brief §8.3, ADR-0016).
type GSAP interface {
	CreatePurchaseOrder(ctx context.Context, pr RoutedPR) (GSAPOrder, error)
}

type RoutedPR struct {
	ID, Name               string
	RequestedBy            actor.Actor
	ShortText              string
	VendorName, VendorCode string
	GSAPPRNumber, PONumber string
	LastPOLine             int
	Lines                  []RoutedLine
}

type RoutedLine struct {
	ID, LineItemID, HXKey, SwbsCode, CostCentre, GLCode, Currency string
	Amount                                                        money.Amount
	DeliveryDate                                                  date.Date
	POLine                                                        string
}

type GSAPOrder struct {
	Source       string
	GSAPPRNumber string
	GSAPPRDate   date.Date
	PONumber     string
	Lines        []OrderLine
}

type OrderLine struct {
	PRLineID, POLine, Status string
	Date                     date.Date
	Value                    money.Amount
}
