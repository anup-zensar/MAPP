package financials

import (
	"context"

	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
)

// DemoFeed is the demo data of FR-031 as a feed: the 11 fictitious GSAP transactions of data model §6, for
// the seed. Vendors, document numbers and SWBS codes are invented. They go through the same apply code as an
// import, so the demo shows exactly what the matching rules do.
type DemoFeed struct{}

// Name is the feed's name on the screens.
func (DemoFeed) Name() string { return demoFeedName }

// Source says that this is the demo data.
func (DemoFeed) Source() Source { return SourceDemo }

// demoRow is one demo transaction, as text.
type demoRow struct {
	typ                                             Type
	document, line, poNumber, posting, accrualMonth string
	amount, currency                                string
	vendorCode, vendorName, description             string
	swbs, trackingKey                               string
}

// The demo vendors (fictitious).
const (
	northlight = "Northlight Events GmbH"
	keynote    = "Keynote Speakers Ltd"
	outdoor    = "Outdoor Frames Ltd"
	brightside = "Brightside Media BV"
)

// demoRows are the rows of data model §6 in file order: the POs first, so their IDs are FT-1001 to FT-1004.
var demoRows = []demoRow{
	{TypePurchaseOrder, "4500001001", "10", "", "2026-01-12", "", "15800.00", "GBP", "V-100101", northlight,
		"Venue hire Berlin", "DE03-M26-0003", "HX1003"},
	{TypePurchaseOrder, "4500001002", "10", "", "2026-05-04", "", "5000.00", "USD", "V-100102", keynote,
		"Keynote speakers", "", "HX1004"},
	{TypePurchaseOrder, "4500001003", "10", "", "2026-01-08", "", "9800.00", "EUR", "V-100103", outdoor,
		"Stand build", "GB01-M26-0006", "HX1006"},
	{TypePurchaseOrder, "4500001004", "10", "", "2026-09-15", "", "12000.00", "USD", "V-100104", brightside,
		"Retail display production", "", "HX1099"},
	{TypeInvoice, "5100002001", "", "4500001001", "2026-02-20", "", "15800.00", "GBP", "V-100101", northlight,
		"Venue hire Berlin", "", ""},
	{TypeInvoice, "5100002002", "", "4500001002", "2026-09-10", "", "2500.00", "USD", "V-100102", keynote,
		"Keynote speakers, first half", "", ""},
	{TypeInvoice, "5100002003", "", "4500001003", "2026-03-02", "", "9800.00", "EUR", "V-100103", outdoor,
		"Stand build", "GB01-M26-0006", ""},
	{TypePayment, "1500003001", "", "4500001001", "2026-03-20", "", "15800.00", "GBP", "V-100101", northlight,
		"Venue hire Berlin", "", ""},
	{TypePayment, "1500003002", "", "", "2026-04-01", "", "9800.00", "EUR", "V-100103", outdoor,
		"Stand build", "GB01-M26-0006", ""},
	{TypeAccrual, "9100004001", "", "4500001002", "2026-08-31", "2026-08", "5000.00", "USD", "V-100102", keynote,
		"Keynote speakers, August accrual", "", ""},
	{TypeAccrual, "9100004002", "", "4500001002", "2026-09-30", "2026-09", "2500.00", "USD", "V-100102", keynote,
		"Keynote speakers, September accrual", "", ""},
}

// Read returns the demo rows, numbered 2 to 12 like the rows of a file. The delivery is not used.
func (DemoFeed) Read(context.Context, Delivery) ([]Incoming, []RowProblem, []string, error) {
	rows := make([]Incoming, 0, len(demoRows))
	for i, r := range demoRows {
		amount, err := money.ParseAmount(r.amount)
		if err != nil {
			return nil, nil, nil, err
		}
		posting, err := date.Parse(r.posting)
		if err != nil {
			return nil, nil, nil, err
		}
		in := Incoming{Row: i + 2, Fields: Fields{
			Type: r.typ, DocumentNumber: r.document, LineNumber: r.line, PONumber: r.poNumber, PostingDate: posting,
			AccrualMonth: r.accrualMonth, Amount: amount, Currency: r.currency, VendorCode: r.vendorCode,
			VendorName: r.vendorName, Description: r.description, SwbsCode: r.swbs, TrackingKey: r.trackingKey,
		}}
		if r.typ == TypePurchaseOrder {
			in.PONumber = r.document
		}
		rows = append(rows, in)
	}
	return rows, []RowProblem{}, []string{}, nil
}
