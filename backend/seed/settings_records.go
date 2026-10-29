package seed

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// addCurrencies creates the five currencies; a currency's ID is its code. Their rates are dated records since
// 016: addRateHistory writes them.
func (d *demo) addCurrencies(ctx context.Context, repo stores.SettingsStore) error {
	d.rates = money.Rates{}
	for _, c := range demoCurrencies {
		rate, err := money.ParseRate(c.rate)
		if err != nil {
			return fmt.Errorf("seed: rate of %s: %w", c.code, err)
		}
		d.rates[c.code] = rate
		currency := settings.Currency{Code: c.code, Name: c.name, Symbol: c.symbol, Active: true, Meta: d.meta()}
		if err := repo.PutCurrency(ctx, currency); err != nil {
			return err
		}
	}
	return d.addRateHistory(ctx, repo)
}

// addRateHistory writes the import RI-1001 "gsap_rates_seed.csv" by MAPP and, for every currency but USD,
// a GSAP-imported rate for each of the 30 days before the seed day (the seeded rate × (1 + swing), half up to
// six decimals) and one on the seed day equal to the seeded rate, so the current rates are the seeded ones
// (XR-1001 …, currency by currency, oldest first).
func (d *demo) addRateHistory(ctx context.Context, repo stores.SettingsStore) error {
	importID, err := d.next(ctx, ids.PrefixRateImport)
	if err != nil {
		return err
	}
	today := date.Today(d.at)
	first := today.AddDays(-seedRateDays)
	added := 0
	for _, c := range demoCurrencies {
		if c.code == money.BaseCurrency {
			continue
		}
		seeded := d.rates[c.code]
		for day := 0; day <= seedRateDays; day++ {
			rate := seeded
			if day < seedRateDays {
				swing := seedRateSwing[day%len(seedRateSwing)]
				rate = money.RateFromMicros((seeded.Micros()*(1000+swing) + 500) / 1000)
			}
			id, err := d.next(ctx, ids.PrefixExchangeRate)
			if err != nil {
				return err
			}
			r := settings.ExchangeRate{ID: id, Currency: c.code, Date: first.AddDays(day), Rate: rate,
				Source: settings.RateGSAPImport, ImportID: importID, Meta: d.meta()}
			if err := repo.PutRate(ctx, r); err != nil {
				return err
			}
			added++
		}
	}
	return repo.PutRateImport(ctx, settings.RateImport{
		ID: importID, FileName: seedRateFile, Feed: settings.GSAPRateFile{}.Name(), FirstDate: &first, LastDate: &today,
		Added: added, Meta: d.meta(),
	})
}

// addActivityTypes creates the six activity types, AT-1001 … AT-1006.
func (d *demo) addActivityTypes(ctx context.Context, repo stores.SettingsStore) error {
	for _, t := range demoActivityTypes {
		id, err := d.next(ctx, ids.PrefixActivityType)
		if err != nil {
			return err
		}
		activityType := settings.ActivityType{ID: id, Name: t.name, Active: true, Meta: d.meta()}
		if t.defaultPlannedCost != "" {
			cost, err := usd(t.defaultPlannedCost)
			if err != nil {
				return err
			}
			activityType.DefaultPlannedCost = &cost
		}
		if err := repo.PutActivityType(ctx, activityType); err != nil {
			return err
		}
		d.settings.ActivityTypes[t.name] = id
	}
	return nil
}

// addModels creates the two models, MOD-1001 and MOD-1002, with their allowed activity types in the order
// of the activity-type list (by name).
func (d *demo) addModels(ctx context.Context, repo stores.SettingsStore) error {
	for _, m := range demoModels {
		var r resolver
		typeNames := slices.Clone(m.activityTypes)
		slices.SortFunc(typeNames, func(a, b string) int { return cmp.Compare(strings.ToLower(a), strings.ToLower(b)) })
		typeIDs := make([]string, 0, len(typeNames))
		for _, name := range typeNames {
			typeIDs = append(typeIDs, r.id(d.settings.ActivityTypes, "activity type", name))
		}
		if r.err != nil {
			return r.err
		}
		inqToMql, err1 := money.ParsePercent(m.inqToMqlRate)
		mqlToDeal, err2 := money.ParsePercent(m.mqlToDealRate)
		dealSize, err3 := usd(m.averageDealSize)
		id, err4 := d.next(ctx, ids.PrefixModel)
		if err := cmp.Or(err1, err2, err3, err4); err != nil {
			return err
		}
		model := settings.Model{
			ID: id, Name: m.name, InqToMqlRate: inqToMql, MqlToDealRate: mqlToDeal, AverageDealSize: dealSize,
			ActivityTypeIDs: typeIDs, Active: true, Meta: d.meta(),
		}
		if err := repo.PutModel(ctx, model); err != nil {
			return err
		}
		d.settings.Models[m.name] = id
	}
	return nil
}

// addFinanceBudgets creates the 2026 finance budgets FB-1001 and FB-1002 with three items each, FBI-1001 … FBI-1006,
// and the 2027 one FB-1003 with FBI-1007 and FBI-1008 for the FY2027 line items (038 FR-061).
func (d *demo) addFinanceBudgets(ctx context.Context, repo stores.SettingsStore) error {
	for _, b := range demoBudgets {
		budgetID, err := d.next(ctx, ids.PrefixFinanceBudget)
		if err != nil {
			return err
		}
		budget := settings.FinanceBudget{ID: budgetID, Year: b.year, Name: b.name, Meta: d.meta()}
		if err := repo.PutFinanceBudget(ctx, budget); err != nil {
			return err
		}
		d.settings.Budgets[fmt.Sprintf("%d/%s", b.year, b.name)] = budgetID
		for _, item := range b.items {
			itemID, err := d.next(ctx, ids.PrefixFinanceBudgetItem)
			if err != nil {
				return err
			}
			stored := settings.FinanceBudgetItem{ID: itemID, BudgetID: budgetID, Name: item[0], Code: item[1], Active: true, Meta: d.meta()}
			if err := repo.PutFinanceBudgetItem(ctx, stored); err != nil {
				return err
			}
			d.settings.Items[fmt.Sprintf("%d/%s/%s", b.year, b.name, item[0])] = itemID
		}
	}
	return nil
}
