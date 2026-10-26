package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The history hooks of feature 016's Settings records: exchange rates (under their currency, restorable),
// rate imports and Field help texts (logged only).

// errNeverRemoved is the Remove of a kind whose records never go to the recycle bin; only a mistake in the
// code can call it.
var errNeverRemoved = errors.New("settings: rate imports and field help texts are never deleted")

// neverRemoved is the Remove of the log-only kinds.
func neverRemoved(context.Context, string) error {
	return errNeverRemoved
}

// rateSnapshot reads the rate id: named "EUR on 19 Oct 2026", under its currency.
func (s *Service) rateSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	r, found, err := s.repo.GetRate(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(r, r.ID, rateName(r), r.Currency)
}

// rateIDs lists the ids of the rates of the currency code: deleting a currency takes its rate history with it.
func (s *Service) rateIDs(ctx context.Context, code string) ([]string, error) {
	rates, err := s.repo.RatesOf(ctx, code)
	if err != nil {
		return nil, err
	}
	rateIDs := make([]string, 0, len(rates))
	for _, r := range rates {
		rateIDs = append(rateIDs, r.ID)
	}
	return rateIDs, nil
}

// describeRate gives the labels and texts of a rate's log entries.
func describeRate(_ context.Context, field string, value any) (string, string, error) {
	switch field {
	case "rate":
		text, _ := value.(string)
		if rate, err := money.ParseRate(text); err == nil {
			return "Rate (units per 1 USD)", rate.Format(), nil
		}
		return "Rate (units per 1 USD)", "", nil
	case "source":
		text, _ := value.(string)
		return "Source", rateSourceWords[RateSource(text)], nil
	}
	return "", "", nil
}

// rateSourceWords are the sources as people read them.
var rateSourceWords = map[RateSource]string{RateManual: "Entered by hand", RateGSAPImport: "GSAP import"}

// prepareRateRestore refuses a rate whose currency has meanwhile got another rate on the same day (data-model
// §2.1): "EUR already has a rate on 19 Oct 2026." Its currency must be stored (history checks the parent).
func (s *Service) prepareRateRestore(ctx context.Context, _ history.RestoreContext, snap *history.Snapshot) ([]string, error) {
	var r ExchangeRate
	if err := json.Unmarshal(snap.Data, &r); err != nil {
		return nil, err
	}
	book, err := s.rateBook(ctx)
	if err != nil {
		return nil, err
	}
	if other, taken := book.onDate(r.Currency, r.Date); taken && other.ID != r.ID {
		return nil, errs.CannotRestore(fmt.Sprintf(rateDateTakenMessage, r.Currency, date.Format(r.Date)))
	}
	return nil, nil
}

// rateImportSnapshot reads the rate import id, named by its file.
func (s *Service) rateImportSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	i, found, err := s.repo.GetRateImport(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(i, i.ID, i.FileName, "")
}

// fieldHelpSnapshot reads the edited help text of key, named "Plan › Plan budget".
func (s *Service) fieldHelpSnapshot(ctx context.Context, key string) (history.Snapshot, bool, error) {
	h, found, err := s.repo.GetFieldHelp(ctx, key)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	return snapshot(h, h.Key, fieldHelpName(key), "")
}

// describeFieldHelp gives the label of a help text's log entries.
func describeFieldHelp(_ context.Context, field string, _ any) (string, string, error) {
	if field == "text" {
		return "Help text", "", nil
	}
	return "", "", nil
}
