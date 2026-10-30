package money

import "slices"

// Display converts amounts into a display currency once per answered amount, usd × rate half-up to the cent
// (D-061), so a total can differ by a cent from its converted parts; WithRates adds plan currencies (016 D-261).
type Display struct {
	currency string
	rate     Rate
	rates    Rates
}

var DisplayUSD = Display{currency: BaseCurrency, rate: One}

// DisplayIn takes the rate per 1 USD as text ("0.92"); an empty code, USD or a rate not above 0 shows US dollars.
func DisplayIn(code, rate string) Display {
	r, err := ParseRate(rate)
	if err != nil || code == "" || code == BaseCurrency {
		return DisplayUSD
	}
	return Display{currency: code, rate: r}
}

func (d Display) WithRates(rs Rates) Display {
	d.rates = rs
	return d
}

func (d Display) Currency() string {
	if d.currency == "" {
		return BaseCurrency
	}
	return d.currency
}

// Money leaves an amount whose currency has no rate in its own currency, so it is never labelled wrongly.
func (d Display) Money(m Money) Money {
	from, to, ok := d.between(m.Currency)
	if !ok {
		return m
	}
	return Money{Amount: Convert(m.Amount, from, to), Currency: d.Currency()}
}

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

func (d Display) Optional(m *Money) *Money {
	if m == nil {
		return nil
	}
	converted := d.Money(*m)
	return &converted
}
