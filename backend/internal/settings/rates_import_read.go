package settings

import (
	"bytes"
	"cmp"
	"context"
	"encoding/csv"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// ListRateImports returns the latest limit imports, newest first (FR-007): 10 when limit is 0, at most 50.
func (s *Service) ListRateImports(ctx context.Context, a actor.Actor, limit int) (RateImportPage, error) {
	if limit <= 0 {
		limit = defaultRecentImports
	}
	limit = min(limit, maxRecentImports)
	page := RateImportPage{Items: []RateImportView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		imports, err := s.repo.ListRateImports(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(imports, func(x, y RateImport) int {
			return cmp.Or(y.CreatedAt.Compare(x.CreatedAt), compareIDs(y.ID, x.ID))
		})
		for _, i := range imports[:min(limit, len(imports))] {
			page.Items = append(page.Items, rateImportView(i))
		}
		page.Total = len(page.Items)
		return nil
	})
	return page, err
}

// ImportSample returns a sample GSAP rate file (FR-006, data-model §2.4): one M row per active currency other
// than USD, dated today, at its current rate × 1.005 rounded half up to six decimals, named
// gsap_rates_<today>.csv.
func (s *Service) ImportSample(ctx context.Context) (string, []byte, error) {
	var buffer bytes.Buffer
	day := s.today()
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		currencies, err := s.currenciesWithRates(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(currencies, compareCurrencies)
		w := csv.NewWriter(&buffer)
		if err := w.Write(gsapColumns); err != nil {
			return err
		}
		for _, c := range currencies {
			if !c.Active || c.Code == money.BaseCurrency || c.Rate.Micros() <= 0 {
				continue
			}
			sample := money.RateFromMicros((c.Rate.Micros()*1005 + 500) / 1000)
			if err := w.Write([]string{sampleRateType, money.BaseCurrency, c.Code, day.String(), sample.String()}); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	})
	return fmt.Sprintf(gsapSampleNamePattern, day.String()), buffer.Bytes(), err
}

// rateImportView returns the import i as the API answers it.
func rateImportView(i RateImport) RateImportView {
	return RateImportView{
		ID: i.ID, FileName: i.FileName, Feed: i.Feed, FirstDate: i.FirstDate, LastDate: i.LastDate, Added: i.Added,
		Replaced: i.Replaced, Unchanged: i.Unchanged, NotUsed: i.NotUsed, Problems: i.Problems,
		ImportedAt: i.CreatedAt, ImportedBy: i.CreatedBy,
	}
}
