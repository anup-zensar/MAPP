package financials

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/race"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/record"
)

// Each measure is taken three times and the fastest counts, so test binaries running at once cannot fail it.
func TestImport_FiveThousandRows_SC004(t *testing.T) {
	if testing.Short() || race.Enabled {
		t.Skip("a timing test: it runs without -short and without the race detector, in CI's Timing tests step")
	}
	lines := []string{header}
	for i := range 2500 {
		po := fmt.Sprintf("46%08d", i)
		lines = append(lines,
			fmt.Sprintf("PO,%s,10,,,2026-10-01,,1000.00,EUR,V-1,Vendor %d,Stress,,HX%d,", po, i, 1003+i%3),
			fmt.Sprintf("INVOICE,52%08d,,%s,10,2026-10-02,,400.00,EUR,V-1,Vendor %d,Stress,,,", i, po, i))
	}
	req := file("stress.csv", lines...)
	elapsed := time.Hour
	for range 3 {
		f := newFixture(t)
		start := time.Now()
		result, err := f.service.Import(t.Context(), alex, req)
		elapsed = min(elapsed, time.Since(start))
		if err != nil {
			t.Fatalf("Import: %v", err)
		}
		if result.Import.Counts.New != 5000 || result.Import.Counts.NotLinked != 0 {
			t.Fatalf("counts = %+v", result.Import.Counts)
		}
	}
	if elapsed > 5*time.Second {
		t.Errorf("a 5,000-row import took %s, want under 5 seconds", elapsed)
	}
	t.Logf("5,000-row import in %s", elapsed)
}

// memRepository keeps records decoded, so the test measures the service's own work, not the fake store's
// decoding of every record.
type memRepository struct {
	transactions []Transaction
	imports      []Import
}

func (m *memRepository) ListTransactions(context.Context) ([]Transaction, error) {
	return slices.Clone(m.transactions), nil
}

func (m *memRepository) PaymentsOnLineItems(_ context.Context, lineItemIDs []string) ([]Transaction, error) {
	return slices.DeleteFunc(slices.Clone(m.transactions), func(t Transaction) bool {
		return t.Type != TypePayment || !slices.Contains(lineItemIDs, t.LineItemID)
	}), nil
}

func (m *memRepository) GetTransaction(_ context.Context, id string) (Transaction, bool, error) {
	i := slices.IndexFunc(m.transactions, func(t Transaction) bool { return t.ID == id })
	if i < 0 {
		return Transaction{}, false, nil
	}
	return m.transactions[i], true, nil
}

func (m *memRepository) AddTransactions(ctx context.Context, list []Transaction) error {
	for _, t := range list {
		if err := m.PutTransaction(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (m *memRepository) PutTransaction(_ context.Context, t Transaction) error {
	if i := slices.IndexFunc(m.transactions, func(x Transaction) bool { return x.ID == t.ID }); i >= 0 {
		m.transactions[i] = t
		return nil
	}
	m.transactions = append(m.transactions, t)
	return nil
}

func (m *memRepository) DeleteTransaction(_ context.Context, id string) error {
	m.transactions = slices.DeleteFunc(m.transactions, func(t Transaction) bool { return t.ID == id })
	return nil
}

func (m *memRepository) ListImports(context.Context) ([]Import, error) {
	return slices.Clone(m.imports), nil
}

func (m *memRepository) GetImport(_ context.Context, id string) (Import, bool, error) {
	i := slices.IndexFunc(m.imports, func(x Import) bool { return x.ID == id })
	if i < 0 {
		return Import{}, false, nil
	}
	return m.imports[i], true, nil
}

func (m *memRepository) PutImport(_ context.Context, i Import) error {
	m.imports = append(m.imports, i)
	return nil
}

func (m *memRepository) DeleteImport(_ context.Context, id string) error {
	m.imports = slices.DeleteFunc(m.imports, func(i Import) bool { return i.ID == id })
	return nil
}

func TestTransactions_TwentyThousand_SC004(t *testing.T) {
	if testing.Short() || race.Enabled {
		t.Skip("a timing test: it runs without -short and without the race detector, in CI's Timing tests step")
	}
	f := newEmptyFixture(t)
	repo := &memRepository{}
	f.service.repo = repo
	lineItems := []string{"L-1003", "L-1004", "L-1006", "L-1007"}
	f.write(t, func(ctx context.Context) error {
		for i := range 20000 {
			tr := Transaction{
				ID: fmt.Sprintf("FT-%d", 10001+i), ImportID: "FIM-1001", Meta: record.NewMeta(actor.System, start),
				Fields: Fields{Type: Types[i%4], DocumentNumber: fmt.Sprintf("47%08d", i), PostingDate: date.MustParse("2026-06-01"),
					Amount: amountOf(t, "125.50"), Currency: []string{"USD", "EUR", "GBP"}[i%3], VendorName: fmt.Sprintf("Vendor %d", i%50)},
			}
			if tr.Type == TypeAccrual {
				tr.AccrualMonth = "2026-09"
			}
			if i%10 == 0 {
				tr.NotLinkedReason = reasonNoReference
			} else {
				tr.LineItemID, tr.MatchedBy = lineItems[i%4], MatchedByTrackingKey
			}
			repo.transactions = append(repo.transactions, tr)
		}
		return nil
	})
	q := Query{Type: filterAll, Search: "vendor 4", Link: filterAll, Sort: sortAmount, Desc: true, Page: 1, PageSize: 200}
	elapsed := time.Hour
	var page TransactionPage
	for range 3 {
		start := time.Now()
		var err error
		page, err = f.service.Transactions(t.Context(), alex, q)
		elapsed = min(elapsed, time.Since(start))
		if err != nil {
			t.Fatalf("Transactions: %v", err)
		}
	}
	if page.Counts.All == 0 || len(page.Items) != 200 {
		t.Errorf("counts %+v, %d items", page.Counts, len(page.Items))
	}
	if elapsed > time.Second {
		t.Errorf("a page of 20,000 transactions took %s, want under 1 second", elapsed)
	}
	t.Logf("20,000 transactions: %d matches in %s", page.Total, elapsed)
}
