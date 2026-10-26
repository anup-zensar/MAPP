package financials

import (
	"context"
	"strings"

	"mapp/internal/platform/actor"
)

// TemplateName is the name of the sample file (FR-009).
const TemplateName = "mapp-gsap-extract-sample.csv"

// sampleRows are the three example rows of the sample file (FR-009): a PO for the HX key HX1001, an invoice
// for that PO, and an accrual of September 2026. Every value is fictitious.
var sampleRows = []string{
	"PO,4500009001,10,,,2026-01-15,,10000.00,USD,V-200001,Example Agency Ltd,Example purchase order,,HX1001,",
	"INVOICE,5100009001,,4500009001,10,2026-02-16,,4000.00,USD,V-200001,Example Agency Ltd,Example invoice,,,",
	"ACCRUAL,9100009001,,4500009001,10,2026-09-30,2026-09,1500.00,USD,V-200001,Example Agency Ltd,Example accrual,,,",
}

// Template returns the sample file of the simulated GSAP extract (FR-009): the header row with every column
// in the documented order, and three example rows. Only people who may import get it (FR-028).
func (s *Service) Template(ctx context.Context, a actor.Actor) ([]byte, error) {
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		return s.access.Import(ctx, a)
	})
	if err != nil {
		return nil, err
	}
	lines := append([]string{strings.Join(csvColumns, ",")}, sampleRows...)
	return []byte(strings.Join(lines, "\r\n") + "\r\n"), nil
}
