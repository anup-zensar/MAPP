package money

import "slices"

// Display converts amounts into a person's display currency (feature 003, D-061): usd × rate, rounded
// half-up to the cent. Features work every sum out in USD first and convert each amount they answer once,
// so a total can differ by a cent from the sum of its converted parts.
//
// Inside a plan, amounts are in the plan currency (016 D-261). A Display given the current rates with
// WithRates converts an amount of any currency it has a rate for — Convert(amount, its rate, the display
// rate), once — and leaves an amount already in the display currency as it is.
type Display struct {
	currency string
	rate     Rate
	rates    Rates // the current rates, for amounts in other currencies than USD; nil = USD amounts only
}

// DisplayUSD shows amounts as they are, in US dollars.
var DisplayUSD = Display{currency: BaseCurrency, rate: One}

// DisplayIn returns the Display of the currency code with its rate in units per 1 USD as decimal text
// ("0.92"). An empty code, USD or a rate that is not above 0 shows US dollars.
func DisplayIn(code, rate string) Display {
	r, err := ParseRate(rate)
	if err != nil || code == "" || code == BaseCurrency {
		return DisplayUSD
	}
	return Display{currency: code, rate: r}
}

// WithRates returns d converting amounts of every currency in rs too (the plan currencies, 016 FR-013).
func (d Display) WithRates(rs Rates) Display {
	d.rates = rs
	return d
}

// Currency is the ISO code amounts are shown in.
func (d Display) Currency() string {
	if d.currency == "" {
		return BaseCurrency
	}
	return d.currency
}

// Money converts m: an amount in USD (or without a currency) as usd × rate; an amount in the display
// currency unchanged; an amount in another currency with the rates of WithRates. An amount whose currency
// has no rate stays as it is, in its own currency, so it is never labelled wrongly.
func (d Display) Money(m Money) Money {
	from, to, ok := d.between(m.Currency)
	if !ok {
		return m
	}
	return Money{Amount: Convert(m.Amount, from, to), Currency: d.Currency()}
}

// Parts converts ms, the parts of one total in one currency, so that they add up to their sum converted
// once (ConvertParts, 038 FR-022): every part but the last as Money does, the last the remainder.
func (d Display) Parts(ms []Money) []Money {
	converted := slices.Clone(ms)
	if len(ms) == 0 {
		return converted
	}
	from, to, ok := d.between(ms[0].Currency)
	if !ok {
		return converted
	}
	amounts := make([]Amount, len(ms))
	for i, m := range ms {
		amounts[i] = m.Amount
	}
	for i, a := range ConvertParts(amounts, from, to) {
		converted[i] = Money{Amount: a, Currency: d.Currency()}
	}
	return converted
}

// between gives the rates that convert an amount in code into the display currency; false when it stays as
// it is (already in the display currency, or no usable rate).
func (d Display) between(code string) (from, to Rate, ok bool) {
	usd := d.currency == "" || d.currency == BaseCurrency
	to = One
	if !usd {
		to = d.rate
	}
	switch {
	case code == "" || code == BaseCurrency:
		return One, to, !usd
	case code == d.Currency():
		return to, to, false
	}
	from, found := d.rates[code]
	return from, to, found && from.micros > 0
}

// Optional converts m, or gives nil when there is no amount.
func (d Display) Optional(m *Money) *Money {
	if m == nil {
		return nil
	}
	converted := d.Money(*m)
	return &converted
}
