package money

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
	if m.Currency == "" || m.Currency == BaseCurrency {
		if d.currency == "" || d.currency == BaseCurrency {
			return m
		}
		return Money{Amount: FromUSD(m.Amount, d.rate), Currency: d.currency}
	}
	if m.Currency == d.Currency() {
		return m
	}
	from, found := d.rates[m.Currency]
	if !found || from.micros <= 0 {
		return m
	}
	to := One
	if d.currency != "" && d.currency != BaseCurrency {
		to = d.rate
	}
	return Money{Amount: Convert(m.Amount, from, to), Currency: d.Currency()}
}

// Optional converts m, or gives nil when there is no amount.
func (d Display) Optional(m *Money) *Money {
	if m == nil {
		return nil
	}
	converted := d.Money(*m)
	return &converted
}
