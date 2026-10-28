package purchaserequest

import (
	"context"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store keeps purchase requests with their lines in position order; lists are never nil and ordered by id.
// Get, Claims and All read whoever asks: the service checks the scope first, or answers only amounts.
type Store interface {
	Get(ctx context.Context, id string) (PurchaseRequest, bool, error)
	GetInScope(ctx context.Context, scope ViewScope, id string) (PurchaseRequest, bool, error)
	ListInScope(ctx context.Context, scope ViewScope, ids []string) ([]PurchaseRequest, error)
	OnLineItems(ctx context.Context, scope ViewScope, lineItemIDs []string) ([]PurchaseRequest, error)
	Page(ctx context.Context, list authz.Scope, q ListQuery) (Page, error)
	Claims(ctx context.Context, lineItemIDs []string) ([]Claim, error)
	All(ctx context.Context) ([]PurchaseRequest, error)
	Put(ctx context.Context, pr PurchaseRequest) error
	Delete(ctx context.Context, id string) error
	GetLine(ctx context.Context, id string) (Line, bool, error)
	PutLine(ctx context.Context, l Line) error
	DeleteLine(ctx context.Context, id string) error
	AddRouting(ctx context.Context, r Routing) error
	Routings(ctx context.Context, purchaseRequestID string) ([]Routing, error)
	Configuration(ctx context.Context) (Configuration, error)
	PutConfiguration(ctx context.Context, c Configuration) error
}

type ListQuery struct {
	Search             string
	Statuses           []Status
	TypeOptionIDs      []string
	TypeNotCompleted   bool
	TypeAttributeID    string
	Vendors            []string
	VendorNotCompleted bool
	OwnerIDs           []string
	CreatedFrom        *date.Date
	CreatedTo          *date.Date
	WaitingForGSAP     bool
	Sort               string
	Page, PageSize     int
	Rates              money.Rates
}

type Page struct {
	Items   []PurchaseRequest
	Total   int
	Vendors []VendorGroup
}

type VendorGroup struct {
	Vendor  string
	Count   int
	Amount  money.Amount
	POLines []POLineRef
}
