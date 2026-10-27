package settings

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// RatesOn returns the rate of every currency current on day d, active or not (D-275): the one way to
// convert at another day than today. USD is 1.
func (s *Service) RatesOn(ctx context.Context, d date.Date) (money.Rates, error) {
	rates := money.Rates{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		for _, c := range currencies {
			rates[c.Code] = withRate(c, book, d).Rate
		}
		return nil
	})
	return rates, err
}

// CurrentRates returns every currency's current rate with its date and source (FR-002, FR-009): USD first,
// then by code.
func (s *Service) CurrentRates(ctx context.Context) ([]RateInfo, error) {
	var infos []RateInfo
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(currencies, compareCurrencies)
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		files, err := s.importFileNames(ctx)
		if err != nil {
			return err
		}
		infos = make([]RateInfo, 0, len(currencies))
		for _, c := range currencies {
			infos = append(infos, rateInfoOf(c, book, s.today(), files))
		}
		return nil
	})
	return infos, err
}

// CurrentRate returns the current rate of the currency code with its date and source, and false when there
// is no such currency.
func (s *Service) CurrentRate(ctx context.Context, code string) (RateInfo, bool, error) {
	infos, err := s.CurrentRates(ctx)
	if err != nil {
		return RateInfo{}, false, err
	}
	for _, info := range infos {
		if info.Currency == code {
			return info, true, nil
		}
	}
	return RateInfo{}, false, nil
}

// ListCurrentRates answers GET /api/v1/exchange-rates/current. Every signed-in person may read it (014 Table C).
func (s *Service) ListCurrentRates(ctx context.Context, a actor.Actor) (CurrentRatePage, error) {
	infos, err := s.CurrentRates(ctx)
	if err != nil {
		return CurrentRatePage{}, err
	}
	return CurrentRatePage{Today: s.today(), Items: infos, Total: len(infos)}, nil
}

// rateInfoOf returns the current rate of c on day with where it came from.
func rateInfoOf(c Currency, book rateBook, day date.Date, files map[string]string) RateInfo {
	info := RateInfo{Currency: c.Code, Rate: money.One, InverseRate: money.One}
	if c.Code == money.BaseCurrency {
		return info
	}
	if r, found := book.on(c.Code, day); found {
		info.Rate, info.InverseRate = r.Rate, r.Rate.Inverse()
		rateDate, source := r.Date, r.Source
		info.RateDate, info.Source = &rateDate, &source
		if name, imported := files[r.ImportID]; imported && r.Source == RateGSAPImport {
			info.ImportFileName = &name
		}
		return info
	}
	if c.Rate.Micros() > 0 { // stored before 016 and not migrated yet
		rateDate, source := date.Today(c.RateChangedAt), RateManual
		info.Rate, info.InverseRate, info.RateDate, info.Source = c.Rate, c.Rate.Inverse(), &rateDate, &source
	}
	return info
}

// importFileNames returns the file name of every rate import, by id.
func (s *Service) importFileNames(ctx context.Context) (map[string]string, error) {
	imports, err := s.repo.ListRateImports(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(imports))
	for _, i := range imports {
		names[i.ID] = i.FileName
	}
	return names, nil
}

// rateView returns the rate r as the API answers it on day.
func rateView(r ExchangeRate, book rateBook, day date.Date, files map[string]string) ExchangeRateView {
	view := ExchangeRateView{
		ID: r.ID, Currency: r.Currency, Date: r.Date, Rate: r.Rate, InverseRate: r.Rate.Inverse(), Source: r.Source,
		Future: r.Date.After(day), Meta: r.Meta,
	}
	if current, found := book.on(r.Currency, day); found && current.ID == r.ID {
		view.Current = true
	}
	if r.ImportID != "" {
		importID := r.ImportID
		view.ImportID = &importID
		if name, found := files[r.ImportID]; found {
			view.ImportFileName = &name
		}
	}
	return view
}

// rateName is how the activity log and the recycle bin name a rate: "EUR on 19 Oct 2026".
func rateName(r ExchangeRate) string {
	return r.Currency + " on " + date.Format(r.Date)
}

// ListRates returns the rate history of the currency code, newest date first, with its current rate (FR-001,
// FR-003). USD has none. Every signed-in person may read it.
func (s *Service) ListRates(ctx context.Context, a actor.Actor, code string) (ExchangeRatePage, error) {
	var page ExchangeRatePage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.storedCurrency(ctx, code)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		files, err := s.importFileNames(ctx)
		if err != nil {
			return err
		}
		day := s.today()
		page = ExchangeRatePage{Items: []ExchangeRateView{}, Current: rateInfoOf(c, book, day, files)}
		rates := book[c.Code]
		for i := len(rates) - 1; i >= 0; i-- {
			page.Items = append(page.Items, rateView(rates[i], book, day, files))
		}
		page.Total = len(page.Items)
		return nil
	})
	return page, err
}

// rateViewOf returns the stored rate r as the API answers it, reading the other rates of its currency.
func (s *Service) rateViewOf(ctx context.Context, r ExchangeRate) (ExchangeRateView, error) {
	book, err := s.rateBook(ctx)
	if err != nil {
		return ExchangeRateView{}, err
	}
	files, err := s.importFileNames(ctx)
	if err != nil {
		return ExchangeRateView{}, err
	}
	return rateView(r, book, s.today(), files), nil
}

// storedRate reads the rate id, or answers not found.
func (s *Service) storedRate(ctx context.Context, id string) (ExchangeRate, error) {
	r, found, err := s.repo.GetRate(ctx, id)
	if err != nil {
		return ExchangeRate{}, err
	}
	if !found {
		return ExchangeRate{}, errs.NotFound(rateGoneMessage)
	}
	return r, nil
}
