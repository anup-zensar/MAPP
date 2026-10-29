package financials

import (
	"context"
	"strings"

	"mapp/internal/platform/actor"
)

const TemplateName = "spa-one-gsap-extract-sample.csv"

// sampleRows are a PO, an invoice for it and an accrual of September 2026; every value is fictitious.
var sampleRows = []string{
	"PO,4500009001,10,,,2026-01-15,,10000.00,USD,V-200001,Example Agency Ltd,Example purchase order,,HX1001,",
	"INVOICE,5100009001,,4500009001,10,2026-02-16,,4000.00,USD,V-200001,Example Agency Ltd,Example invoice,,,",
	"ACCRUAL,9100009001,,4500009001,10,2026-09-30,2026-09,1500.00,USD,V-200001,Example Agency Ltd,Example accrual,,,",
}

// Template returns the sample GSAP extract file to people who may import.
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
