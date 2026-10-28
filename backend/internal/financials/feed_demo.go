package financials

import (
	"context"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// DemoFeed is the seed's 11 fictitious GSAP transactions (FR-031, data model §6) through the import's apply code;
// their SWBS codes are those the simulated GSAP gives Venue hire and Stand build (feature 030).
type DemoFeed struct{}

func (DemoFeed) Name() string { return demoFeedName }

func (DemoFeed) Source() Source { return SourceDemo }

type demoRow struct {
	typ                                             Type
	document, line, poNumber, posting, accrualMonth string
	amount, currency                                string
	vendorCode, vendorName, description             string
	swbs, trackingKey                               string
}

const (
	northlight = "Northlight Events GmbH"
	keynote    = "Keynote Speakers Ltd"
	outdoor    = "Outdoor Frames Ltd"
	brightside = "Brightside Media BV"

	venueHireBerlin = "Venue hire Berlin"
	standBuild      = "Stand build"
	venueHireSwbs   = "SIM-DE03-0001"
	standBuildSwbs  = "SIM-GB01-0001"
)

// demoRows are in file order: the POs first, so their IDs are FT-1001 to FT-1004.
var demoRows = []demoRow{
	{TypePurchaseOrder, "4500001001", "10", "", "2026-01-12", "", "15800.00", "GBP", "V-100101", northlight,
		venueHireBerlin, venueHireSwbs, "HX1003"},
	{TypePurchaseOrder, "4500001002", "10", "", "2026-05-04", "", "5000.00", "USD", "V-100102", keynote,
		"Keynote speakers", "", "HX1004"},
	{TypePurchaseOrder, "4500001003", "10", "", "2026-01-08", "", "9800.00", "EUR", "V-100103", outdoor,
		standBuild, standBuildSwbs, "HX1006"},
	{TypePurchaseOrder, "4500001004", "10", "", "2026-09-15", "", "12000.00", "USD", "V-100104", brightside,
		"Retail display production", "", "HX1099"},
	{TypeInvoice, "5100002001", "", "4500001001", "2026-02-20", "", "15800.00", "GBP", "V-100101", northlight,
		venueHireBerlin, "", ""},
	{TypeInvoice, "5100002002", "", "4500001002", "2026-09-10", "", "2500.00", "USD", "V-100102", keynote,
		"Keynote speakers, first half", "", ""},
	{TypeInvoice, "5100002003", "", "4500001003", "2026-03-02", "", "9800.00", "EUR", "V-100103", outdoor,
		standBuild, standBuildSwbs, ""},
	{TypePayment, "1500003001", "", "4500001001", "2026-03-20", "", "15800.00", "GBP", "V-100101", northlight,
		venueHireBerlin, "", ""},
	{TypePayment, "1500003002", "", "", "2026-04-01", "", "9800.00", "EUR", "V-100103", outdoor,
		standBuild, standBuildSwbs, ""},
	{TypeAccrual, "9100004001", "", "4500001002", "2026-08-31", "2026-08", "5000.00", "USD", "V-100102", keynote,
		"Keynote speakers, August accrual", "", ""},
	{TypeAccrual, "9100004002", "", "4500001002", "2026-09-30", "2026-09", "2500.00", "USD", "V-100102", keynote,
		"Keynote speakers, September accrual", "", ""},
}

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
